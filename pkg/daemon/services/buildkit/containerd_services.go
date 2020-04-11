package buildkit

import (
	"context"

	"github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/containerd/api/services/diff/v1"
	"github.com/containerd/containerd/api/services/images/v1"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
)

var _ containers.ContainersClient = &nsContainersClient{}
var _ diff.DiffClient = &nsDiffClient{}
var _ images.ImagesClient = &nsImagesClient{}
var _ tasks.TasksClient = &nsTasksClient{}

type nsContainersClient struct {
	n string
	w containers.ContainersClient
}

func (n nsContainersClient) Get(ctx context.Context, in *containers.GetContainerRequest, opts ...grpc.CallOption) (*containers.GetContainerResponse, error) {
	return n.w.Get(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsContainersClient) List(ctx context.Context, in *containers.ListContainersRequest, opts ...grpc.CallOption) (*containers.ListContainersResponse, error) {
	return n.w.List(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsContainersClient) ListStream(ctx context.Context, in *containers.ListContainersRequest, opts ...grpc.CallOption) (containers.Containers_ListStreamClient, error) {
	return n.w.ListStream(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsContainersClient) Create(ctx context.Context, in *containers.CreateContainerRequest, opts ...grpc.CallOption) (*containers.CreateContainerResponse, error) {
	return n.w.Create(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsContainersClient) Update(ctx context.Context, in *containers.UpdateContainerRequest, opts ...grpc.CallOption) (*containers.UpdateContainerResponse, error) {
	return n.w.Update(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsContainersClient) Delete(ctx context.Context, in *containers.DeleteContainerRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Delete(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

type nsDiffClient struct {
	n string
	w diff.DiffClient
}

func (n nsDiffClient) Apply(ctx context.Context, in *diff.ApplyRequest, opts ...grpc.CallOption) (*diff.ApplyResponse, error) {
	return n.w.Apply(namespaces.WithNamespace(ctx, n.n), in, opts...)
}
func (n nsDiffClient) Diff(ctx context.Context, in *diff.DiffRequest, opts ...grpc.CallOption) (*diff.DiffResponse, error) {
	return n.w.Diff(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

type nsImagesClient struct {
	n string
	w images.ImagesClient
}

func (n nsImagesClient) Get(ctx context.Context, in *images.GetImageRequest, opts ...grpc.CallOption) (*images.GetImageResponse, error) {
	return n.w.Get(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsImagesClient) List(ctx context.Context, in *images.ListImagesRequest, opts ...grpc.CallOption) (*images.ListImagesResponse, error) {
	return n.w.List(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsImagesClient) Create(ctx context.Context, in *images.CreateImageRequest, opts ...grpc.CallOption) (*images.CreateImageResponse, error) {
	return n.w.Create(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsImagesClient) Update(ctx context.Context, in *images.UpdateImageRequest, opts ...grpc.CallOption) (*images.UpdateImageResponse, error) {
	return n.w.Update(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsImagesClient) Delete(ctx context.Context, in *images.DeleteImageRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Delete(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

type nsTasksClient struct {
	n string
	w tasks.TasksClient
}

func (n nsTasksClient) Create(ctx context.Context, in *tasks.CreateTaskRequest, opts ...grpc.CallOption) (*tasks.CreateTaskResponse, error) {
	return n.w.Create(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Start(ctx context.Context, in *tasks.StartRequest, opts ...grpc.CallOption) (*tasks.StartResponse, error) {
	return n.w.Start(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Delete(ctx context.Context, in *tasks.DeleteTaskRequest, opts ...grpc.CallOption) (*tasks.DeleteResponse, error) {
	return n.w.Delete(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) DeleteProcess(ctx context.Context, in *tasks.DeleteProcessRequest, opts ...grpc.CallOption) (*tasks.DeleteResponse, error) {
	return n.w.DeleteProcess(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Get(ctx context.Context, in *tasks.GetRequest, opts ...grpc.CallOption) (*tasks.GetResponse, error) {
	return n.w.Get(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) List(ctx context.Context, in *tasks.ListTasksRequest, opts ...grpc.CallOption) (*tasks.ListTasksResponse, error) {
	return n.w.List(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Kill(ctx context.Context, in *tasks.KillRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Kill(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Exec(ctx context.Context, in *tasks.ExecProcessRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Exec(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) ResizePty(ctx context.Context, in *tasks.ResizePtyRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.ResizePty(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) CloseIO(ctx context.Context, in *tasks.CloseIORequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.CloseIO(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Pause(ctx context.Context, in *tasks.PauseTaskRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Pause(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Resume(ctx context.Context, in *tasks.ResumeTaskRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Resume(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) ListPids(ctx context.Context, in *tasks.ListPidsRequest, opts ...grpc.CallOption) (*tasks.ListPidsResponse, error) {
	return n.w.ListPids(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Checkpoint(ctx context.Context, in *tasks.CheckpointTaskRequest, opts ...grpc.CallOption) (*tasks.CheckpointTaskResponse, error) {
	return n.w.Checkpoint(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Update(ctx context.Context, in *tasks.UpdateTaskRequest, opts ...grpc.CallOption) (*types.Empty, error) {
	return n.w.Update(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Metrics(ctx context.Context, in *tasks.MetricsRequest, opts ...grpc.CallOption) (*tasks.MetricsResponse, error) {
	return n.w.Metrics(namespaces.WithNamespace(ctx, n.n), in, opts...)
}

func (n nsTasksClient) Wait(ctx context.Context, in *tasks.WaitRequest, opts ...grpc.CallOption) (*tasks.WaitResponse, error) {
	return n.w.Wait(namespaces.WithNamespace(ctx, n.n), in, opts...)
}
