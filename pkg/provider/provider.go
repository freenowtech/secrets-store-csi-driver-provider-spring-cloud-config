package provider

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	springGetConfigPath    = "/config/"
	springRawGetConfigPath = "/springconfig/"
	defaultConfigBranch    = "master"
)

type ConfigClient interface {
	GetConfig(attributes Attributes) (io.ReadCloser, error)
	GetConfigRaw(attributes Attributes, source string) (io.ReadCloser, error)
}

type SpringCloudConfigClient struct {
	client *http.Client
}

func NewSpringCloudConfigClient(c *http.Client) SpringCloudConfigClient {
	if c == nil {
		c = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return SpringCloudConfigClient{
		client: c,
	}
}

// GetConfig pulls the config from spring-cloud config server and the server parses it to the specified format
// if the config contains secrets they are decoded on the server side
func (c *SpringCloudConfigClient) GetConfig(attributes Attributes) (io.ReadCloser, error) {
	fullAddress := attributes.ServerAddress + springGetConfigPath + attributes.Application + "/" + attributes.Profile + "." + attributes.FileType
	r, err := c.client.Get(fullAddress)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v instead of 200 while calling %s", r.StatusCode, fullAddress)
	}
	return r.Body, nil
}

// GetConfigRaw pulls the config from spring-cloud-config without parsing or decrypting the secrets
func (c *SpringCloudConfigClient) GetConfigRaw(attributes Attributes, source string) (io.ReadCloser, error) {
	fullAddress := attributes.ServerAddress + springRawGetConfigPath + attributes.Application + "/" + attributes.Profile + "/" + defaultConfigBranch + "/" + source
	r, err := c.client.Get(fullAddress)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v instead of 200 while calling %s", r.StatusCode, fullAddress)
	}
	return r.Body, nil

}
