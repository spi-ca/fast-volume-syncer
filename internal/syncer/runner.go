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

type Runner struct {
	Sandboxed bool

	ReportEnabled      bool
	SandboxMountOption string

	SourceMountHost    string
	SourceMountOptions string
	SourceMountName    string

	DestinationMountHost    string
	DestinationMountOptions string
	DestinationMountName    string

	Copier copier.Runner

	SourceMountPath    string
	SourceMountSubPath string

	DestinationMountPath    string
	DestinationMountSubPath string
}

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

	err = r.Copier.Execute(ctx, srcPath, dstPath)

	if r.ReportEnabled {
		r.logVolumeInfo(srcPath)
		r.logVolumeInfo(dstPath)
	}

	return err

}

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

func (r *Runner) logVolumeInfo(path string) {
	if out, err := exec.Command("ls", "-al", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(ls): %v", err)
	} else {
		r.logOutput(bytes.NewReader(out), fmt.Sprintf("directory_info(%s)=>", path), "\t")
	}

	if out, err := exec.Command("findmnt", "-T", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(findmnt): %v", err)
	} else {
		r.logOutput(bytes.NewReader(out), fmt.Sprintf("mount_info(%s)=>", path), "\t")
	}

	if out, err := exec.Command("df", "-h", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(df): %v", err)
	} else {
		r.logOutput(bytes.NewReader(out), "fs            =>\t", "\t")
	}
}

func (r *Runner) prepareMountPoint() (string, string, string, error) {
	if r.Sandboxed {
		if err := sys.Sandbox(r.SandboxMountOption); err != nil {
			return "", "", "", fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}

	// 아래 영역은 이제 host os 와 격리되었다.
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("syncer-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to make temp directory: %w", err)
	}

	util.InfoLog.Printf("created temporary directory: '%s'", tempDir)

	if err = os.Chdir(tempDir); err != nil {
		return tempDir, "", "", fmt.Errorf("failed to change directory(%s): %w", tempDir, err)
	}

	// 이제 tempDir가 cwd이다.

	// host:path -> {tempdir}/mountName
	// 볼륨 마운트 위치
	srcMountPath := filepath.Join(tempDir, r.SourceMountName)
	dstMountPath := filepath.Join(tempDir, r.DestinationMountName)
	// 실제 복사 대상
	srcMountSubPath := filepath.Join(srcMountPath, r.SourceMountSubPath)
	dstMountSubPath := filepath.Join(dstMountPath, r.DestinationMountSubPath)

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
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", dstMountInfo, srcMountPath, err)
	}
	util.InfoLog.Printf("source mount success!(%s %s)", srcMountInfo, srcMountPath)

	if err = sys.Mount(dstMountInfo.Source(), dstMountPath, dstMountInfo.Type(), dstMountInfo.RefinedOptions()); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", dstMountInfo, dstMountPath, err)
	}
	util.InfoLog.Printf("destination mount success!(%s %s)", dstMountInfo, dstMountPath)

	return tempDir, srcMountSubPath, dstMountSubPath, nil
}

func (r *Runner) cleanupDirectory(tempPath string) {
	if len(tempPath) == 0 {
		return
	}

	// 볼륨 마운트 위치
	srcMountPath := filepath.Join(tempPath, r.SourceMountName)
	dstMountPath := filepath.Join(tempPath, r.DestinationMountName)

	umountPaths := []string{srcMountPath, dstMountPath}
	removePaths := []string{srcMountPath, dstMountPath}
	if r.Sandboxed {
		removePaths = append(removePaths, tempPath)
	}

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
