package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/pkg/provider"
	log "github.com/sirupsen/logrus"
)

func main() {
	var retryBaseWait time.Duration
	var retryMax int64
	flag.DurationVar(&retryBaseWait, "retry-base-wait", 1*time.Second, "Duration to wait for the exponential retry algorithm before retrying a request to Config Server.")
	flag.Int64Var(&retryMax, "retry-max", 0, "Max number of retries in case Config Server responds with an error. 0 disables retries.")
	flag.Parse()
	socketPath := filepath.Join(os.Getenv("TARGET_DIR"), "spring-cloud-config.sock")
	// Delete previous socket if exists
	_ = os.Remove(socketPath)

	server, err := provider.NewSpringCloudConfigCSIProviderServer(socketPath, nil, retryBaseWait, uint64(retryMax))
	if err != nil {
		log.Fatalf("error occured on server initialization: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	err = server.Start()
	if err != nil {
		log.Fatalf("error occured on server start: %v", err)
	}

	sig := <-c
	log.Info(fmt.Sprintf("Caught signal %s, shutting down", sig))
	server.Stop()
}
