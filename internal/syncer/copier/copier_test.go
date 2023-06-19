package copier

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/find"
)

func TestCopier_copyNewFile(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &find.Scanner{
		FinderBinaryPath: "find",
		EntryChannelSize: 0,
	}

	src, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w; %w", ErrCopierCopyFailed, err))
	}
	defer os.RemoveAll(src)
	t1.Log("source directory created ", src)
	dst, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w; %w", ErrCopierCopyFailed, err))
	}
	defer os.RemoveAll(dst)
	t1.Log("destination directory created ", dst)

	dirPath := filepath.Join(src, "dir_test")
	err = os.Mkdir(dirPath, 0o755)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 3000; i++ {
		filename := fmt.Sprintf("f_%04d", i)
		link_filename := fmt.Sprintf("l_%04d", i)
		f, err := os.Create(filepath.Join(dirPath, filename))
		if err != nil {
			panic(err)
		}

		_, err = io.Copy(f, &io.LimitedReader{rand.Reader, 4 * 1024})
		if err != nil {
			panic(err)
		}
		f.Close()

		f, err = os.Create(filepath.Join(src, filename))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(f, &io.LimitedReader{rand.Reader, 4 * 1024})
		if err != nil {
			panic(err)
		}
		f.Close()

		err = os.Symlink("/nonexist", filepath.Join(dirPath, link_filename))
		if err != nil {
			panic(err)
		}
		err = os.Symlink("/nonexist", filepath.Join(src, link_filename))
		if err != nil {
			panic(err)
		}
	}

	infoChan, scannerErrorChan := s.Scan(ctx, src)
	t := &Copier{
		SourceRoot:      src,
		DestinationRoot: dst,
		FileMode:        0o640,
	}
	files := []returns.Fileinfo{}
	for entry := range infoChan {
		files = append(files, entry)
	}
	err = t.Execute(ctx, files)
	if err != nil {
		panic(err)
	}
	if scannerErr, ok := <-scannerErrorChan; ok {
		panic(scannerErr)
	}
}
