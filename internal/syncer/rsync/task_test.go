package rsync

import (
	"fmt"
	"log"
	"regexp"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
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
