package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/pkg/provider"
	log "github.com/sirupsen/logrus"
)

func main() {
	socketPath := filepath.Join(os.Getenv("TARGET_DIR"), "scc.sock")
	// Delete previous socket if exists
	_ = os.Remove(socketPath)

	server, err := provider.NewSpringCloudConfigCSIProviderServer(socketPath)
	if err != nil {
		log.Fatalf("error occured on server initialization: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	server.Start()

	sig := <-c
	log.Info(fmt.Sprintf("Caught signal %s, shutting down", sig))
	server.Stop()
}
