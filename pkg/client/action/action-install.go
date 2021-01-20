package action

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/server"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

type InstallBuilder struct {
	Force    bool   `usage:"Force installation by deleting existing builder"`
	Selector string `usage:"Selector for nodes (label query) to apply builder role"`
	server.Config
}

func (_ *InstallBuilder) Namespace(_ context.Context, k *client.Interface) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns, err := k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: k.Namespace,
					Labels: labels.Set{
						"app.kubernetes.io/managed-by": "k3c",
					},
				},
			}
			ns, err = k.Core.Namespace().Create(ns)
			return err
		}
		if ns.Labels == nil {
			ns.Labels = labels.Set{}
		}
		if _, ok := ns.Labels["app.kubernetes.io/managed-by"]; !ok {
			ns.Labels["app.kubernetes.io/managed-by"] = "k3c"
		}
		ns, err = k.Core.Namespace().Update(ns)
		return err
	})
}

func (a *InstallBuilder) containerPort(name string) corev1.ContainerPort {
	switch name {
	case "buildkit":
		return corev1.ContainerPort{
			Name:          name,
			ContainerPort: int32(a.BuildkitPort),
			Protocol:      corev1.ProtocolTCP,
		}
	case "k3c":
		return corev1.ContainerPort{
			Name:          name,
			ContainerPort: int32(a.AgentPort),
			Protocol:      corev1.ProtocolTCP,
		}
	default:
		return corev1.ContainerPort{Name: name}
	}
}

func (a *InstallBuilder) servicePort(name string) corev1.ServicePort {
	switch name {
	case "buildkit":
		return corev1.ServicePort{
			Name:     name,
			Port:     int32(a.BuildkitPort),
			Protocol: corev1.ProtocolTCP,
		}
	case "k3c":
		return corev1.ServicePort{
			Name:     name,
			Port:     int32(a.AgentPort),
			Protocol: corev1.ProtocolTCP,
		}
	default:
		return corev1.ServicePort{Name: name}
	}
}

func (a *InstallBuilder) Service(_ context.Context, k *client.Interface) error {
	if a.Force {
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
		k.Core.Service().Delete(k.Namespace, "builder", &deleteOptions)
	}
	if a.AgentPort <= 0 {
		a.AgentPort = server.DefaultAgentPort
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		svc, err := k.Core.Service().Get(k.Namespace, "builder", metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			svc = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "builder",
					Namespace: k.Namespace,
					Labels: labels.Set{
						"app.kubernetes.io/managed-by": "k3c",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Selector: labels.Set{
						"app.kubernetes.io/name":      "k3c",
						"app.kubernetes.io/component": "builder",
					},
					Ports: []corev1.ServicePort{
						a.servicePort("buildkit"),
						a.servicePort("k3c"),
					},
				},
			}
			svc, err = k.Core.Service().Create(svc)
			return err
		}
		if svc.Labels == nil {
			svc.Labels = labels.Set{}
		}
		if _, ok := svc.Labels["app.kubernetes.io/managed-by"]; !ok {
			svc.Labels["app.kubernetes.io/managed-by"] = "k3c"
		}
		svc, err = k.Core.Service().Update(svc)
		return err
	})
}

func (a *InstallBuilder) DaemonSet(_ context.Context, k *client.Interface) error {
	if a.Force {
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
		k.Apps.DaemonSet().Delete(k.Namespace, "builder", &deleteOptions)
	}
	privileged := true
	hostPathDirectory := corev1.HostPathDirectory
	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	mountPropagationBidirectional := corev1.MountPropagationBidirectional
	containerProbe := corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"buildctl", "debug", "workers"},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       20,
	}

	daemon := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "builder",
			Namespace: k.Namespace,
			Labels: labels.Set{
				"app.kubernetes.io/name":       "k3c",
				"app.kubernetes.io/component":  "builder",
				"app.kubernetes.io/managed-by": "k3c",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.Set{
					"app":       "k3c",
					"component": "builder",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.Set{
						"app":                          "k3c",
						"component":                    "builder",
						"app.kubernetes.io/name":       "k3c",
						"app.kubernetes.io/component":  "builder",
						"app.kubernetes.io/managed-by": "k3c",
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					HostIPC:     true,
					NodeSelector: labels.Set{
						"node-role.kubernetes.io/builder": "true",
					},
					DNSPolicy: corev1.DNSClusterFirstWithHostNet,
					Containers: []corev1.Container{{
						Name:  "buildkit",
						Image: "moby/buildkit:v0.8.1",
						Args: []string{
							fmt.Sprintf("--addr=%s", a.BuildkitAddress),
							fmt.Sprintf("--addr=tcp://0.0.0.0:%d", a.BuildkitPort),
							"--containerd-worker=true",
							fmt.Sprintf("--containerd-worker-addr=%s", a.ContainerdAddress),
							"--containerd-worker-gc",
							"--oci-worker=false",
						},
						Ports: []corev1.ContainerPort{
							a.containerPort("buildkit"),
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "cgroup", MountPath: "/sys/fs/cgroup"},
							{Name: "run", MountPath: "/run", MountPropagation: &mountPropagationBidirectional},
							{Name: "tmp", MountPath: "/tmp", MountPropagation: &mountPropagationBidirectional},
							{Name: "var-lib-buildkit", MountPath: "/var/lib/buildkit", MountPropagation: &mountPropagationBidirectional},
							{Name: "var-lib-rancher", MountPath: "/var/lib/rancher", MountPropagation: &mountPropagationBidirectional},
						},
						ReadinessProbe: &containerProbe,
						LivenessProbe:  &containerProbe,
					}, {
						Name:    "agent",
						Image:   a.GetAgentImage(),
						Command: []string{"k3c", "--debug", "agent"},
						Args: []string{
							fmt.Sprintf("--agent-port=%d", a.AgentPort),
							fmt.Sprintf("--buildkit-address=%s", a.BuildkitAddress),
							fmt.Sprintf("--buildkit-port=%d", a.BuildkitPort),
							fmt.Sprintf("--containerd-address=%s", a.ContainerdAddress),
						},
						Ports: []corev1.ContainerPort{
							a.containerPort("k3c"),
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "run", MountPath: "/run", MountPropagation: &mountPropagationBidirectional},
							{Name: "var-lib-rancher", MountPath: "/var/lib/rancher", MountPropagation: &mountPropagationBidirectional},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "cgroup", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/fs/cgroup", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "run", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "tmp", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/tmp", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "var-lib-buildkit", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/buildkit", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "var-lib-rancher", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/rancher", Type: &hostPathDirectoryOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := k.Apps.DaemonSet().Create(daemon)
	if apierr.IsAlreadyExists(err) {
		return errors.Errorf("buildkit already installed, pass the --force option to recreate")
	}
	return err
}

func (a *InstallBuilder) NodeRole(_ context.Context, k *client.Interface) error {
	nodeList, err := k.Core.Node().List(metav1.ListOptions{
		LabelSelector: a.Selector,
	})
	if err != nil {
		return err
	}
	if len(nodeList.Items) == 1 {
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			node, err := k.Core.Node().Get(nodeList.Items[0].Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			node.Labels = labels.Merge(node.Labels, labels.Set{
				"node-role.kubernetes.io/builder": "true",
			})
			_, err = k.Core.Node().Update(node)
			return err
		})
	}
	return errors.Errorf("too many nodes, please specify a selector, e.g. k3s.io/hostname=%s", nodeList.Items[0].Name)
}
