package provider

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const springGetConfigPath = "/config/"

type ConfigClient interface {
	GetConfig(string, string, string, string) (io.ReadCloser, error)
}

type SpringCloudConfigClient struct {
	client http.Client
}

func NewSpringCloudConfigClient() SpringCloudConfigClient {
	return SpringCloudConfigClient{
		client: http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *SpringCloudConfigClient) GetConfig(attributes Attributes) (io.ReadCloser, error) {
	fullAddress := attributes.serverAddress + springGetConfigPath + attributes.application + "/" + attributes.profile + "." + attributes.fileType
	r, err := c.client.Get(fullAddress)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v instead of 200 while calling %s", r.StatusCode, fullAddress)
	}
	return r.Body, nil
}
