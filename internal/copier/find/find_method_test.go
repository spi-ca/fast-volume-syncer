package find

import (
	"context"
	"log"
	"strconv"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
)

func TestScanner_testRegex(t1 *testing.T) {
	lines := []string{
		"a",
		"562949984882802    7 -rw-r--r--   1 root     root         6822 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_211609.txt",
		"562949984882872    8 -rw-r--r--   1 root     root         7957 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_31928.txt",
		"562949984882803   11 -rw-r--r--   1 root     root        10260 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_193801.txt",
		"562949984882873    5 -rw-r--r--   1 root     root         4364 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_38123.txt",
		"562949984882804    5 -rw-r--r--   1 root     root         4455 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_185840.txt",
		"562949984882874    9 -rw-r--r--   1 root     root         8284 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_25557.txt",
		"562949984882805    8 -rw-r--r--   1 root     root         7942 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_173461.txt",
		"1125899938327967    5 -rw-r--r--   1 root     root         5086 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_124635.txt",
		"562949984882806   12 -rw-r--r--   1 root     root        11670 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_162252.txt",
		"1125899938327968    5 -rw-r--r--   1 root     root         4398 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_92679.txt",
		"562949984882807    8 -rw-r--r--   1 root     root         7862 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_34692.txt",
		"1125899938327969    9 -rw-r--r--   1 root     root         8312 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_111492.txt",
		"1125899938327970    8 -rw-r--r--   1 root     root         7220 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_88633.txt",
		"1125899938327971    2 -rw-r--r--   1 root     root         1998 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_92435.txt",
		"1125899938327972    4 -rw-r--r--   1 root     root         4052 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_37492.txt",
		"1125899938327973   11 -rw-r--r--   1 root     root        10689 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_128165.txt",
		"1125899938327974    8 -rw-r--r--   1 root     root         7754 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_87747.txt",
		"562949984882808    8 -rw-r--r--   1 root     root         7896 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_146627.txt",
		"1125899938327975   10 -rw-r--r--   1 root     root         9615 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_98458.txt",
		"562949984882809    7 -rw-r--r--   1 root     root         6923 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_162346.txt",
		"1125899938327976   11 -rw-r--r--   1 root     root        11247 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_122811.txt",
		"562949984882810    9 -rw-r--r--   1 root     root         9100 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_154966.txt",
		"562949984882811    7 -rw-r--r--   1 root     root         6891 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_83198.txt",
		"1125899938327977   12 -rw-r--r--   1 root     root        12257 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_67471.txt",
		"1125899938327978    5 -rw-r--r--   1 root     root         4375 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_86047.txt",
		"562949984882812    9 -rw-r--r--   1 root     root         9063 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_139665.txt",
		"562949984882813    6 -rw-r--r--   1 root     root         5287 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_139827.txt",
		"562949984882814    4 -rw-r--r--   1 root     root         3625 Oct 14  2020 fixture/rw/fixture-data/fixture/rw/datasets/dataset-a/archive_entry_147033.txt",
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
		mode := sys.UnFilemodeStr(match(3))
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
		"23643898148733866   15 -rw-r--r--   1 root     root        14415 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ff8fdc2aa13faa.json",
		"31525197496617622   25 -rw-r--r--   1 root     root        25200 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffaec78cddcba3.json",
		"16325548754174623    6 -rw-r--r--   1 root     root         5753 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ff900823a7f97b.json",
		"1688849965084091   27 -rw-r--r--   1 root     root        26771 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffaeec78e9045c.json",
		"24206848103198616    3 -rw-r--r--   1 root     root         2232 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ff90dc5862b0f4.json",
		"29836347637195917   23 -rw-r--r--   1 root     root        22912 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffb04036d61de1.json",
		"28147497786343861    6 -rw-r--r--   1 root     root         5612 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffb12ef374419a.json",
		"19140298521765726   61 -rw-r--r--   1 root     root        62186 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffb1d6de4d1486.json",
		"15199648847443209   46 -rw-r--r--   1 root     root        46996 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ff6ec6a34aec28.json",
		"63050394888249546   11 -rw-r--r--   1 root     root        10391 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffb2c29f660457.json",
		"17451448660930700   13 -rw-r--r--   1 root     root        12990 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ff6f4764135a25.json",
		"63613344841452655    9 -rw-r--r--   1 root     root         8417 May  9  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/lowcode/code-fixture/input/CodeFixture/train/ffb348011806aa.json",
		"68679894491775710    1 -rw-r-----   1 740376   89939         458 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/122/sample_270.png",
		"33776997365964070    1 -rw-r--r--   1 root     root          234 Mar 16 13:13 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/repos/tensorlib/aten/src/TensorCore/native/quantized/cpu/quantpack/wrappers/quant_sparse/8x4c1x4-packed-sse2.c",
		"7881299521365300    1 -rw-r-----   1 740376   89939         494 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/106/sample_654.png",
		"7318349568700189    1 -rw-r-----   1 740376   89939         377 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/132/sample_273.png",
		"19703248543174196    1 -rw-r-----   1 740376   89939         506 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/27/sample_17.png",
		"68116944537320215    1 -rw-r-----   1 740376   89939         358 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/150/sample_625.png",
		"29836347599531195   18 -rw-r--r--   1 root     root        17737 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/70252782/en.csv",
		"7318349568701171    1 -rw-r-----   1 740376   89939         452 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/27/sample_170.png",
		"29836347599531197   13 -rw-r--r--   1 root     root        13141 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/70252782/ko.csv",
		"29273397707200248  148 -rw-r--r--   1 root     root       151275 Sep 11  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/bak/data/SpeechFixture/train/left_0132a06d_4.pt",
		"18577348636263665    1 -rw-r-----   1 740376   89939         434 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/27/sample_171.png",
		"38280596900117006   16 -rw-r--r--   1 root     root        15600 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/70252782/en.csv.refine",
		"47850746169817761  148 -rw-r--r--   1 root     root       151275 Sep 11  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/bak/data/SpeechFixture/train/left_0135f3f2_0.pt",
		"19703248543174197    1 -rw-r-----   1 740376   89939         378 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/27/sample_172.png",
		"41095346667185320   13 -rw-r--r--   1 root     root        13021 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/70252782/ko.csv.refine",
		"65302194664290604   26 -rw-r--r--   1 root     root        26611 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/80057725/en.csv.refine",
		"71494644214371736  148 -rw-r--r--   1 root     root       151275 Sep 16  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/data/SpeechFixture/train/happy_b36c27c2_1.pt",
		"58546795329473290    1 -rw-r-----   1 740376   89939         390 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/186/sample_437.png",
		"58546795223531841   40 -rw-r--r--   1 root     root        40531 Oct 13  2021 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/translation-data/raw/media-corpus/multi-lang/tmp/80198188/ko.csv",
		"5629499663867342  148 -rw-r--r--   1 root     root       151275 Sep 16  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/data/SpeechFixture/train/happy_b36c27c2_2.pt",
		"17451448729534042    1 -rw-r-----   1 740376   89939         387 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/102/sample_745.png",
		"562950126822291    1 -rw-r-----   1 740376   89939         340 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/67/sample_168.png",
		"26458647940815421  148 -rw-r--r--   1 root     root       151275 Sep 16  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/data/SpeechFixture/train/happy_b36c27c2_3.pt",
		"5629499707506174    1 -rw-r-----   1 740376   89939         442 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/120/sample_170.png",
		"44473046493453068    1 -rw-r-----   1 740376   89939         440 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/121/sample_118.png",
		"58546795285700414  148 -rw-r--r--   1 root     root       151275 Sep 16  2022 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/tmp/speech-fixture/data/SpeechFixture/train/happy_b3849d6e_0.pt",
		"46161896354308072    1 -rw-r-----   1 740376   89939         382 Dec  1  2020 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/data/sequence-fixture/sequence_release/sequence_release/pathset32/curve_baseline/imgs/67/sample_169.png",
		"44473046493269710    6 -rw-r--r--   1 root     root         5420 Apr 28 23:02 /fixture_ro/sandbox/fixture-data/units/unit-a/sandbox/session-a/___backup___/00000000-0000-4000-8000-000000000000/2023-04-28T14:00:26Z/root/transformer_backend/build.bak/_deps/repo-transformer-src/3rdparty/matrixlib/docs/structmatrixlib_1_1gemm_1_1thread_1_1detail_1_1MatrixOp.html",
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

func TestScanner_executeFindCommand(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Scanner{}
	s.FinderBinaryPath = "find"
	infoChan := make(chan returns.Fileinfo)
	errChan := make(chan error, 1)
	go func() {
		defer close(infoChan)
		errChan <- s.executeFind(ctx, ".", infoChan)
	}()
	for entry := range infoChan {
		log.Printf("entry %v", entry)
	}
	if err := <-errChan; err != nil {
		t1.Fatal(err)
	}
}
