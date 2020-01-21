package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type configClientMock struct {
	configResult string
	err          error
}

func (c *configClientMock) GetConfig(address, profile, application, fileType string) (io.ReadCloser, error) {
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
		attrib        map[string]string
		expected      string
		expectedError error
		clientError   error
	}{
		{
			name: "When all attributes are passed and correct then it creates the secret file",
			attrib: map[string]string{
				"serverAddress": "http://example.com/",
				"profile":       "testing",
				"application":   "some",
				"fileType":      "json",
			},
			expected: "{\"some\":\"json\"}",
		},
		{
			name: "When an attribute is not set then an error is returned",
			attrib: map[string]string{
				"serverAddress": "http://example.com/",
				"profile":       "testing",
				"application":   "some",
			},
			expectedError: errors.New("fileType is not set"),
		},
		{
			name: "When the serverAddress is wrong then an error is returned",
			attrib: map[string]string{
				"serverAddress": "example.com/",
				"profile":       "testing",
				"application":   "some",
				"fileType":      "json",
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
		targetPath := path.Join(dir, "some-testing.json")
		sccMock := newConfigClientMock(tc.expected, tc.clientError)
		provider := NewProvider()

		fmt.Println(targetPath)
		err = provider.MountSecretsStoreObjectContent(tc.attrib, dir, 0777, &sccMock)
		if tc.expectedError != nil {
			assert.EqualError(t, tc.expectedError, err.Error(), tc.name)
		} else {
			assert.NoError(t, err, tc.name)
			actual, err := ioutil.ReadFile(targetPath)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.expected, string(actual), tc.name)
		}
	}

}
