package find

import (
	"context"
	"log"
	"strconv"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/system"
)

func TestScanner_testRegex(t1 *testing.T) {
	lines := []string{
		"a",
		"51791395877894146  598 -rw-r--r--   1 root     root       612192 Feb 15 20:54 /fixture/root/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/rein20/dataset-host/image-prompt-fixture/input/image-set-2m/00000000-0000-4000-8000-000000000000.png",
		"35465847138781934 3571 -rw-r--r--   1 root     root      3655851 Nov 11  2021 /tmp/fixture/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/org/deployments/model_fixture_low_loss/model_recommendation/output/model_fixture_long_train/account_map.json",
		"6192449664352658    0 drwxr-xr-x   2 root     root            0 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9",
		"7881299523698987    1 lrwxrwxrwx   1 root     root           52 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9/vocab.json -> ../../blobs/0c9fccca89c9a8d2554dc00cc621c044aae04adb",
		"38035803        8 -rw-r--r--    1 example.user          staff                 362 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-os.cpython-311.pyc",
		"38035731        8 -rw-r--r--    1 example.user          staff                 562 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.Adw.cpython-311.pyc",
		"38035759        8 -rw-r--r--    1 example.user          staff                 570 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.GstNet.cpython-311.pyc",
	}

	for _, line := range lines {
		log.Printf("line: %s", line)
		matched := findFormat.FindStringSubmatchIndex(line)
		if groups := len(matched) / 2; groups < 1 {
			log.Printf("scan: invalid find result %s", line)
			continue
		}

		match := func(i int) string {
			if len(matched) < (i+1)*2 {
				return ""
			}
			return line[matched[i*2]:matched[i*2+1]]
		}

		inode, _ := strconv.Atoi(match(1))
		size, _ := strconv.Atoi(match(2))
		mode := util.UnFilemodeStr(match(3))
		num_of_hardlink, _ := strconv.Atoi(match(4))
		owner := match(5)
		group := match(6)
		store_size, _ := strconv.Atoi(match(7))
		date := match(8)
		path := match(9)

		log.Printf(
			"inode %d size %d mode %s num_of_hardlink %d owner %s group %s store_size %d date %s path %s",
			inode, size, mode.String(), num_of_hardlink, owner, group, store_size, date, path,
		)
		for i := 1; i < 10; i++ {
			log.Printf("group(%d): %d->%d %s", i, matched[i*2], matched[i*2+1], match(i))
		}

	}
}

func TestScanner_parseFindEntry(t1 *testing.T) {
	s := &Scanner{}
	lines := []string{
		"51791395877894146  598 -rw-r--r--   1 root     root       612192 Feb 15 20:54 /fixture/root/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/rein20/dataset-host/image-prompt-fixture/input/image-set-2m/00000000-0000-4000-8000-000000000000.png",
		"35465847138781934 3571 -rw-r--r--   1 root     root      3655851 Nov 11  2021 /tmp/fixture/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/org/deployments/model_fixture_low_loss/model_recommendation/output/model_fixture_long_train/account_map.json",
		"6192449664352658    0 drwxr-xr-x   2 root     root            0 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9",
		"7881299523698987    1 lrwxrwxrwx   1 root     root           52 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9/vocab.json -> ../../blobs/0c9fccca89c9a8d2554dc00cc621c044aae04adb",
		"38035803        8 -rw-r--r--    1 example.user          staff                 362 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-os.cpython-311.pyc",
		"38035731        8 -rw-r--r--    1 example.user          staff                 562 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.Adw.cpython-311.pyc",
		"38035759        8 -rw-r--r--    1 example.user          staff                 570 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.GstNet.cpython-311.pyc",
	}

	for _, line := range lines {
		entry, err := s.parseFindEntry([]byte(line))
		if err != nil {
			panic(err)
		}

		log.Print(entry)
	}
}

func TestScanner_scanDirectory(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Scanner{}
	infoChan := make(chan returns.Fileinfo)
	go s.scanDirectory(ctx, ".", infoChan)
	for entry := range infoChan {
		log.Printf("entry %v", entry)
	}
	log.Printf("ended ")
}

func TestScanner_executeFindCommand(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Scanner{}
	s.FinderBinaryPath = "find"
	infoChan := make(chan returns.Fileinfo)
	go s.executeFind(ctx, ".", infoChan)
	for entry := range infoChan {
		log.Printf("entry %v", entry)
	}
}
