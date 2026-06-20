package rsync

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier/find"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func TestLogger(t *testing.T) {
	bar := progressbar.NewOptions(1000,
		progressbar.OptionSetWriter(util.LogWriter{}),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionOnCompletion(func() { log.Print("?") }),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionSetItsString("op"),
		progressbar.OptionSetDescription(fmt.Sprintf("rsync[%d]", 33)),
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

func TestRsyncTask_Regex(t *testing.T) {
	re := regexp.MustCompile(`^(.+?)( is uptodate)?$`)

	line := "aaa"
	matched := re.FindStringSubmatchIndex(line)
	groups := (len(matched) / 2) - 1
	log.Printf("matched %v", matched)
	log.Printf("groups %d", groups)

	match := func(i int) string {
		if len(matched) < (i+1)*2 {
			return ""
		} else if matched[i*2] < 0 || matched[i*2+1] < 0 {
			return ""
		}

		return line[matched[i*2]:matched[i*2+1]]
	}
	log.Printf("group(1) %s", match(1))

	if len(match(2)) > 0 {
		log.Printf("group(2) %s", match(2))
	}

}

func TestRsyncArgs_assembleArgs(t *testing.T) {
	args := args.RsyncArgs{
		Verbose:            false,
		Delete:             false,
		PreservePermission: false,
		PreserveOwnership:  false,
		CopySpecial:        false,
		Compress:           false,
		WholeFile:          true,
		Inplace:            false,
		Recursive:          true,
		BandwidthLimit:     "20m",
	}
	log.Print("format arguments", args.Assemble("src", "dst"))
}

func requireRsync(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("rsync executable not found in PATH")
	}
}

func TestRsyncTask_Execute_find_method(t1 *testing.T) {
	requireRsync(t1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &find.Scanner{
		FinderBinaryPath: "find",
		EntryChannelSize: 0,
	}

	src, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w", err))
	}

	defer os.RemoveAll(src)
	t1.Log("source directory created ", src)
	dst, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w", err))
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

		_, err = io.Copy(f, &io.LimitedReader{R: rand.Reader, N: 4 * 1024})
		if err != nil {
			panic(err)
		}
		f.Close()

		f, err = os.Create(filepath.Join(src, filename))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(f, &io.LimitedReader{R: rand.Reader, N: 4 * 1024})
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
		t = &Task{
			FileMode:        0o640,
			SourcePath:      src,
			DestinationPath: dst,
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

func TestRsyncTask_Execute_scan_method(t1 *testing.T) {
	requireRsync(t1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &find.Scanner{
		EntryChannelSize: 0,
	}

	src, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w", err))
	}

	defer os.RemoveAll(src)
	t1.Log("source directory created ", src)
	dst, err := os.MkdirTemp("", fmt.Sprintf(".tmp-%x", int64(os.Getpid())^time.Now().Unix()))
	if err != nil {
		panic(fmt.Errorf("failed to create a tempfile :%w", err))
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

		_, err = io.Copy(f, &io.LimitedReader{R: rand.Reader, N: 4 * 1024})
		if err != nil {
			panic(err)
		}
		f.Close()

		f, err = os.Create(filepath.Join(src, filename))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(f, &io.LimitedReader{R: rand.Reader, N: 4 * 1024})
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
		t = &Task{
			FileMode:        0o640,
			SourcePath:      src,
			DestinationPath: dst,
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
