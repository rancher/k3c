package action

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	buildkit "github.com/moby/buildkit/client"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DoImagesFunc func(context.Context, imagesv1.ImagesClient) error
type DoControlFunc func(context.Context, *buildkit.Client) error

func DoImages(ctx context.Context, k8s *client.Interface, fn DoImagesFunc) error {
	addr, err := GetServiceAddress(ctx, k8s, "k3c")
	if err != nil {
		return err
	}
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(ctx, imagesv1.NewImagesClient(conn))
}

func DoControl(ctx context.Context, k8s *client.Interface, fn DoControlFunc) error {
	addr, err := GetServiceAddress(ctx, k8s, "buildkit")
	if err != nil {
		return err
	}
	bkc, err := buildkit.New(ctx, fmt.Sprintf("tcp://%s", addr))
	if err != nil {
		return err
	}
	defer bkc.Close()
	return fn(ctx, bkc)
}

func GetServiceAddress(_ context.Context, k8s *client.Interface, port string) (string, error) {
	// TODO handle multiple addresses
	endpoints, err := k8s.Core.Endpoints().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, sub := range endpoints.Subsets {
		if len(sub.Addresses) > 0 {
			for _, p := range sub.Ports {
				if p.Name == port {
					return net.JoinHostPort(sub.Addresses[0].IP, strconv.FormatInt(int64(p.Port), 10)), nil
				}
			}
		}
	}
	return "", errors.New("unknown service port")
}
