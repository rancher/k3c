package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	criutil "github.com/containerd/cri/pkg/containerd/util"
	"github.com/containerd/cri/pkg/server"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/defaults"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/wrangler/pkg/kv"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	AnnotationPodConfig     = "k3c.io/pod-config"
	AnnotationRestartPolicy = "k3c.io/restart-policy"
)

func toPodUser(val string) *pb.Int64Value {
	if val == "" || val == "root" {
		return nil
	}

	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil
	}
	return &pb.Int64Value{
		Value: n,
	}
}

func toID(val string) (*pb.Int64Value, error) {
	if val == "" || val == "root" {
		return nil, nil
	}

	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", val)
	}
	return &pb.Int64Value{
		Value: n,
	}, nil
}

func (c *Daemon) CreatePod(ctx context.Context, name string, opts *v1alpha1.PodOptions) (string, error) {
	if opts == nil {
		opts = &v1alpha1.PodOptions{}
	}
	if opts.Labels == nil {
		opts.Labels = map[string]string{}
	}
	opts.Labels[criutil.UnlistedLabel] = defaults.PrivateNamespace

	uid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	hostname := opts.Hostname
	if hostname == "" {
		hostname = name
	}

	user := toPodUser(opts.GetSecurityContext().GetUser())
	group, err := toID(opts.GetSecurityContext().GetGroup())
	if err != nil {
		return "", errors.Wrap(err, "invalid group")
	}

	groups, err := toGroupIDs(opts.GetSecurityContext().GetGroups())
	if err != nil {
		return "", errors.Wrap(err, "parsing groups")
	}

	logs := ""
	if c.logPath != "" {
		logs = filepath.Join(c.logPath, fmt.Sprintf("%s_%s_%s", "k3c", name, uid))
	}

	config := &pb.PodSandboxConfig{
		Metadata: &pb.PodSandboxMetadata{
			Name:      name,
			Uid:       uid.String(),
			Namespace: "k3c",
		},
		Hostname:     hostname,
		LogDirectory: logs,
		DnsConfig:    opts.DnsConfig,
		PortMappings: opts.PortMappings,
		Labels:       opts.Labels,
		Annotations:  opts.Annotations,
		Linux: &pb.LinuxPodSandboxConfig{
			CgroupParent: opts.CgroupParent,
			SecurityContext: &pb.LinuxSandboxSecurityContext{
				NamespaceOptions: &pb.NamespaceOption{
					Network: opts.GetSecurityContext().GetNetMode(),
					Pid:     opts.GetSecurityContext().GetPidMode(),
					Ipc:     opts.GetSecurityContext().GetIpcMode(),
				},
				SelinuxOptions:     opts.GetSecurityContext().GetSelinuxOptions(),
				RunAsUser:          user,
				RunAsGroup:         group,
				ReadonlyRootfs:     opts.GetSecurityContext().GetReadonlyRoot(),
				SupplementalGroups: groups,
				Privileged:         opts.GetSecurityContext().GetPrivileged(),
				SeccompProfilePath: opts.GetSecurityContext().GetSeccompProfile(),
			},
			Sysctls: opts.Sysctls,
		},
	}

	if err := storePodConfig(config); err != nil {
		return "", err
	}

	pod, err := c.crt.RunPodSandbox(ctx, &pb.RunPodSandboxRequest{
		Config:         config,
		RuntimeHandler: opts.Runtime,
	})
	if err != nil {
		return "", err
	}

	return pod.PodSandboxId, nil
}

func storePodConfig(config *pb.PodSandboxConfig) error {
	m := jsonpb.Marshaler{}
	buf := &bytes.Buffer{}
	if err := m.Marshal(buf, config); err != nil {
		return errors.Wrap(err, "marshal pod config")
	}

	if config.Annotations == nil {
		config.Annotations = map[string]string{}
	}
	config.Annotations[AnnotationPodConfig] = buf.String()
	return nil
}

func getPodConfig(pod *pb.PodSandbox) (*pb.PodSandboxConfig, error) {
	var config pb.PodSandboxConfig
	anno := pod.Annotations[AnnotationPodConfig]
	return &config, jsonpb.UnmarshalString(anno, &config)
}

type podData struct {
	ID         string
	sandbox    *pb.PodSandbox
	containers []*containerData
}

func (c *Daemon) ListPods(ctx context.Context) ([]v1.Pod, error) {
	return c.listPods(ctx, true)
}

func (c *Daemon) gc(ctx context.Context) {
	for c.gck.Kicked() {
		if err := c.runGC(ctx); err != nil {
			logrus.Errorf("failed to run pod GC: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func (c *Daemon) runGC(ctx context.Context) error {
	resp, err := c.crt.ListPodSandbox(ctx, &pb.ListPodSandboxRequest{
		Filter: &pb.PodSandboxFilter{
			LabelSelector: map[string]string{
				criutil.UnlistedLabel: defaults.PrivateNamespace,
			},
		},
	})
	if err != nil {
		return err
	}

	pods := map[string]int{}
	for _, pod := range resp.Items {
		pods[pod.Id] = 0
	}
	cResp, err := c.crt.ListContainers(ctx, &pb.ListContainersRequest{
		Filter: &pb.ContainerFilter{
			LabelSelector: map[string]string{
				criutil.UnlistedLabel: defaults.PrivateNamespace,
			},
		},
	})
	if err != nil {
		return err
	}

	for _, container := range cResp.Containers {
		if _, _, err := getContainerConfig(container); err != nil {
			c.crt.StopContainer(ctx, &pb.StopContainerRequest{
				ContainerId: container.Id,
			})
			c.crt.RemoveContainer(ctx, &pb.RemoveContainerRequest{
				ContainerId: container.Id,
			})
		} else {
			pods[container.PodSandboxId]++
		}
	}

	time.Sleep(5 * time.Second)

	var lastErr error
	for podID, count := range pods {
		if count == 0 {
			logrus.Debugf("Removing pod %s", podID)
			_, err := c.crt.StopPodSandbox(ctx, &pb.StopPodSandboxRequest{
				PodSandboxId: podID,
			})
			if err != nil {
				logrus.Errorf("Failed to stop pod: %s: %v", podID, err)
				lastErr = err
				continue
			}

			_, err = c.crt.RemovePodSandbox(ctx, &pb.RemovePodSandboxRequest{
				PodSandboxId: podID,
			})
			if err != nil {
				logrus.Errorf("Failed to remove pod: %s: %v", podID, err)
				lastErr = err
			}
		}
	}

	return lastErr
}

func (c *Daemon) listPods(ctx context.Context, network bool) ([]v1.Pod, error) {
	pods := map[string]*podData{}

	resp, err := c.crt.ListPodSandbox(ctx, &pb.ListPodSandboxRequest{})
	if err != nil {
		return nil, err
	}

	for _, pod := range resp.Items {
		pods[pod.Id] = &podData{
			ID:      pod.Id,
			sandbox: pod,
		}
	}

	containers, err := c.crt.ListContainers(ctx, &pb.ListContainersRequest{})
	if err != nil {
		return nil, err
	}

	for _, container := range containers.Containers {
		pod, ok := pods[container.PodSandboxId]
		if !ok {
			continue
		}
		resp, err := c.crt.ContainerStatus(ctx, &pb.ContainerStatusRequest{
			ContainerId: container.Id,
			Verbose:     true,
		})
		if err != nil {
			return nil, err
		}
		pod.containers = append(pod.containers, &containerData{
			container: container,
			status:    resp.Status,
			info:      resp.Info,
		})
	}

	var (
		result []v1.Pod
	)

	var ips map[string][]string
	if network {
		ips = getIPs("/var/lib/cni/results")
	}

	for _, pod := range pods {
		podResult, err := c.toPod(ctx, pod)
		if err != nil {
			logrus.Warnf("failed to convert pod %s: %v", pod.ID, err)
			continue
		}
		ips := ips[pod.ID]
		if len(ips) > 0 {
			podResult.Status.PodIP = ips[0]
			for _, ip := range ips {
				podResult.Status.PodIPs = append(podResult.Status.PodIPs, v1.PodIP{
					IP: ip,
				})
			}
		}
		result = append(result, podResult)
	}

	return result, nil
}

func getIPs(dir string) map[string][]string {
	result := map[string][]string{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Errorf("failed to read CNI status for pods: %v", err)
		return nil
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "k3c-net-") {
			continue
		}
		if !strings.HasSuffix(file.Name(), "-eth0") {
			continue
		}

		podID := strings.TrimSuffix(strings.TrimPrefix(file.Name(), "k3c-net-"), "-eth0")

		r, err := os.Open(filepath.Join(dir, file.Name()))
		if err != nil {
			logrus.Errorf("failed to open CNI status for %s/%s: %v", dir, file.Name(), err)
			continue
		}

		var cniResult current.Result
		if err := json.NewDecoder(r).Decode(&cniResult); err != nil {
			logrus.Errorf("failed to decode CNI status for %s/%s: %v", dir, file.Name(), err)
			continue
		}

		var ips []string
		for _, ip := range cniResult.IPs {
			ips = append(ips, ip.Address.IP.String())
		}

		result[podID] = ips
	}

	return result
}

func restartPolicy(annotations map[string]string) v1.RestartPolicy {
	pol := v1.RestartPolicy(annotations[AnnotationRestartPolicy])
	if pol == "" {
		return v1.RestartPolicyNever
	}
	return pol
}

func intPointer(val *pb.Int64Value) *int64 {
	if val == nil {
		return nil
	}
	return &val.Value
}

func toSysctl(vals map[string]string) (result []v1.Sysctl) {
	for k, v := range vals {
		result = append(result, v1.Sysctl{
			Name:  k,
			Value: v,
		})
	}
	return
}

func toPodDNSConfigOptions(opts []string) (result []v1.PodDNSConfigOption) {
	for _, opt := range opts {
		k, v := kv.Split(opt, ":")
		dnsOpt := v1.PodDNSConfigOption{
			Name: k,
		}
		if strings.Contains(opt, ":") {
			dnsOpt.Value = &v
		}
		result = append(result, dnsOpt)
	}

	return
}

func toRuntime(runtime string) *string {
	if runtime == "" {
		return nil
	}
	return &runtime
}

func toPodStartTime(podData *podData) *metav1.Time {
	started := int64(0)
	for _, container := range podData.containers {
		if started == 0 || container.status.GetStartedAt() < started {
			started = container.status.GetStartedAt()
		}
	}
	if started == 0 {
		return nil
	}
	t := metav1.Unix(0, started)
	return &t
}

func toContainerStatus(podData *podData) (result []v1.ContainerStatus) {
	for _, container := range podData.containers {
		containerState := v1.ContainerState{
			Waiting: &v1.ContainerStateWaiting{
				Reason:  container.status.GetReason(),
				Message: container.status.GetMessage(),
			},
		}
		if container.status.GetFinishedAt() > 0 {
			containerState.Waiting = nil
			containerState.Terminated = &v1.ContainerStateTerminated{
				ExitCode: container.status.GetExitCode(),
				//Signal:      container.status.
				Reason:      container.status.GetReason(),
				Message:     container.status.GetMessage(),
				StartedAt:   metav1.Unix(0, container.status.GetStartedAt()),
				FinishedAt:  metav1.Unix(0, container.status.GetFinishedAt()),
				ContainerID: container.status.GetId(),
			}
		} else if container.status.GetStartedAt() > 0 {
			containerState.Waiting = nil
			containerState.Running = &v1.ContainerStateRunning{
				StartedAt: metav1.Unix(0, container.status.GetStartedAt()),
			}
		}

		result = append(result, v1.ContainerStatus{
			Name:         container.container.Metadata.Name,
			State:        containerState,
			RestartCount: int32(container.status.Metadata.Attempt),
			Image:        container.status.GetImage().GetImage(),
			ImageID:      container.status.GetImageRef(),
			ContainerID:  container.status.GetId(),
			Started:      &[]bool{true}[0],
		})
	}

	return
}

func toPhase(podData *podData) v1.PodPhase {
	if podData.sandbox.State == pb.PodSandboxState_SANDBOX_READY {
		if len(podData.containers) == 0 {
			return v1.PodPending
		}

		allZero := true
		allRunning := true
		for _, container := range podData.containers {
			switch container.container.State {
			case pb.ContainerState_CONTAINER_CREATED:
				return v1.PodPending
			case pb.ContainerState_CONTAINER_RUNNING:
			case pb.ContainerState_CONTAINER_EXITED:
				if container.status.GetExitCode() != 0 {
					allZero = false
				}
			case pb.ContainerState_CONTAINER_UNKNOWN:
				return v1.PodUnknown
			}
		}

		if allRunning {
			return v1.PodRunning
		} else if allZero {
			return v1.PodSucceeded
		}
		return v1.PodFailed
	}

	return v1.PodPending
}

func (c *Daemon) toPod(ctx context.Context, podData *podData) (v1.Pod, error) {
	podConfig, err := getPodConfig(podData.sandbox)
	if err != nil {
		var (
			req = &pb.PodSandboxStatusRequest{PodSandboxId: podData.ID, Verbose: true}
			res *pb.PodSandboxStatusResponse
		)
		if res, err = c.crt.PodSandboxStatus(ctx, req); err != nil {
			return v1.Pod{}, err
		}
		var info server.SandboxInfo
		if err = json.Unmarshal([]byte(res.Info["info"]), &info); err != nil {
			return v1.Pod{}, err
		}
		podConfig = info.Config
	}

	pod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:               types.UID(podData.ID),
			CreationTimestamp: metav1.Unix(0, podData.sandbox.CreatedAt),
			Name:              podData.sandbox.Metadata.Name,
			Namespace:         podData.sandbox.Metadata.Namespace,
			Labels:            map[string]string{},
			Annotations:       map[string]string{},
		},
		Spec: v1.PodSpec{
			Volumes:               nil,
			Containers:            nil,
			RestartPolicy:         restartPolicy(podData.sandbox.Annotations),
			HostNetwork:           podConfig.GetLinux().GetSecurityContext().GetNamespaceOptions().GetNetwork() == pb.NamespaceMode_NODE,
			HostPID:               podConfig.GetLinux().GetSecurityContext().GetNamespaceOptions().GetPid() == pb.NamespaceMode_NODE,
			HostIPC:               podConfig.GetLinux().GetSecurityContext().GetNamespaceOptions().GetIpc() == pb.NamespaceMode_NODE,
			ShareProcessNamespace: &[]bool{podConfig.GetLinux().GetSecurityContext().GetNamespaceOptions().GetPid() == pb.NamespaceMode_POD}[0],
			SecurityContext: &v1.PodSecurityContext{
				SELinuxOptions: &v1.SELinuxOptions{
					User:  podConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetUser(),
					Role:  podConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetRole(),
					Type:  podConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetType(),
					Level: podConfig.GetLinux().GetSecurityContext().GetSelinuxOptions().GetLevel(),
				},
				RunAsUser:          intPointer(podConfig.GetLinux().GetSecurityContext().GetRunAsUser()),
				RunAsGroup:         intPointer(podConfig.GetLinux().GetSecurityContext().GetRunAsGroup()),
				SupplementalGroups: podConfig.GetLinux().GetSecurityContext().GetSupplementalGroups(),
				Sysctls:            toSysctl(podConfig.GetLinux().GetSysctls()),
			},
			Hostname:    podConfig.Hostname,
			Subdomain:   "",
			HostAliases: nil,
			DNSConfig: &v1.PodDNSConfig{
				Nameservers: podConfig.GetDnsConfig().GetServers(),
				Searches:    podConfig.GetDnsConfig().GetSearches(),
				Options:     toPodDNSConfigOptions(podConfig.GetDnsConfig().GetOptions()),
			},
			RuntimeClassName:   toRuntime(podData.sandbox.RuntimeHandler),
			EnableServiceLinks: &[]bool{false}[0],
		},
		Status: v1.PodStatus{
			StartTime:         toPodStartTime(podData),
			Phase:             toPhase(podData),
			HostIP:            "",
			PodIP:             "",
			PodIPs:            nil,
			ContainerStatuses: toContainerStatus(podData),
		},
	}

	for i, container := range podData.containers {
		addStrings(pod.Labels, container.container.Labels)
		addStrings(pod.Annotations, container.container.Annotations)

		infoString, err := json.Marshal(container.info)
		if err == nil {
			pod.Annotations["info.k3c.io/"+container.container.Metadata.Name] = string(infoString)
		} else {
			pod.Annotations["info.k3c.io/"+container.container.Metadata.Name] = "{}"
		}

		pm := podConfig.PortMappings
		if i > 0 {
			pm = nil
		}

		volumes, container, err := c.toContainer(ctx, pm, pod.Spec.Volumes, container)
		if err == nil {
			pod.Spec.Containers = append(pod.Spec.Containers, container)
			pod.Spec.Volumes = volumes
		} else {
			logrus.Errorf("failed to read container %s on pod %s", container.Name, podData.ID)
		}
	}

	return pod, nil
}

func addStrings(into, from map[string]string) {
	for k, v := range from {
		if strings.HasPrefix(k, "k3c.io/") {
			continue
		}
		into[k] = v
	}
}
