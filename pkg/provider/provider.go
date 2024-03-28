package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sethvargo/go-retry"
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
	backoff retry.Backoff
	client  *http.Client
}

func NewSpringCloudConfigClient(c *http.Client, retryBaseWait time.Duration, retryMax uint64) SpringCloudConfigClient {
	if c == nil {
		c = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	var b retry.Backoff
	if retryMax != 0 {
		b = retry.NewExponential(retryBaseWait)
		b = retry.WithMaxRetries(retryMax, b)
	}

	return SpringCloudConfigClient{
		backoff: b,
		client:  c,
	}
}

// GetConfig pulls the config from spring-cloud config server and the server parses it to the specified format
// if the config contains secrets they are decoded on the server side
func (c *SpringCloudConfigClient) GetConfig(attributes Attributes) (io.ReadCloser, error) {
	fullAddress := attributes.ServerAddress + springGetConfigPath + attributes.Application + "/" + attributes.Profile + attributes.extension()
	req, err := http.NewRequest("GET", fullAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("create new GetConfig request: %w", err)
	}

	r, err := c.do(req)
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
	req, err := http.NewRequest("GET", fullAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("create new GetConfigRaw request: %w", err)
	}

	r, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received %v instead of 200 while calling %s", r.StatusCode, fullAddress)
	}
	return r.Body, nil

}

func (c *SpringCloudConfigClient) do(req *http.Request) (*http.Response, error) {
	if c.backoff == nil {
		return c.client.Do(req)
	}

	var resp *http.Response
	err := retry.Do(context.Background(), c.backoff, func(ctx context.Context) error {
		var doErr error
		resp, doErr = c.client.Do(req)
		if doErr != nil {
			return retry.RetryableError(doErr)
		}

		if resp != nil && resp.StatusCode >= 500 {
			return retry.RetryableError(fmt.Errorf("request failed with status code %d", resp.StatusCode))
		}

		return nil
	})

	return resp, err
}
