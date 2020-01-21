package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

const springGetConfigPath = "/config/"

func NewProvider() Provider {
	log.Debug("New spring cloud config provider")
	return Provider{}
}

// Provider implements the secrets-store-csi-driver provider interface
type Provider struct {
	// the name of the pod (if using POD AAD Identity)
	PodName string
	// the namespace of the pod (if using POD AAD Identity)
	PodNamespace string
}

type ConfigClient interface {
	GetConfig(string, string, string, string) (io.ReadCloser, error)
}

func (p *Provider) MountSecretsStoreObjectContent(attrib map[string]string, targetPath string, permission os.FileMode, sccClient ConfigClient) (err error) {

	p.PodName = attrib["csi.storage.k8s.io/pod.name"]
	p.PodNamespace = attrib["csi.storage.k8s.io/pod.namespace"]

	serverAddress := attrib["serverAddress"]
	application := attrib["application"]
	profile := attrib["profile"]
	fileType := attrib["fileType"]

	if serverAddress == "" {
		return fmt.Errorf("serverAddress is not set")
	}

	if application == "" {
		return fmt.Errorf("application is not set")
	}

	if profile == "" {
		return fmt.Errorf("profile is not set")
	}

	if fileType == "" {
		return fmt.Errorf("fileType is not set")
	}

	log.Infof("mounting secrets store object content for %s/%s", p.PodNamespace, p.PodName)
	configName := fmt.Sprintf("%s-%s.%s", application, profile, fileType)

	content, err := sccClient.GetConfig(serverAddress, profile, application, fileType)
	if err != nil {
		return fmt.Errorf("failed to retrieve secrets for %s: %w", configName, err)
	}
	defer content.Close()

	file, err := os.OpenFile(path.Join(targetPath, configName), os.O_RDWR|os.O_CREATE, permission)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", configName, targetPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", configName, targetPath, err)
	}
	log.Infof("secrets store csi driver mounted %s", configName)
	log.Infof("mount point: %s", targetPath)
	return nil
}

type springCloudConfigClient struct {
	client http.Client
}

func newSpringCloudConfigClient() springCloudConfigClient {
	return springCloudConfigClient{
		client: http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *springCloudConfigClient) GetConfig(address, profile, application, fileType string) (io.ReadCloser, error) {
	fullAddress := address + springGetConfigPath + application + "/" + profile + "." + fileType
	r, err := c.client.Get(fullAddress)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v instead of 200 while calling %s", r.StatusCode, fullAddress)
	}
	return r.Body, nil
}
