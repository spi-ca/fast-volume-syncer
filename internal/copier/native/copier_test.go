package native

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier/find"
)

func TestLogger(t *testing.T) {

	bar := progressbar.NewOptions(1000,
		progressbar.OptionSetWriter(util.LogWriter{}),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionSetItsString("op"),
		progressbar.OptionSetDescription(fmt.Sprintf("[chk:%d]\t", 33)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "-",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	defer bar.Close()
	for i := 0; i < 1000; i++ {
		bar.Add(1)
		time.Sleep(5 * time.Millisecond)
	}
}

func TestCopier_Execute_find_method(t1 *testing.T) {

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
		link_filename := fmt.Sprintf("l %04d", i)
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

		err = os.Symlink("/nonex ist", filepath.Join(dirPath, link_filename))
		if err != nil {
			panic(err)
		}
		err = os.Symlink("/nonex ist", filepath.Join(src, link_filename))
		if err != nil {
			panic(err)
		}
	}

	infoChan, scannerErrorChan := s.Scan(ctx, src)
	var (
		t = &Copier{
			SourceRoot:      src,
			DestinationRoot: dst,
			FileMode:        0o640,
		}
		files []returns.Fileinfo
	)
	for entry := range infoChan {
		files = append(files, entry)
	}
	_, err = t.Execute(ctx, files)
	if err != nil {
		panic(err)
	}
	if scannerErr, ok := <-scannerErrorChan; ok {
		panic(scannerErr)
	}
}

func TestCopier_Execute_scan_method(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &find.Scanner{
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
		link_filename := fmt.Sprintf("l %04d", i)
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

		err = os.Symlink("/nonex ist", filepath.Join(dirPath, link_filename))
		if err != nil {
			panic(err)
		}
		err = os.Symlink("/nonex ist", filepath.Join(src, link_filename))
		if err != nil {
			panic(err)
		}
	}

	infoChan, scannerErrorChan := s.Scan(ctx, src)
	var (
		t = &Copier{
			SourceRoot:      src,
			DestinationRoot: dst,
			FileMode:        0o640,
		}
		files []returns.Fileinfo
	)
	for entry := range infoChan {
		files = append(files, entry)
	}
	_, err = t.Execute(ctx, files)
	if err != nil {
		panic(err)
	}
	if scannerErr, ok := <-scannerErrorChan; ok {
		panic(scannerErr)
	}
}
