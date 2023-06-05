package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
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
	var nFlag = flag.Int("n", 1234, "help message for flag n")
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
