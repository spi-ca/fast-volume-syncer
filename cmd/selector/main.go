package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer"
)

const (
	name = "syncer"
)

func parseEnviron(defaultMap map[string]string) {
	// parse environs
	viper.SetEnvPrefix(name)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	for k, v := range defaultMap {
		viper.SetDefault(k, v)
	}
	log.Println("environ loaded")
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	name := fmt.Sprintf("%s[%d]", name, os.Getpid())

	log.SetPrefix(name)
	t := &syncer.rsyncTask{}
	t.ChunkSize = 1000
	t.FindCommandPath = "find"

	err := t.prepare(ctx)
	if err != nil {
		log.Printf("error occured : %v", err)
	}

}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	parseEnviron(map[string]string{})

	name := fmt.Sprintf("%s[%d]", name, os.Getpid())

	""
	log.SetPrefix(name)
	t := &syncer.rsyncTask{
		Source: syncer.MountInfo{
			StorageInfo: syncer.StorageInfo{
				Host:       "",
				MountPoint: "",
				Options:    "",
			},
			Volume: "",
		},
		Destination: syncer.MountInfo{
			StorageInfo: syncer.StorageInfo{
				Host:       "",
				MountPoint: "",
				Options:    "",
			},
			Volume: "",
		},
		TaskSize:        0,
		ChunkSize:       0,
		FindCommandPath: "",
		Args: syncer.RsyncArgs{
			Verbose:            false,
			PreservePermission: false,
			PreserveOwnership:  false,
			CopySpecial:        false,
			Compress:           false,
			WholeFile:          false,
			Inplace:            false,
			DryRun:             false,
			Recursive:          false,
		},
	}

	config.ParseEnviron()
	config.SetRandSeed()

	for k, v := range defaultMap {
		viper.SetDefault(k, v)
	}
	log.Println("environ loaded")

	ctx, canceler := context.WithCancel(context.Background())
	defer canceler()

	loc := &k8s.Location{
		Namespace:       viper.GetString("namespace"),
		Name:            viper.GetString("name"),
		ApplicationName: viper.GetString("app_name"),
		Owner:           k8s.GetPodOwnerReference(viper.GetString("name"), viper.GetString("uid"), true, true),
		NodeSelector:    k8s.ParseNodeSelector(viper.GetString("node_selector")),
	}
	if cleanup, err := config.SetLogOutput(viper.GetString("report_directory"), loc.Name); err != nil {
		log.Print(err.Error())
	} else {
		defer cleanup()
	}

	log.Printf("this application has located in %s", loc.String())

	c, err := k8s.NewClient(loc)
	if err != nil {
		log.Fatalf("failed to initialize k8s client: %v", err)
	}

	volmon, err := volume.NewMonitor(c, loc)
	if err != nil {
		log.Fatalf("failed to initialize a volume monitor: %v", err)
	}
	volmon.Start(ctx)

	defer volmon.Stop()

	podmon, err := tasklet.NewPodMonitor(c, loc)
	if err != nil {
		log.Fatalf("failed to initialize a pod monitor: %v", err)
	}
	podmon.Start(ctx)
	defer podmon.Stop()

	fixtureTool := &fixturetool.Task{
		Context:                       ctx,
		Location:                      loc,
		Client:                        c,
		ReportDirectory:               filepath.Join(viper.GetString("report_directory"), "syncer"),
		ScratchDirectory:              viper.GetString("scratch_directory"),
		ScratchClaimSize:              viper.GetString("scratch_claim_size"),
		ScratchVolumeStorageClassName: viper.GetString("scratch_storage_class_name"),
		TaskletImage:                  viper.GetString("tasklet_image"),
		TaskletSize:                   viper.GetInt("tasklet_size"),
		FioExecutable:                 viper.GetString("fio_executable"),
		VolumeMonitor:                 volmon,
		TaskletMonitor:                podmon,
	}

	// load tests
	tests, err := fixturetool.LoadTests(viper.GetString("test_directory"))
	if err != nil {
		log.Fatalf("failed to load test(s): %v", err)
	}

	rand.Shuffle(len(tests), func(i, j int) { tests[i], tests[j] = tests[j], tests[i] })
	log.Printf("loaded %d test(s)", len(tests))

	// iterate tests
	failed := 0
	for i, t := range tests {
		log.Printf("start %dth test: %s(%s)", i+1, t.Name, t.JobPath)
		err = fixtureTool.Play(t)
		if err != nil {
			log.Printf("failed %dth scenario(%s): %v", i+1, t.Name, err)
			failed++
			continue
		}
		log.Printf("completed %dth test", i+1)
	}

	log.Printf("completed: tot(%d) success(%d) failed(%d)", len(tests), len(tests)-failed, failed)
}
