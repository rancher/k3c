package action

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type UninstallBuilder struct {
}

func (_ *UninstallBuilder) Namespace(ctx context.Context, k *client.Interface) error {
	ns, err := k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ns.Labels == nil || ns.Labels["app.kubernetes.io/managed-by"] != "k3c" {
		return errors.Errorf("namespace not managed by k3c")
	}
	deletePropagation := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}
	// is there a better way to wait for the namespace to actually be deleted?
	done := make(chan struct{})
	informer := k.Core.Namespace().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			close(done)
		},
	})
	go informer.Run(done)
	err = k.Core.Namespace().Delete(k.Namespace, &deleteOptions)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			_, err = k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
			if !apierr.IsNotFound(err) {
				continue
			}
			return nil
		}
	}
}

func (a *UninstallBuilder) NodeRole(_ context.Context, k *client.Interface) error {
	nodeList, err := k.Core.Node().List(metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/builder",
	})
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		if err = retry.RetryOnConflict(retry.DefaultRetry, removeBuilderRole(k, node.Name)); err != nil {
			logrus.Warnf("failed to remove builder label from %s", node.Name)
		}
	}
	return nil
}

func removeBuilderRole(k *client.Interface, nodeName string) func() error {
	return func() error {
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			node, err := k.Core.Node().Get(nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if node.Labels == nil {
				return nil
			}
			delete(node.Labels, "node-role.kubernetes.io/builder")
			_, err = k.Core.Node().Update(node)
			return err
		})
	}
}
