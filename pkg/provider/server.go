package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"

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
	objects                 []*v1alpha1.ObjectVersion
	files                   []*v1alpha1.File
}

type Attributes struct {
	ServerAddress string `json:"server_address,omitempty"`
	Application   string `json:"application,omitempty"`
	Profile       string `json:"profile,omitempty"`
	FileType      string `json:"file_type,omitempty"`
}

func (a *Attributes) verify() error {
	if a.ServerAddress == "" {
		return fmt.Errorf("serverAddress is not set")
	}

	if a.Application == "" {
		return fmt.Errorf("application is not set")
	}

	if a.Profile == "" {
		return fmt.Errorf("profile is not set")
	}

	if a.FileType == "" {
		return fmt.Errorf("FileType is not set")
	}

	return nil
}

// NewSpringCloudConfigCSIProviderServer returns CSI provider that uses the spring as the secret backend
func NewSpringCloudConfigCSIProviderServer(socketPath string) (*SpringCloudConfigCSIProviderServer, error) {
	client := NewSpringCloudConfigClient()
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

	fileName := fmt.Sprintf("%s-%s.%s", attrib.Application, attrib.Profile, attrib.FileType)
	content, err := m.springCloudConfigClient.GetConfig(attrib)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secrets for %s: %w", fileName, err)
	}
	defer content.Close()

	out := &v1alpha1.MountResponse{
		ObjectVersion: m.objects,
		Error: &v1alpha1.Error{
			Code: m.errorCode,
		},
		Files: m.files,
	}
	file, err := os.OpenFile(path.Join(req.GetTargetPath(), fileName), os.O_RDWR|os.O_CREATE, filePermission)
	if err != nil {
		return nil, fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", fileName, req.GetTargetPath(), err)
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return nil, fmt.Errorf("secrets store csi driver failed to mount %s at %s: %w", fileName, req.GetTargetPath(), err)
	}
	log.Infof("secrets store csi driver mounted %s", fileName)
	log.Infof("mount point: %s", req.GetTargetPath())

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
