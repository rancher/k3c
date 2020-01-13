package daemon

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/daemon/volume"
	"github.com/rancher/k3c/pkg/log"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/kuberuntime/logs"
)

const (
	AnnotationUserContainerConfig = "k3c.io/user-container-config"
	AnnotationContainerConfig     = "k3c.io/container-config"
	AnnotationCpus                = "k3c.io/cpus"
)

var (
	propNone            = v1.MountPropagationNone
	propHostToContainer = v1.MountPropagationHostToContainer
	propBidirectional   = v1.MountPropagationBidirectional
)

func (c *Daemon) Exec(ctx context.Context, containerName string, cmd []string, opts *v1alpha1.ExecOptions) (*client.StreamResponse, error) {
	if opts == nil {
		opts = &v1alpha1.ExecOptions{}
	}

	_, _, id, err := c.GetContainer(ctx, containerName)
	if err != nil {
		return nil, err
	}

	req := &pb.ExecRequest{
		ContainerId: id,
		Cmd:         cmd,
		Tty:         opts.Tty,
		Stdin:       opts.Stdin,
		Stdout:      true,
		Stderr:      !opts.Tty,
	}

	resp, err := c.runtime.Exec(ctx, req)
	if err != nil {
		return nil, err
	}

	return &client.StreamResponse{
		URL:   resp.Url,
		TTY:   req.Tty,
		Stdin: req.Stdin,
	}, nil
}

func (c *Daemon) LogContainer(ctx context.Context, containerID string, opts *v1.PodLogOptions) (<-chan log.Entry, error) {
	result := make(chan log.Entry)

	if opts == nil {
		opts = &v1.PodLogOptions{}
	}

	_, _, containerID, err := c.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	containers, err := c.runtime.ListContainers(ctx, &pb.ListContainersRequest{
		Filter: &pb.ContainerFilter{
			Id: containerID,
		},
	})
	if err != nil || len(containers.Containers) != 1 {
		close(result)
		return result, err
	}

	cfg, _, err := getContainerConfig(containers.Containers[0])
	if err != nil {
		return nil, err
	}

	pods, err := c.runtime.ListPodSandbox(ctx, &pb.ListPodSandboxRequest{
		Filter: &pb.PodSandboxFilter{
			Id: containers.Containers[0].PodSandboxId,
		},
	})
	if err != nil || len(pods.Items) != 1 {
		close(result)
		return result, err
	}

	podCfg, err := getPodConfig(pods.Items[0])
	if err != nil {
		return nil, err
	}

	if podCfg.LogDirectory == "" || cfg.LogPath == "" {
		close(result)
		return result, err
	}

	logPath := filepath.Join(podCfg.LogDirectory, cfg.LogPath)
	logOpts := logs.NewLogOptions(opts, time.Now())

	go func() {
		defer close(result)
		err := logs.ReadLogs(ctx, logPath, containerID, logOpts, &wrapper{
			ctx: ctx,
			svc: c.runtime,
		}, &chanWriter{
			stderr: false,
			c:      result,
		}, &chanWriter{
			stderr: true,
			c:      result,
		})
		if err != nil {
			logrus.Infof("error reading %s: %v", logPath, err)
		}
	}()

	return result, nil
}

func (c *Daemon) Attach(ctx context.Context, name string, opts *v1alpha1.AttachOptions) (*client.StreamResponse, error) {
	if opts == nil {
		opts = &v1alpha1.AttachOptions{}
	}

	_, container, containerID, err := c.GetContainer(ctx, name)
	if err != nil {
		return nil, err
	}

	request := &pb.AttachRequest{
		ContainerId: containerID,
		Tty:         container.TTY,
		Stdin:       container.Stdin,
		Stdout:      true,
		Stderr:      !container.TTY,
	}

	if opts.NoStdin {
		request.Stdin = false
	}

	resp, err := c.runtime.Attach(ctx, request)
	if err != nil {
		return nil, err
	}

	return &client.StreamResponse{
		URL:   resp.Url,
		TTY:   request.Tty,
		Stdin: request.Stdin,
	}, nil
}

func (c *Daemon) GetContainer(ctx context.Context, name string) (*v1.Pod, *v1.Container, string, error) {
	pods, err := c.listPods(ctx, true)
	if err != nil {
		return nil, nil, "", err
	}

	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == name || strings.HasPrefix(status.ContainerID, name) {
				for _, container := range pod.Spec.Containers {
					if container.Name == status.Name {
						return &pod, &container, status.ContainerID, nil
					}
				}
			}
		}
	}

	return nil, nil, "", client.ErrContainerNotFound
}

func (c *Daemon) RemoveContainer(ctx context.Context, containerID string) error {
	defer c.gcKick.Kick()

	_, _, containerID, err := c.GetContainer(ctx, containerID)
	if err != nil {
		return err
	}
	_, err = c.runtime.RemoveContainer(ctx, &pb.RemoveContainerRequest{
		ContainerId: containerID,
	})

	return err
}

func (c *Daemon) StartContainer(ctx context.Context, containerID string) error {
	_, _, containerID, err := c.GetContainer(ctx, containerID)
	if err != nil {
		return err
	}
	_, err = c.runtime.StartContainer(ctx, &pb.StartContainerRequest{
		ContainerId: containerID,
	})
	return err
}

func (c *Daemon) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	_, _, containerID, err := c.GetContainer(ctx, containerID)
	if err != nil {
		return err
	}
	_, err = c.runtime.StopContainer(ctx, &pb.StopContainerRequest{
		ContainerId: containerID,
		Timeout:     timeout,
	})
	return err
}

func (c *Daemon) resolveMounts(ctx context.Context, mounts []*pb.Mount) (result []*pb.Mount, err error) {
	for _, m := range mounts {
		m, err := c.Setup(ctx, m)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (c *Daemon) CreateContainer(ctx context.Context, podID, image string, opts *v1alpha1.ContainerOptions) (string, error) {
	if opts == nil {
		opts = &v1alpha1.ContainerOptions{}
	}

	pods, err := c.runtime.ListPodSandbox(ctx, &pb.ListPodSandboxRequest{
		Filter: &pb.PodSandboxFilter{
			Id: podID,
		},
	})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("failed to find pod ID %s", podID)
	}

	pod := pods.Items[0]
	podConfig, err := getPodConfig(pod)
	if err != nil {
		return "", errors.Wrap(err, "getting pod config")
	}

	name := opts.Name
	if name == "" {
		name = pod.Metadata.Name
	}

	username, user := toUserID(opts.GetSecurityContext().GetUser())

	group, err := toID(opts.GetSecurityContext().GetGroup())
	if err != nil {
		return "", errors.Wrap(err, "parsing group")
	}

	groups, err := toGroupIDs(opts.GetSecurityContext().GetGroups())
	if err != nil {
		return "", errors.Wrap(err, "parsing groups")
	}

	logPath := filepath.Join(name, fmt.Sprintf("%d.log", opts.Attempt))
	if podConfig.LogDirectory != "" {
		os.MkdirAll(filepath.Join(podConfig.LogDirectory, name), 0700)
	}

	mounts, err := c.resolveMounts(ctx, opts.Mounts)
	if err != nil {
		return "", err
	}

	config := &pb.ContainerConfig{
		Metadata: &pb.ContainerMetadata{
			Name:    name,
			Attempt: opts.Attempt,
		},
		Image: &pb.ImageSpec{
			Image: image,
		},
		Command:    opts.Command,
		Args:       opts.Args,
		WorkingDir: opts.WorkingDir,
		Envs:       opts.Envs,
		Mounts:     mounts,
		Devices:    opts.Devices,
		Labels:     opts.Labels,
		LogPath:    logPath,
		Stdin:      opts.Stdin,
		StdinOnce:  opts.StdinOnce,
		Tty:        opts.Tty,
		Linux: &pb.LinuxContainerConfig{
			Resources: opts.LinuxResources,
			SecurityContext: &pb.LinuxContainerSecurityContext{
				Capabilities: &pb.Capability{
					AddCapabilities:  opts.AddCapabilities,
					DropCapabilities: opts.DropCapabilities,
				},
				Privileged: opts.GetSecurityContext().GetPrivileged(),
				NamespaceOptions: &pb.NamespaceOption{
					Network: opts.GetSecurityContext().GetNetMode(),
					Pid:     opts.GetSecurityContext().GetPidMode(),
					Ipc:     opts.GetSecurityContext().GetIpcMode(),
				},
				SelinuxOptions:     opts.GetSecurityContext().GetSelinuxOptions(),
				RunAsUser:          user,
				RunAsGroup:         group,
				RunAsUsername:      username,
				ReadonlyRootfs:     opts.GetSecurityContext().GetReadonlyRoot(),
				SupplementalGroups: groups,
				ApparmorProfile:    "",
				SeccompProfilePath: opts.GetSecurityContext().GetSeccompProfile(),
				NoNewPrivs:         opts.NoNewPrivs,
				MaskedPaths:        opts.MaskedPaths,
				ReadonlyPaths:      opts.ReadonlyPaths,
			},
		},
	}

	if err := storeContainerConfig(config, opts); err != nil {
		return "", err
	}

	container, err := c.runtime.CreateContainer(ctx, &pb.CreateContainerRequest{
		PodSandboxId:  podID,
		Config:        config,
		SandboxConfig: podConfig,
	})
	if err != nil {
		return "", err
	}

	return container.ContainerId, nil
}

func toGroupIDs(groups []string) (result []int64, err error) {
	for _, group := range groups {
		id, err := toID(group)
		if err != nil {
			return nil, err
		}
		result = append(result, id.Value)
	}

	return
}

func toUserID(val string) (string, *pb.Int64Value) {
	if val == "" || val == "root" {
		return val, nil
	}

	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return val, nil
	}
	return "", &pb.Int64Value{
		Value: n,
	}
}

func storeContainerConfig(config *pb.ContainerConfig, opts *v1alpha1.ContainerOptions) error {
	m := jsonpb.Marshaler{}
	buf := &bytes.Buffer{}
	if err := m.Marshal(buf, config); err != nil {
		return errors.Wrap(err, "marshal pod config")
	}

	if config.Annotations == nil {
		config.Annotations = map[string]string{}
	}
	config.Annotations[AnnotationContainerConfig] = buf.String()

	buf.Reset()
	if err := m.Marshal(buf, opts); err != nil {
		return errors.Wrap(err, "marshal container opts")
	}
	config.Annotations[AnnotationUserContainerConfig] = buf.String()
	return nil
}

func getContainerConfig(pod *pb.Container) (*pb.ContainerConfig, *v1alpha1.ContainerOptions, error) {
	var (
		config pb.ContainerConfig
		opts   v1alpha1.ContainerOptions
	)
	anno := pod.Annotations[AnnotationContainerConfig]
	if err := jsonpb.UnmarshalString(anno, &config); err != nil {
		return nil, nil, err
	}

	anno = pod.Annotations[AnnotationUserContainerConfig]
	if err := jsonpb.UnmarshalString(anno, &opts); err != nil {
		return nil, nil, err
	}

	return &config, &opts, nil
}

type containerData struct {
	container *pb.Container
	info      map[string]string
	status    *pb.ContainerStatus
}

func toEnv(containerConfig *pb.ContainerConfig) (result []v1.EnvVar) {
	for _, env := range containerConfig.Envs {
		result = append(result, v1.EnvVar{
			Name:  env.Key,
			Value: env.Value,
		})
	}

	return
}

func toResources(podID string, containerConfig *pb.ContainerConfig) (result v1.ResourceRequirements) {
	mem := containerConfig.GetLinux().GetResources().GetMemoryLimitInBytes()
	if mem > 0 {
		result.Requests = v1.ResourceList{}
		result.Requests[v1.ResourceMemory] = *resource.NewQuantity(mem, resource.BinarySI)
	}
	cpu := containerConfig.GetAnnotations()[AnnotationCpus]
	if cpu != "" {
		cpuMillis, err := strconv.ParseInt(cpu, 10, 64)
		if err == nil {
			if result.Requests == nil {
				result.Requests = v1.ResourceList{}
			}

			cpuShares := (cpuMillis * 1024) / 1000
			if cpuShares < 2 {
				cpuShares = 2
			}
			result.Requests[v1.ResourceCPU] = *resource.NewQuantity(cpuShares, resource.DecimalSI)
		} else {
			logrus.Errorf("Failed to parse CPU %s for container %s/%s", cpu,
				podID,
				containerConfig.GetMetadata().GetName())
		}
	}

	return
}

func toCaps(caps []string) (result []v1.Capability) {
	for _, cap := range caps {
		result = append(result, v1.Capability(cap))
	}
	return
}

func boolPointer(priv bool) *bool {
	if priv {
		return &priv
	}
	return nil
}

func toContainer(mappings []*pb.PortMapping, volumes []v1.Volume, data *containerData) ([]v1.Volume, v1.Container, error) {
	containerConfig, _, err := getContainerConfig(data.container)
	if err != nil {
		return nil, v1.Container{}, err
	}

	var (
		volumeMounts []v1.VolumeMount
		ports        []v1.ContainerPort
	)

	for _, mount := range containerConfig.Mounts {
		volumes, volumeMounts = addMount(volumes, volumeMounts, mount)
	}

	for i, port := range mappings {
		cp := v1.ContainerPort{
			Name:          fmt.Sprintf("port%d", i),
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			HostIP:        port.HostIp,
		}
		if port.Protocol == pb.Protocol_UDP {
			cp.Protocol = v1.ProtocolUDP
		}
		ports = append(ports, cp)
	}

	return volumes, v1.Container{
		Name:          data.container.Metadata.Name,
		Image:         data.container.Image.Image,
		Command:       containerConfig.Command,
		Args:          containerConfig.Args,
		WorkingDir:    containerConfig.WorkingDir,
		Ports:         ports,
		Env:           toEnv(containerConfig),
		Resources:     toResources(data.container.PodSandboxId, containerConfig),
		VolumeMounts:  volumeMounts,
		VolumeDevices: nil,
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Add:  toCaps(containerConfig.GetLinux().GetSecurityContext().GetCapabilities().GetAddCapabilities()),
				Drop: toCaps(containerConfig.GetLinux().GetSecurityContext().GetCapabilities().GetDropCapabilities()),
			},
			Privileged: boolPointer(containerConfig.GetLinux().GetSecurityContext().GetPrivileged()),
			SELinuxOptions: &v1.SELinuxOptions{
				User:  containerConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetUser(),
				Role:  containerConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetRole(),
				Type:  containerConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetType(),
				Level: containerConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetLevel(),
			},
			RunAsUser:              intPointer(containerConfig.GetLinux().GetSecurityContext().GetRunAsUser()),
			RunAsGroup:             intPointer(containerConfig.GetLinux().GetSecurityContext().GetRunAsGroup()),
			ReadOnlyRootFilesystem: boolPointer(containerConfig.GetLinux().GetSecurityContext().GetReadonlyRootfs()),
		},
		Stdin:     containerConfig.GetStdin(),
		StdinOnce: containerConfig.GetStdinOnce(),
		TTY:       containerConfig.GetTty(),
	}, nil
}

func addMount(volumes []v1.Volume, mounts []v1.VolumeMount, mount *pb.Mount) ([]v1.Volume, []v1.VolumeMount) {
	name, vType := volume.PathToType(mount.HostPath)
	found := false
	for _, v := range volumes {
		if v.Name == name {
			found = true
			break
		}
	}

	if !found {
		switch vType {
		case volume.HostPathVolumeType:
			volumes = append(volumes, v1.Volume{
				Name: name,
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: mount.HostPath,
					},
				},
			})
		case volume.NamedVolumeType:
			fallthrough
		case volume.Ephemeral:
			volumes = append(volumes, v1.Volume{
				Name: name,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				},
			})
		}
	}

	volumeMount := v1.VolumeMount{
		Name:      name,
		ReadOnly:  mount.Readonly,
		MountPath: mount.ContainerPath,
	}

	switch mount.Propagation {
	case pb.MountPropagation_PROPAGATION_PRIVATE:
		volumeMount.MountPropagation = &propNone
	case pb.MountPropagation_PROPAGATION_HOST_TO_CONTAINER:
		volumeMount.MountPropagation = &propHostToContainer
	case pb.MountPropagation_PROPAGATION_BIDIRECTIONAL:
		volumeMount.MountPropagation = &propBidirectional
	}

	mounts = append(mounts, volumeMount)

	return volumes, mounts
}
