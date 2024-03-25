package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

type SpringCloudConfigCSIProviderServer struct {
	grpcServer              *grpc.Server
	listener                net.Listener
	socketPath              string
	returnErr               error
	errorCode               string
	springCloudConfigClient ConfigClient
}

type Attributes struct {
	ServerAddress string `json:"serverAddress,omitempty"`
	Application   string `json:"application,omitempty"`
	Profile       string `json:"profile,omitempty"`
	FileType      string `json:"fileType,omitempty"`
	Raw           string `json:"raw"`
}

type Raw struct {
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
}

func (a *Attributes) getRaw() (raw []Raw, err error) {
	if a.Raw == "" {
		return raw, nil
	}

	err = json.Unmarshal([]byte(a.Raw), &raw)
	if err != nil {
		return raw, fmt.Errorf("unmarshal attribute 'raw' from JSON: %w", err)
	}

	return
}

func (a *Attributes) verify() (err error) {
	var raw []Raw
	if a.Raw != "" {
		raw, err = a.getRaw()
		if err != nil {
			return err
		}
		for idx, item := range raw {
			if item.Source == "" || item.Target == "" {
				return fmt.Errorf("source or target not set for raw file on index %d", idx)
			}
		}
	}

	if a.ServerAddress == "" {
		return fmt.Errorf("serverAddress is not set")
	}

	if a.Application == "" {
		return fmt.Errorf("application is not set")
	}

	if a.Profile == "" {
		return fmt.Errorf("profile is not set")
	}

	// TODO might want to warn/info in-case only raw files were created
	if a.FileType == "" && len(raw) == 0 {
		return fmt.Errorf("FileType and raw are not set, atleast one is required")
	}

	return nil
}

// NewSpringCloudConfigCSIProviderServer returns CSI provider that uses the spring as the secret backend
func NewSpringCloudConfigCSIProviderServer(socketPath string, httpClient *http.Client, retryBaseWait time.Duration, retryMax uint64) (*SpringCloudConfigCSIProviderServer, error) {
	client := NewSpringCloudConfigClient(httpClient, retryBaseWait, retryMax)
	server := grpc.NewServer()
	s := &SpringCloudConfigCSIProviderServer{
		springCloudConfigClient: &client,
		grpcServer:              server,
		socketPath:              socketPath,
	}
	v1alpha1.RegisterCSIDriverProviderServer(server, s)
	return s, nil
}

func (m *SpringCloudConfigCSIProviderServer) Start() error {
	var err error
	m.listener, err = net.Listen("unix", m.socketPath)
	if err != nil {
		return err
	}
	go m.grpcServer.Serve(m.listener)
	return nil
}

func (m *SpringCloudConfigCSIProviderServer) Stop() {
	m.grpcServer.GracefulStop()
}

// Mount implements a provider csi-provider method
func (m *SpringCloudConfigCSIProviderServer) Mount(ctx context.Context, req *v1alpha1.MountRequest) (*v1alpha1.MountResponse, error) {
	var attrib Attributes
	var secret map[string]string
	var filePermission os.FileMode
	var err error

	if m.returnErr != nil {
		return &v1alpha1.MountResponse{}, m.returnErr
	}
	if err = json.Unmarshal([]byte(req.GetAttributes()), &attrib); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes, error: %+v", err)
	}
	if err = json.Unmarshal([]byte(req.GetSecrets()), &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets, error: %+v", err)
	}
	if err = json.Unmarshal([]byte(req.GetPermission()), &filePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file permission, error: %+v", err)
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, fmt.Errorf("missing target path")
	}

	err = attrib.verify()
	if err != nil {
		return nil, err
	}

	// TODO: needs to be refactored to actually reflect the object version
	// check out e. g. gcp provider https://github.com/GoogleCloudPlatform/secrets-store-csi-driver-provider-gcp/blob/main/server/server.go#L140
	// currently sets the object to the repo name and the version to the env
	objectVersions := []*v1alpha1.ObjectVersion{
		{
			Id:      attrib.Application,
			Version: attrib.Profile,
		},
	}

	out := &v1alpha1.MountResponse{
		ObjectVersion: objectVersions,
		Error: &v1alpha1.Error{
			Code: m.errorCode,
		},
	}

	if attrib.FileType != "" {
		err = m.mountFile(attrib, req.GetTargetPath(), filePermission)
		if err != nil {
			return nil, err
		}
	}

	raw, err := attrib.getRaw()
	if err != nil {
		return nil, err
	}
	for _, item := range raw {
		err = m.mountRawFile(attrib, item, req.GetTargetPath(), filePermission)
		if err != nil {
			return nil, err
		}
	}

	// Files should not exceed 1MiB
	return out, nil
}

// Version implements a provider csi-provider method
func (m *SpringCloudConfigCSIProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "springcloudconfigprovider",
		RuntimeVersion: "0.1.0",
	}, nil
}
func (m *SpringCloudConfigCSIProviderServer) mountFile(attrib Attributes, targetPath string, filePermission os.FileMode) error {
	fileName := fmt.Sprintf("%s-%s.%s", attrib.Application, attrib.Profile, attrib.FileType)
	content, err := m.springCloudConfigClient.GetConfig(attrib)
	if err != nil {
		return fmt.Errorf("failed to retrieve secrets for %s: %w", fileName, err)
	}
	defer content.Close()

	file, err := os.OpenFile(path.Join(targetPath, fileName), os.O_RDWR|os.O_CREATE, filePermission)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", fileName, targetPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", fileName, targetPath, err)
	}
	log.Infof("secrets store csi driver mounted %s", fileName)
	log.Infof("mount point: %s", targetPath)
	return nil
}

func (m *SpringCloudConfigCSIProviderServer) mountRawFile(attrib Attributes, item Raw, targetPath string, filePermission os.FileMode) error {
	content, err := m.springCloudConfigClient.GetConfigRaw(attrib, item.Source)
	if err != nil {
		return fmt.Errorf("failed to retrieve raw secrets for %s with path %s: %w", attrib.Application, item.Source, err)
	}
	defer content.Close()

	file, err := os.OpenFile(path.Join(targetPath, item.Target), os.O_RDWR|os.O_CREATE, filePermission)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount raw file %s for %s at %s: %w", item.Source, attrib.Application, item.Target, err)
	}
	defer file.Close()
	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("secrets store csi driver failed to mount raw file %s for %s at %s: %w", item.Source, attrib.Application, item.Target, err)
	}
	log.Infof("secrets store csi driver mounted raw file %s for %s at %s", item.Source, attrib.Application, item.Target)
	return nil
}
