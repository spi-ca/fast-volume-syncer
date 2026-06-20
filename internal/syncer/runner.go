// Package syncer prepares sandboxed mounts and runs the copier against resolved paths.
package syncer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// Runner optionally enters a sandbox, mounts both storage roots, and delegates copying.
type Runner struct {
	// Sandboxed decides whether the process should isolate its mount namespace first.
	Sandboxed bool

	// ReportEnabled logs filesystem state before and after the copy run.
	ReportEnabled bool
	// SandboxMountOption configures the sandbox/mount namespace behavior.
	SandboxMountOption string

	// SourceMountHost is the host portion of the source mount target.
	SourceMountHost string
	// SourceMountOptions are passed to the source mount syscall.
	SourceMountOptions string
	// SourceMountName is the workspace directory name used for the source mount.
	SourceMountName string

	// DestinationMountHost is the host portion of the destination mount target.
	DestinationMountHost string
	// DestinationMountOptions are passed to the destination mount syscall.
	DestinationMountOptions string
	// DestinationMountName is the workspace directory name used for the destination mount.
	DestinationMountName string

	// Copier performs the actual file transfer once the effective paths are ready.
	Copier copier.Runner

	// SourceMountPath is the source storage root to mount into the workspace.
	SourceMountPath string
	// SourceMountSubPath is the source subdirectory copied within the mounted root.
	SourceMountSubPath string

	// DestinationMountPath is the destination storage root to mount into the workspace.
	DestinationMountPath string
	// DestinationMountSubPath is the destination subdirectory copied within the mounted root.
	DestinationMountSubPath string
}

// Execute prepares mounts, optionally logs state, runs the copier, and always cleans up.
func (r *Runner) Execute(ctx context.Context) error {
	tempPath, srcPath, dstPath, err := r.prepareMountPoint()
	defer r.cleanupDirectory(tempPath)
	if err != nil {
		return err
	}

	if r.ReportEnabled {
		r.logVolumeInfo(srcPath)
		r.logVolumeInfo(dstPath)
	}

	if err = ensureNoSymlinkPath(srcPath); err != nil {
		return fmt.Errorf("unsafe source path before copy: %w", err)
	}
	if err = ensureNoSymlinkPath(dstPath); err != nil {
		return fmt.Errorf("unsafe destination path before copy: %w", err)
	}

	err = r.Copier.Execute(ctx, srcPath, dstPath)

	if r.ReportEnabled {
		r.logVolumeInfo(srcPath)
		r.logVolumeInfo(dstPath)
	}

	return err

}

// logOutput folds multiline command output into one log record with a shared header and prefix.
func (r *Runner) logOutput(reader io.Reader, header, prefix string) {
	scanner := bufio.NewScanner(reader)
	builder := strings.Builder{}
	builder.WriteString(header)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		builder.WriteString(prefix)
		builder.WriteString(line)
	}
	util.InfoLog.Print(builder.String())
}

// logVolumeInfo snapshots directory, mount, and filesystem state for sync reporting.
func (r *Runner) logVolumeInfo(path string) {
	r.runReportCommand("ls", []string{"-al", path}, fmt.Sprintf("directory_info(%s)=>", path), "\t")
	r.runReportCommand("findmnt", []string{"-T", path}, fmt.Sprintf("mount_info(%s)=>", path), "\t")
	r.runReportCommand("df", []string{"-h", path}, "fs            =>\t", "\t")
}

// runReportCommand resolves a diagnostic helper from the trusted PATH and logs its output.
func (r *Runner) runReportCommand(name string, args []string, header, prefix string) {
	binaryPath := util.LookupBinary(name)
	if binaryPath == "" {
		util.ErrLog.Printf("failed to find executable(%s)", name)
		return
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = util.TrustedChildEnvironment()
	if out, err := cmd.CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(%s): %v", name, err)
	} else {
		r.logOutput(bytes.NewReader(out), header, prefix)
	}
}

const (
	pinnedSourceMountName      = ".pinned-src"
	pinnedDestinationMountName = ".pinned-dst"
)

// prepareMountPoint optionally isolates the process, mounts both storage roots, pins subpaths, and returns copy paths.
func (r *Runner) prepareMountPoint() (string, string, string, error) {
	if r.Sandboxed {
		if err := sys.Sandbox(r.SandboxMountOption); err != nil {
			return "", "", "", fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}

	// After sandboxing, all mounts and temporary paths are created inside the isolated workspace.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("syncer-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to make temp directory: %w", err)
	}

	util.InfoLog.Printf("created temporary directory: '%s'", tempDir)

	if err = os.Chdir(tempDir); err != nil {
		return tempDir, "", "", fmt.Errorf("failed to change directory(%s): %w", tempDir, err)
	}

	// Build mount roots under the workspace, then resolve the effective copy subpaths inside them.
	srcMountPath, err := safeWorkspacePath(tempDir, r.SourceMountName)
	if err != nil {
		return tempDir, "", "", fmt.Errorf("invalid source mount name: %w", err)
	}
	dstMountPath, err := safeWorkspacePath(tempDir, r.DestinationMountName)
	if err != nil {
		return tempDir, "", "", fmt.Errorf("invalid destination mount name: %w", err)
	}
	srcMountInfo := returns.MountInfo{
		Host:    r.SourceMountHost,
		Path:    r.SourceMountPath,
		Options: r.SourceMountOptions,
	}
	dstMountInfo := returns.MountInfo{
		Host:    r.DestinationMountHost,
		Path:    r.DestinationMountPath,
		Options: r.DestinationMountOptions,
	}

	if err = sys.Mount(srcMountInfo.Source(), srcMountPath, srcMountInfo.Type(), srcMountInfo.RefinedOptions()); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", srcMountInfo, srcMountPath, err)
	}
	util.InfoLog.Printf("source mount success!(%s %s)", srcMountInfo, srcMountPath)

	if err = sys.Mount(dstMountInfo.Source(), dstMountPath, dstMountInfo.Type(), dstMountInfo.RefinedOptions()); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", dstMountInfo, dstMountPath, err)
	}
	util.InfoLog.Printf("destination mount success!(%s %s)", dstMountInfo, dstMountPath)

	srcMountSubPath, err := r.pinSubpathMount(tempDir, pinnedSourceMountName, srcMountPath, r.SourceMountSubPath, false)
	if err != nil {
		return tempDir, "", "", fmt.Errorf("invalid source subpath: %w", err)
	}
	dstMountSubPath, err := r.pinSubpathMount(tempDir, pinnedDestinationMountName, dstMountPath, r.DestinationMountSubPath, true)
	if err != nil {
		return tempDir, "", "", fmt.Errorf("invalid destination subpath: %w", err)
	}

	return tempDir, srcMountSubPath, dstMountSubPath, nil
}

// pinSubpathMount opens a mounted subdirectory by fd and bind-mounts that stable directory for copier use.
func (r *Runner) pinSubpathMount(tempDir, pinName, mountRoot, subpath string, create bool) (string, error) {
	pinPath, err := safeWorkspacePath(tempDir, pinName)
	if err != nil {
		return "", err
	}
	pinnedDir, err := sys.OpenDirBeneath(mountRoot, subpath, create)
	if err != nil {
		return "", err
	}
	defer pinnedDir.Close()
	if create {
		if err := sys.ChmodFD(pinnedDir.Fd(), r.Copier.FileMode.Perm()|0o700); err != nil {
			return "", err
		}
	}
	if err := sys.BindMountFD(pinnedDir.Fd(), pinPath); err != nil {
		return "", err
	}
	return pinPath, nil
}

// safeWorkspacePath joins a caller-controlled relative path without allowing escape from the workspace root.
func safeWorkspacePath(root, subpath string) (string, error) {
	if subpath == "" || subpath == "." {
		return root, nil
	}
	if !filepath.IsLocal(subpath) {
		return "", fmt.Errorf("path %q must be relative and stay within the mount root", subpath)
	}
	cleanRoot := filepath.Clean(root)
	cleanSubpath := filepath.Clean(subpath)
	joined := filepath.Join(cleanRoot, cleanSubpath)
	current := cleanRoot
	for _, part := range strings.Split(cleanSubpath, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("inspect path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path %q must not traverse symlink component %q", subpath, current)
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return "", fmt.Errorf("resolve path %q: %w", current, err)
		}
		rel, err := filepath.Rel(cleanRoot, resolved)
		if err != nil {
			return "", fmt.Errorf("verify path %q: %w", current, err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q escapes mount root %q", current, cleanRoot)
		}
	}
	return joined, nil
}

// ensureNoSymlinkPath rejects existing symlink components in an effective copy root just before use.
func ensureNoSymlinkPath(path string) error {
	cleanPath := filepath.Clean(path)
	current := filepath.VolumeName(cleanPath)
	if current == "" {
		current = string(os.PathSeparator)
	}
	trimmed := strings.TrimPrefix(cleanPath, current)
	for _, part := range strings.Split(trimmed, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("inspect path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path must not traverse symlink component %q", current)
		}
	}
	return nil
}

// cleanupDirectory unmounts the workspace roots and removes the temporary directories it created.
func (r *Runner) cleanupDirectory(tempPath string) {
	if len(tempPath) == 0 {
		return
	}

	// Unmount pinned subpath mounts before the backing source and destination roots.
	srcMountPath, srcErr := safeWorkspacePath(tempPath, r.SourceMountName)
	dstMountPath, dstErr := safeWorkspacePath(tempPath, r.DestinationMountName)
	pinnedSrcPath, pinnedSrcErr := safeWorkspacePath(tempPath, pinnedSourceMountName)
	pinnedDstPath, pinnedDstErr := safeWorkspacePath(tempPath, pinnedDestinationMountName)
	if srcErr != nil || dstErr != nil || pinnedSrcErr != nil || pinnedDstErr != nil {
		util.ErrLog.Printf("skip unsafe cleanup paths: source=%v destination=%v pinnedSource=%v pinnedDestination=%v", srcErr, dstErr, pinnedSrcErr, pinnedDstErr)
		return
	}

	umountPaths := []string{pinnedSrcPath, pinnedDstPath, srcMountPath, dstMountPath}
	removePaths := []string{pinnedSrcPath, pinnedDstPath, srcMountPath, dstMountPath, tempPath}

	for _, path := range umountPaths {
		if err := sys.Umount(path); err != nil {
			util.ErrLog.Printf("failed to unmount %s: %s", path, err)
		}
	}
	for _, path := range removePaths {
		if err := os.Remove(path); err != nil {
			util.ErrLog.Printf("failed to remove %s: %s", path, err)
		}
	}
}
