package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"

	"github.com/stretchr/testify/assert"
)

type configClientMock struct {
	configResult string
	err          error
}

func (c *configClientMock) GetConfig(_ Attributes) (io.ReadCloser, error) {
	if c.err != nil {
		return nil, c.err
	}
	return ioutil.NopCloser(strings.NewReader(c.configResult)), nil
}

func newConfigClientMock(expected string, err error) configClientMock {
	return configClientMock{
		configResult: expected,
		err:          err,
	}
}

func TestMountSecretsStoreObjectContent(t *testing.T) {
	testcases := []struct {
		name          string
		attrib        Attributes
		expected      string
		expectedError error
		clientError   error
	}{
		{
			name: "When all attributes are passed and correct then it creates the secret file",
			attrib: Attributes{
				ServerAddress: "http://example.com/",
				Profile:       "testing",
				Application:   "some",
				FileType:      "json",
			},
			expected: "{\"some\":\"json\"}",
		},
		{
			name: "When an attribute is not set then an error is returned",
			attrib: Attributes{
				ServerAddress: "http://example.com/",
				Profile:       "testing",
				Application:   "some",
			},
			expectedError: errors.New("FileType is not set"),
		},
		{
			name: "When the ServerAddress is wrong then an error is returned",
			attrib: Attributes{
				ServerAddress: "example.com/",
				Profile:       "testing",
				Application:   "some",
				FileType:      "json",
			},
			expectedError: errors.New("failed to retrieve secrets for some-testing.json: some error"),
			clientError:   errors.New("some error"),
		},
	}

	for _, tc := range testcases {
		dir, err := ioutil.TempDir("", "scc-secrets-store-unittest")
		if err != nil {
			t.Fatal(err)
		}
		file := path.Join(dir, "some-testing.json")
		sccMock := newConfigClientMock(tc.expected, tc.clientError)

		provider, _ := NewSpringCloudConfigCSIProviderServer(filepath.Join(dir, "scc.sock"))
		provider.springCloudConfigClient = &sccMock

		attributes, err := json.Marshal(tc.attrib)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := provider.Mount(context.TODO(), &v1alpha1.MountRequest{
			Attributes: string(attributes),
			Secrets:    "{\"some\":\"json\"}",
			TargetPath: dir,
			Permission: "777",
		})

		if resp != nil && resp.Error != nil && resp.Error.String() != "" {
			t.Fatal(resp.Error.String())
		}

		if tc.expectedError != nil {
			assert.EqualError(t, err, tc.expectedError.Error(), tc.name)
		} else {
			assert.NoError(t, err, tc.name)
			actual, err := ioutil.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.expected, string(actual), tc.name)
		}
	}

}
