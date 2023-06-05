package rsync

//
//func TestRsyncTask_Execute(t1 *testing.T) {
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	log.Printf("??")
//	s := Task{}
//	s.Execute(ctx)
//}
//
//func TestRsyncArgs_assembleArgs(t *testing.T) {
//	args := Args{
//		Verbose:            false,
//		PreservePermission: false,
//		PreserveOwnership:  false,
//		CopySpecial:        false,
//		Compress:           false,
//		WholeFile:          true,
//		Inplace:            false,
//		DryRun:             false,
//		Recursive:          true,
//	}
//	log.Print("??", args.Assemble("src", "dst"))
//}
//
//func TestRsyncTask_logVolumeInfo(t1 *testing.T) {
//	t := &Task{}
//	t.logVolumeInfo(context.Background(), "/")
//}
//
//func TestRsyncTask_scanDirectory(t1 *testing.T) {
//
//	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
//	defer cancel()
//	t := &Task{}
//	for entry := range t.scanDirectory(ctx, "/Users/example.user/Codebase/bc/legacy-volume-migration/cmd") {
//		log.Printf("entry %v", entry)
//	}
//	log.Printf("ended ")
//}
//
//func TestRsyncTask_scanSourceByChunk(t1 *testing.T) {
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	t := &Task{}
//	t.ChunkSize = 1000
//	pool := newFileinfoChunkPool(t.ChunkSize)
//	for entry := range t.scanSourceByChunk(ctx, pool, "/Users/example.user/Codebase/bc") {
//		func() {
//			defer pool.Put(entry)
//			log.Printf("entry %d", len(entry))
//		}()
//	}
//}
//
//func TestRsyncTask_executeFindCommand(t1 *testing.T) {
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	t := &Task{}
//	t.FindCommandPath = "find"
//	entryChan := t.findScanDirectory(ctx, "/Users/example.user/Codebase/bc/legacy-volume-migration/cmd")
//	for entry := range entryChan {
//		log.Printf("entry %v", entry)
//	}
//}
//
//func TestRsyncTask_parseFindEntry(t1 *testing.T) {
//	lines := []string{
//		"a",
//		"51791395877894146  598 -rw-r--r--   1 root     root       612192 Feb 15 20:54 /fixture/root/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/rein20/dataset-host/image-prompt-fixture/input/image-set-2m/00000000-0000-4000-8000-000000000000.png",
//		"35465847138781934 3571 -rw-r--r--   1 root     root      3655851 Nov 11  2021 /tmp/fixture/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/org/deployments/model_fixture_low_loss/model_recommendation/output/model_fixture_long_train/account_map.json",
//		"6192449664352658    0 drwxr-xr-x   2 root     root            0 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9",
//		"7881299523698987    1 lrwxrwxrwx   1 root     root           52 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9/vocab.json -> ../../blobs/0c9fccca89c9a8d2554dc00cc621c044aae04adb",
//		"38035803        8 -rw-r--r--    1 example.user          staff                 362 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-os.cpython-311.pyc",
//		"38035731        8 -rw-r--r--    1 example.user          staff                 562 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.Adw.cpython-311.pyc",
//		"38035759        8 -rw-r--r--    1 example.user          staff                 570 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.GstNet.cpython-311.pyc",
//	}
//
//	for _, line := range lines {
//		log.Printf("line: %s", line)
//		matched := findFormat.FindStringSubmatchIndex(line)
//		if groups := len(matched) / 2; groups < 1 {
//			log.Printf("scan: invalid find result %s", line)
//			continue
//		}
//
//		match := func(i int) string {
//			if len(matched) < (i+1)*2 {
//				return ""
//			}
//			return line[matched[i*2]:matched[i*2+1]]
//		}
//
//		inode, _ := strconv.Atoi(match(1))
//		size, _ := strconv.Atoi(match(2))
//		mode := UnFilemodeStr(match(3))
//		num_of_hardlink, _ := strconv.Atoi(match(4))
//		owner := match(5)
//		group := match(6)
//		store_size, _ := strconv.Atoi(match(7))
//		date := match(8)
//		path := match(9)
//
//		log.Printf(
//			"inode %d size %d mode %s num_of_hardlink %d owner %s group %s store_size %d date %s path %s",
//			inode, size, mode.String(), num_of_hardlink, owner, group, store_size, date, path,
//		)
//		for i := 1; i < 10; i++ {
//			log.Printf("group(%d): %d->%d %s", i, matched[i*2], matched[i*2+1], match(i))
//		}
//
//	}
//}
//
//func TestRsyncTask_scanSourceByChunkWithFindCommand(t1 *testing.T) {
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	t := &Task{}
//	t.ChunkSize = 1000
//	t.FindCommandPath = "find"
//
//	pool := newFileinfoChunkPool(t.ChunkSize)
//	for entry := range t.scanSourceByChunk(ctx, pool, "/Users/example.user/Codebase/bc") {
//		func() {
//			defer pool.Put(entry)
//			log.Printf("entry %d", len(entry))
//		}()
//	}
//}
//
//func TestRsyncTask_executeRsync(t1 *testing.T) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	t := &Task{}
//	t.ChunkSize = 1000
//	t.FindCommandPath = "find"
//
//	args := []string{}
//	filelist := []fileinfo{
//		{
//			Path: "a",
//			Mode: 0,
//			Size: 0,
//		}, {
//			Path: "b",
//			Mode: 0,
//			Size: 0,
//		},
//	}
//	err := t.executeRsync(ctx, args, filelist)
//	if err != nil {
//		log.Printf("error occured : %v", err)
//	}
//}
//
//func TestRsyncTask_prepare(t1 *testing.T) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	t := &Task{}
//	t.ChunkSize = 1000
//	t.FindCommandPath = "find"
//
//	err := t.prepare(ctx)
//	if err != nil {
//		log.Printf("error occured : %v", err)
//	}
//}
