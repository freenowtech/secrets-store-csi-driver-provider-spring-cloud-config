package provider

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
	"os"
	"path"

	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

type SpringCloudConfigCSIProviderServer struct {
	grpcServer              *grpc.Server
	listener                net.Listener
	socketPath              string
	returnErr               error
	errorCode               string
	springCloudConfigClient SpringCloudConfigClient
	objects                 []*v1alpha1.ObjectVersion
	files                   []*v1alpha1.File
}

type Attributes struct {
	serverAddress string
	application   string
	profile       string
	fileType      string
}

func (a *Attributes) verify() error {
	if a.serverAddress == "" {
		return fmt.Errorf("serverAddress is not set")
	}

	if a.application == "" {
		return fmt.Errorf("application is not set")
	}

	if a.profile == "" {
		return fmt.Errorf("profile is not set")
	}

	if a.fileType == "" {
		return fmt.Errorf("fileType is not set")
	}

	return nil
}

// NewMocKCSIProviderServer returns a mock csi-provider grpc server
func NewSpringCloudConfigCSIProviderServer(socketPath string) (*SpringCloudConfigCSIProviderServer, error) {
	client := NewSpringCloudConfigClient()
	server := grpc.NewServer()
	s := &SpringCloudConfigCSIProviderServer{
		springCloudConfigClient: client,
		grpcServer:              server,
		socketPath:              socketPath,
	}
	v1alpha1.RegisterCSIDriverProviderServer(server, s)
	return s, nil
}

// SetReturnError sets expected error
func (m *SpringCloudConfigCSIProviderServer) SetReturnError(err error) {
	m.returnErr = err
}

// SetObjects sets expected objects id and version
func (m *SpringCloudConfigCSIProviderServer) SetObjects(objects map[string]string) {
	var ov []*v1alpha1.ObjectVersion
	for k, v := range objects {
		ov = append(ov, &v1alpha1.ObjectVersion{Id: k, Version: v})
	}
	m.objects = ov
}

// SetFiles sets provider files to return on Mount
func (m *SpringCloudConfigCSIProviderServer) SetFiles(files []*v1alpha1.File) {
	var ov []*v1alpha1.File
	for _, v := range files {
		ov = append(ov, &v1alpha1.File{
			Path:     v.Path,
			Mode:     v.Mode,
			Contents: v.Contents,
		})
	}
	m.files = ov
}

// SetProviderErrorCode sets provider error code to return
func (m *SpringCloudConfigCSIProviderServer) SetProviderErrorCode(errorCode string) {
	m.errorCode = errorCode
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

	fileName := fmt.Sprintf("%s-%s.%s", attrib.application, attrib.profile, attrib.fileType)
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
	//out.Files = append(out.Files, &v1alpha1.File{
	//	Path:     path.Join(req.GetTargetPath(), fileName),
	//	Mode:     filePermission,
	//	Contents: nil,
	//})

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
