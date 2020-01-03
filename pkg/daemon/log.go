package daemon

import (
	"context"
	"time"

	"github.com/rancher/k3c/pkg/log"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type chanWriter struct {
	stderr bool
	c      chan log.Entry
}

func (c *chanWriter) Write(p []byte) (n int, err error) {
	c.c <- log.Entry{
		Stderr:  c.stderr,
		Message: p,
	}
	return len(p), nil
}

type wrapper struct {
	ctx context.Context
	svc pb.RuntimeServiceClient
}

func (w *wrapper) Version(apiVersion string) (*pb.VersionResponse, error) {
	panic("implement me")
}

func (w *wrapper) CreateContainer(podSandboxID string, config *pb.ContainerConfig, sandboxConfig *pb.PodSandboxConfig) (string, error) {
	panic("implement me")
}

func (w *wrapper) StartContainer(containerID string) error {
	panic("implement me")
}

func (w *wrapper) StopContainer(containerID string, timeout int64) error {
	panic("implement me")
}

func (w *wrapper) RemoveContainer(containerID string) error {
	panic("implement me")
}

func (w *wrapper) ListContainers(filter *pb.ContainerFilter) ([]*pb.Container, error) {
	panic("implement me")
}

func (w *wrapper) ContainerStatus(containerID string) (*pb.ContainerStatus, error) {
	resp, err := w.svc.ContainerStatus(w.ctx, &pb.ContainerStatusRequest{
		ContainerId: containerID,
	})
	return resp.GetStatus(), err
}

func (w *wrapper) UpdateContainerResources(containerID string, resources *pb.LinuxContainerResources) error {
	panic("implement me")
}

func (w *wrapper) ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error) {
	panic("implement me")
}

func (w *wrapper) Exec(*pb.ExecRequest) (*pb.ExecResponse, error) {
	panic("implement me")
}

func (w *wrapper) Attach(req *pb.AttachRequest) (*pb.AttachResponse, error) {
	panic("implement me")
}

func (w *wrapper) ReopenContainerLog(ContainerID string) error {
	panic("implement me")
}

func (w *wrapper) RunPodSandbox(config *pb.PodSandboxConfig, runtimeHandler string) (string, error) {
	panic("implement me")
}

func (w *wrapper) StopPodSandbox(podSandboxID string) error {
	panic("implement me")
}

func (w *wrapper) RemovePodSandbox(podSandboxID string) error {
	panic("implement me")
}

func (w *wrapper) PodSandboxStatus(podSandboxID string) (*pb.PodSandboxStatus, error) {
	panic("implement me")
}

func (w *wrapper) ListPodSandbox(filter *pb.PodSandboxFilter) ([]*pb.PodSandbox, error) {
	panic("implement me")
}

func (w *wrapper) PortForward(*pb.PortForwardRequest) (*pb.PortForwardResponse, error) {
	panic("implement me")
}

func (w *wrapper) ContainerStats(containerID string) (*pb.ContainerStats, error) {
	panic("implement me")
}

func (w *wrapper) ListContainerStats(filter *pb.ContainerStatsFilter) ([]*pb.ContainerStats, error) {
	panic("implement me")
}

func (w *wrapper) UpdateRuntimeConfig(runtimeConfig *pb.RuntimeConfig) error {
	panic("implement me")
}

func (w *wrapper) Status() (*pb.RuntimeStatus, error) {
	panic("implement me")
}
