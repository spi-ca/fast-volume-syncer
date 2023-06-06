package syncer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/find"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
)

var (
	pathSeperatorStr = string(filepath.Separator)
)

type Runner struct {
	Sandboxed          bool
	SandboxMountOption string

	Args rsync.Args

	Source          common.RemoteInfo
	SourceMountName string

	Destination          common.RemoteInfo
	DestinationMountName string

	ScanDuration     time.Duration
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int

	RetryAttempts  int
	RetryDelay     time.Duration
	RetryMaxDelay  time.Duration
	RetryMaxJitter time.Duration
}

func (r *Runner) logLineByLine(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		log.Print(prefix, line)
	}
}

func (r *Runner) logVolumeInfo(path string) {
	ctx, cencel := context.WithCancel(context.Background())
	defer cencel()
	if out, err := exec.CommandContext(ctx, "ls", "-al", path).CombinedOutput(); err != nil {
		log.Printf("failed to start executable(ls): %v", err)
	} else {
		log.Printf("directory_info(%s)=>", path)
		r.logLineByLine(bytes.NewReader(out), "\t")
	}

	if out, err := exec.CommandContext(ctx, "findmnt", "-T", path).CombinedOutput(); err != nil {
		log.Printf("failed to start executable(findmnt): %v", err)
	} else {
		log.Printf("mount_info(%s)=>", path)
		r.logLineByLine(bytes.NewReader(out), "\t")
	}

	if out, err := exec.CommandContext(ctx, "df", "-h", path).CombinedOutput(); err != nil {
		log.Printf("failed to start executable(df): %v", err)
	} else {
		log.Printf("fs            =>\t")
		r.logLineByLine(bytes.NewReader(out), "\t")
	}
}

func (r *Runner) locateFindBinary() string {
	if len(r.FinderBinaryPath) < 1 {
		return ""
	}

	if foundPath, err := exec.LookPath(r.FinderBinaryPath); err != nil {
		log.Printf("find path(%s) not found", r.FinderBinaryPath)
		return ""
	} else {
		absPath, _ := filepath.Abs(foundPath)
		return absPath
	}
}
func (r *Runner) prepareDirectory() (string, string, string, error) {
	if r.Sandboxed {
		if err := common.Sandbox(r.SandboxMountOption); err != nil {
			return "", "", "", fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}
	// 아래 영역은 이제 host os 와 격리되었다.

	tempDir, err := os.MkdirTemp("", "syncer-")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to make temp directory: %w", err)
	}

	log.Printf("created temporary directory: '%s'", tempDir)

	if err = os.Chdir(tempDir); err != nil {
		return tempDir, "", "", fmt.Errorf("failed to change directory(%s): %w", tempDir, err)
	}

	// 이제 tempDir가 cwd이다.

	// host:path -> {tempdir}/mountName
	srcMountPath := filepath.Join(tempDir, r.SourceMountName)
	dstMountPath := filepath.Join(tempDir, r.DestinationMountName)

	if err = common.Mount(r.Source.MountInfo, srcMountPath); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", r.Source.MountInfo, srcMountPath, err)
	}
	log.Printf("source mount success!(%s %s)", r.Source.MountInfo, srcMountPath)

	if err = common.Mount(r.Destination.MountInfo, dstMountPath); err != nil {
		return tempDir, "", "", fmt.Errorf("mount failed(%s %s) : %w", r.Destination.MountInfo, dstMountPath, err)
	}
	log.Printf("destination mount success!(%s %s)", r.Destination.MountInfo, dstMountPath)

	srcMountSubPath := filepath.Join(srcMountPath, r.Source.SubPath)
	dstMountSubPath := filepath.Join(dstMountPath, r.Destination.SubPath)

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
		log.Printf("directory %s created", dstMountSubPath)
	}

	return tempDir, srcMountSubPath, dstMountSubPath, nil
}
func (r *Runner) cleanupDirectory(tempPath string) {
	if len(tempPath) == 0 {
		return
	}

	srcMountPath := filepath.Join(tempPath, r.SourceMountName)
	dstMountPath := filepath.Join(tempPath, r.DestinationMountName)

	umountPaths := []string{srcMountPath, dstMountPath}
	removePaths := []string{srcMountPath, dstMountPath, tempPath}
	for _, path := range umountPaths {
		if err := common.Umount(path); err != nil {
			log.Printf("failed to unmount %s: %s", path, err)
		}
	}
	for _, path := range removePaths {
		if err := os.Remove(path); err != nil {
			log.Printf("failed to remove %s: %s", path, err)
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

	log.Printf("TaskSize %d ChunkSize %d srcPath: %s dstPath: %s", r.TaskSize, r.ChunkSize, srcPath, dstPath)

	r.logVolumeInfo(srcPath)
	r.logVolumeInfo(dstPath)

	log.Print("=> split rsync")

	rsyncInvoker := rsync.Task{
		Arguments:       r.Args.Assemble(srcPath, dstPath),
		RetryAttempts:   r.RetryAttempts,
		RetryDelay:      r.RetryDelay,
		RetryMaxDelay:   r.RetryMaxDelay,
		RetryMaxJitter:  r.RetryMaxJitter,
		DestinationPath: dstPath,
	}

	if r.Args.Recursive {
		return rsyncInvoker.Execute(ctx, nil)
	}

	scanner := find.Scanner{
		FinderBinaryPath: finderBinaryPath,
		TaskSize:         r.TaskSize,
		ChunkSize:        r.ChunkSize,
	}

	entryRecvChan := scanner.Scan(ctx, srcPath)

	joiner := newChunkJoiner(r.TaskSize, r.ChunkSize, r.ScanDuration, &rsyncInvoker)

	err = joiner.Execute(ctx, entryRecvChan)
	if err == nil && ctx.Err() == nil {
		r.logVolumeInfo(srcPath)
		r.logVolumeInfo(dstPath)
		log.Printf("볼륨 싱크 완료(%s->%s)", srcPath, dstPath)
	}
	return err

}
