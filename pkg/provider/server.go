package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

const springGetConfigPath = "/config/"

type SpringCloudConfigCSIProviderServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
	socketPath string
	returnErr  error
	errorCode  string
	objects    []*v1alpha1.ObjectVersion
	files      []*v1alpha1.File
}

// NewMocKCSIProviderServer returns a mock csi-provider grpc server
func NewSpringCloudConfigCSIProviderServer(socketPath string) (*SpringCloudConfigCSIProviderServer, error) {
	server := grpc.NewServer()
	s := &SpringCloudConfigCSIProviderServer{
		grpcServer: server,
		socketPath: socketPath,
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
	var attrib, secret map[string]string
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
	return &v1alpha1.MountResponse{
		ObjectVersion: m.objects,
		Error: &v1alpha1.Error{
			Code: m.errorCode,
		},
		Files: m.files,
	}, nil
}

// Version implements a provider csi-provider method
func (m *SpringCloudConfigCSIProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "springcloudconfigprovider",
		RuntimeVersion: "0.1.0",
	}, nil
}
