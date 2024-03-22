package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func TestSpringCloudConfigCSIProviderServer_Mount(t *testing.T) {
	type configServerRequest struct {
		path            string
		statusCode      int
		responsePayload string
	}

	testcases := []struct {
		name                 string
		configServerRequests []*configServerRequest
		attrib               Attributes
		wantFiles            map[string]string
		expected             string
		wantRawContent       map[string]string
		wantError            error
	}{
		{
			name: "When all attributes are passed and correctly then it creates the secret file",
			configServerRequests: []*configServerRequest{
				{
					path:            "/config/some/testing.json",
					statusCode:      200,
					responsePayload: `{"some":"json"}`,
				},
			},
			attrib: Attributes{
				ServerAddress: "http://configserver.localhost",
				Profile:       "testing",
				Application:   "some",
				FileType:      "json",
			},
			wantFiles: map[string]string{"some-testing.json": `{"some":"json"}`},
		},
		{
			name: "When raw files are part of the attributes then it creates the raw files",
			configServerRequests: []*configServerRequest{
				{
					path:            "/springconfig/some/testing/master/abc.conf",
					statusCode:      200,
					responsePayload: "content abc.def",
				},
			},
			attrib: Attributes{
				ServerAddress: "http://configserver.localhost",
				Profile:       "testing",
				Application:   "some",
				Raw:           `[{"source":"abc.conf","target":"def.conf"}]`,
			},
			wantFiles: map[string]string{"def.conf": "content abc.def"},
		},
		{
			name: "When an attribute is not set then an error is returned",
			attrib: Attributes{
				ServerAddress: "http://configserver.localhost",
				Profile:       "testing",
				Application:   "some",
			},
			wantError: errors.New("FileType and raw are not set, atleast one is required"),
		},
		{
			name: "When ConfigServer returns an error then it errors",
			configServerRequests: []*configServerRequest{
				{path: "/config/some/testing.json",
					statusCode:      500,
					responsePayload: "an error occurred",
				},
			},
			attrib: Attributes{
				ServerAddress: "http://configserver.localhost",
				Profile:       "testing",
				Application:   "some",
				FileType:      "json",
			},
			wantError: errors.New("failed to retrieve secrets for some-testing.json: received 500 instead of 200 while calling http://configserver.localhost/config/some/testing.json"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			defer gock.Off()
			//gock.Observe(gock.DumpRequest)
			for _, mockRequest := range tc.configServerRequests {
				gock.New(tc.attrib.ServerAddress).
					Get(mockRequest.path).
					Reply(mockRequest.statusCode).
					BodyString(mockRequest.responsePayload)
			}

			dir, err := os.MkdirTemp("", "scc-secrets-store-unittest")
			if err != nil {
				t.Fatal(err)
			}

			httpClient := createHttpClient()
			provider, _ := NewSpringCloudConfigCSIProviderServer(filepath.Join(dir, "scc.sock"), httpClient)

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

			if tc.wantError != nil {
				assert.EqualError(t, err, tc.wantError.Error(), tc.name)
			} else {
				assert.NoError(t, err)
				entries, err := os.ReadDir(dir)
				require.NoError(t, err)
				actualFiles := map[string]string{}
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					content, err := os.ReadFile(path.Join(dir, entry.Name()))
					require.NoError(t, err)
					actualFiles[entry.Name()] = string(content)
				}

				assert.Equal(t, tc.wantFiles, actualFiles)
			}

			require.True(t, gock.IsDone())
		})
	}
}

func createHttpClient() *http.Client {
	c := &http.Client{}
	gock.InterceptClient(c)
	return c
}
