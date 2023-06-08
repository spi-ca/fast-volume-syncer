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
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/find"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/system"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

var (
	pathSeperatorStr = string(filepath.Separator)
)

type Runner struct {
	Sandboxed bool
	Common    args.SyncerCommonArguments

	SourceMountPath    string
	SourceMountSubPath string

	DestinationMountPath    string
	DestinationMountSubPath string
}

func (r *Runner) logLineByLine(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		util.InfoLog.Print(prefix, line)
	}
}

func (r *Runner) logVolumeInfo(path string) {
	ctx, cencel := context.WithCancel(context.Background())
	defer cencel()
	if out, err := exec.CommandContext(ctx, "ls", "-al", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(ls): %v", err)
	} else {
		util.InfoLog.Printf("directory_info(%s)=>", path)
		r.logLineByLine(bytes.NewReader(out), "\t")
	}

	if out, err := exec.CommandContext(ctx, "findmnt", "-T", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(findmnt): %v", err)
	} else {
		util.InfoLog.Printf("mount_info(%s)=>", path)
		r.logLineByLine(bytes.NewReader(out), "\t")
	}

	if out, err := exec.CommandContext(ctx, "df", "-h", path).CombinedOutput(); err != nil {
		util.ErrLog.Printf("failed to start executable(df): %v", err)
	} else {
		util.InfoLog.Printf("fs            =>\t")
		r.logLineByLine(bytes.NewReader(out), "\t")
	}
}

func (r *Runner) locateFindBinary() string {
	if len(r.Common.FinderBinaryPath) < 1 {
		return ""
	}

	if foundPath, err := exec.LookPath(r.Common.FinderBinaryPath); err != nil {
		util.ErrLog.Printf("find path(%s) not found", r.Common.FinderBinaryPath)
		return ""
	} else {
		absPath, _ := filepath.Abs(foundPath)
		return absPath
	}
}
func (r *Runner) prepareDirectory() (string, string, string, error) {
	if r.Sandboxed {
		if err := system.Sandbox(r.Common.SandboxMountOption); err != nil {
			return "", "", "", fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}
	// 아래 영역은 이제 host os 와 격리되었다.

	tempDir, err := os.MkdirTemp("", "syncer-")
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
	srcMountPath := filepath.Join(tempDir, r.Common.SourceMountName)
	dstMountPath := filepath.Join(tempDir, r.Common.DestinationMountName)
	// 실제 복사 대상
	srcMountSubPath := filepath.Join(srcMountPath, r.SourceMountSubPath)
	dstMountSubPath := filepath.Join(dstMountPath, r.DestinationMountSubPath)

	srcMountInfo := returns.MountInfo{
		Host:    r.Common.SourceMountHost,
		Path:    r.SourceMountPath,
		Options: r.Common.SourceMountOptions,
	}
	dstMountInfo := returns.MountInfo{
		Host:    r.Common.DestinationMountHost,
		Path:    r.DestinationMountPath,
		Options: r.Common.DestinationMountOptions,
	}

	if err = system.Mount(srcMountInfo.Source(), srcMountPath, srcMountInfo.Type(), srcMountInfo.RefinedOptions()); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", dstMountInfo, srcMountPath, err)
	}
	util.InfoLog.Printf("source mount success!(%s %s)", srcMountInfo, srcMountPath)

	if err = system.Mount(dstMountInfo.Source(), dstMountPath, dstMountInfo.Type(), dstMountInfo.RefinedOptions()); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", dstMountInfo, dstMountPath, err)
	}
	util.InfoLog.Printf("destination mount success!(%s %s)", dstMountInfo, dstMountPath)

	// source 확인
	sourceStat, err := os.Stat(srcMountSubPath)
	if err != nil {
		return tempDir, "", "", fmt.Errorf("failed to get source file info(%s): %w", srcMountSubPath, err)
	} else if sourceStat.IsDir() {
		// directory인 경우 slash 추가.
		srcMountSubPath += pathSeperatorStr
	}

	// destination 확인
	destinationFilemode := sourceStat.Mode() | 0o700

	if err := os.MkdirAll(dstMountSubPath, destinationFilemode.Perm()); err == nil {
		util.InfoLog.Printf("directory %s created", dstMountSubPath)
	}

	return tempDir, srcMountSubPath, dstMountSubPath, nil
}

func (r *Runner) cleanupDirectory(tempPath string) {
	if len(tempPath) == 0 {
		return
	}

	// 볼륨 마운트 위치
	srcMountPath := filepath.Join(tempPath, r.Common.SourceMountName)
	dstMountPath := filepath.Join(tempPath, r.Common.DestinationMountName)

	umountPaths := []string{srcMountPath, dstMountPath}
	removePaths := []string{srcMountPath, dstMountPath}
	if r.Sandboxed {
		removePaths = append(removePaths, tempPath)
	}

	for _, path := range umountPaths {
		if err := system.Umount(path); err != nil {
			util.ErrLog.Printf("failed to unmount %s: %s", path, err)
		}
	}
	for _, path := range removePaths {
		if err := os.Remove(path); err != nil {
			util.ErrLog.Printf("failed to remove %s: %s", path, err)
		}
	}
}
func (r *Runner) Execute(ctx context.Context) error {
	// chdir 영향으로 미리 발견하여야 된다.
	finderBinaryPath := r.locateFindBinary()

	tempPath, srcPath, dstPath, err := r.prepareDirectory()
	defer r.cleanupDirectory(tempPath)
	if err != nil {
		return err
	}

	util.InfoLog.Printf("TaskSize %d ChunkSize %d srcPath: %s dstPath: %s", r.Common.TaskSize, r.Common.ChunkSize, srcPath, dstPath)

	r.logVolumeInfo(srcPath)
	r.logVolumeInfo(dstPath)

	util.InfoLog.Print("=> split rsync")

	rsyncInvoker := rsync.Task{
		Arguments:       r.Common.Args.Assemble(srcPath, dstPath),
		Retry:           r.Common.Retry,
		DestinationPath: dstPath,
	}

	if r.Common.Args.Recursive {
		return rsyncInvoker.Execute(ctx, nil)
	}

	scanner := find.Scanner{
		FinderBinaryPath: finderBinaryPath,
		TaskSize:         r.Common.TaskSize,
		ChunkSize:        r.Common.ChunkSize,
	}

	entryRecvChan := scanner.Scan(ctx, srcPath)

	joiner := newChunkJoiner(r.Common.TaskSize, r.Common.ChunkSize, r.Common.ScanDuration, &rsyncInvoker)

	err = joiner.Execute(ctx, entryRecvChan)
	if err == nil && ctx.Err() == nil {
		r.logVolumeInfo(srcPath)
		r.logVolumeInfo(dstPath)
		util.InfoLog.Printf("볼륨 싱크 완료(%s->%s)", srcPath, dstPath)
	}
	return err

}
