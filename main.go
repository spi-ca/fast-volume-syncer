package main

import (
	"amuz.es/src/spi-ca/ch-boot/internal"
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// GOGC=100
// GOMEMLIMIT=32Mib
func main() {
	if len(os.Args) != 2 {
		log.Fatalf("args: %s [socket_path] %v", filepath.Base(os.Args[0]), os.Args)
	}

	c := internal.NewNodeClient(os.Args[1])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errorChan := make(chan error, 1)
	go internal.NodeStatusChecker(ctx, c, internal.NodeStatusRunning, errorChan)
	for err := range errorChan {
		log.Printf("err %v", err)
	}

	log.Printf("initiated shutdown")
}
