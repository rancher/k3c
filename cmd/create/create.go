package create

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	ports2 "github.com/rancher/k3c/pkg/ports"

	"github.com/rancher/k3c/pkg/volumes"

	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/auth"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/client"
	"github.com/rancher/k3c/pkg/kvfile"
	"github.com/rancher/k3c/pkg/name"
	"github.com/rancher/k3c/pkg/progress"
	"github.com/rancher/k3c/pkg/remote/apis/k3c/v1alpha1"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/rancher/wrangler/pkg/kv"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type Create struct {
	CapAdd        []string          `usage:"Add Linux capabilities"`
	CapDrop       []string          `usage:"Drop Linux capabilities"`
	CgroupParent  string            `usage:"Optional parent cgroup for the container"`
	CpuPeriod     int               `usage:"Limit CPU CFS (Completely Fair Scheduler) period"`
	CpuQuota      int               `usage:"Limit CPU CFS (Completely Fair Scheduler) quota"`
	C_CpuShares   int               `usage:"CPU shares (relative weight)"`
	CpusetCpus    string            `usage:"CPUs in which to allow execution (0-3, 0,1)"`
	CpusetMems    string            `usage:"MEMs in which to allow execution (0-3, 0,1)"`
	Cpus          string            `usage:"Number of CPUs"`
	OomScoreAdj   int               `usage:"Tune host's OOM preferences (-1000 to 1000)"`
	L_Label       []string          `usage:"Set meta data on a container"`
	LabelFile     []string          `usage:"Read in a line delimited file of labels"`
	Dns           []string          `usage:"Set custom DNS servers"`
	DnsOption     []string          `usage:"Set DNS options"`
	DnsSearch     []string          `usage:"Set custom DNS search domains"`
	E_Env         []string          `usage:"Set environment variables"`
	EnvFile       []string          `usage:"Read in a file of environment variables"`
	Entrypoint    string            `usage:"Overwrite the default ENTRYPOINT of the image"`
	GroupAdd      []string          `usage:"Add additional groups to join"`
	Init          bool              `usage:"Run an init inside the container that forwards signals and reaps processes"`
	Hostname      string            `usage:"Container host name"`
	I_Interactive bool              `usage:"Keep STDIN open even if not attached"`
	Name          string            `usage:"Assign a name to the container"`
	ReadOnly      bool              `usage:"Mount the container's root filesystem as read only"`
	Runtime       string            `usage:"Runtime to use for this container"`
	Sysctl        map[string]string `usage:"Sysctl options (default map[])"`
	T_Tty         bool              `usage:"Allocate a pseudo-TTY"`
	U_User        string            `usage:"Username or UID (format: <name|uid>[:<group|gid>])"`
	W_Workdir     string            `usage:"Working directory inside the container"`
	Pid           string            `usage:"PID namespace to use"`
	Net           string            `usage:"Connect a container to a network"`
	Ipc           string            `usage:"IPC mode to use"`
	Privileged    bool              `usage:"Give extended privileges to this container"`
	M_Memory      string            `usage:"Memory limit"`
	V_Volume      []string          `usage:"Bind mount a volume (format: [/host-src:]/container-dest[:ro])"`
	P_Publish     []string          `usage:"Publish a container's port(s) to the host (format [src:]dst[/tcp|/udp])"`

	/*
		--add-host list                  Add a custom host-to-IP mapping (host:ip)
		--cidfile string                 Write the container ID to the file
		--device list                    Add a host device to the container
		--device-cgroup-rule list        Add a rule to the cgroup allowed devices list
		--domainname string              Container NIS domain name
		--link list                      Add link to another container
		-P, --publish-all                    Publish all exposed ports to random ports
		--restart string                 Restart policy to apply when a container exits (default "no")
		--rm                             Automatically remove the container when it exits
		--security-opt list              Security Options
		--stop-signal string             Signal to stop a container (default "SIGTERM")
		--stop-timeout int               Timeout (in seconds) to stop a container
		--tmpfs list                     Mount a tmpfs directory
		--volumes-from list              Mount volumes from the specified container(s)
	*/
}

func (c *Create) toPodOptions() (*v1alpha1.PodOptions, error) {
	ports, err := ports2.Parse(c.P_Publish)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.PodOptions{
		Hostname:    c.Hostname,
		Labels:      nil,
		Annotations: nil,
		Runtime:     c.Runtime,
		DnsConfig: &pb.DNSConfig{
			Servers:  c.Dns,
			Searches: c.DnsSearch,
			Options:  c.DnsOption,
		},
		PortMappings:    ports,
		CgroupParent:    c.CgroupParent,
		Sysctls:         c.Sysctl,
		SecurityContext: c.toSharedSecretOptions(true),
	}, nil
}

func (c *Create) toSharedSecretOptions(pod bool) *v1alpha1.SharedSecurityOptions {
	opts := &v1alpha1.SharedSecurityOptions{
		Privileged:     c.Privileged,
		SelinuxOptions: nil,
		ReadonlyRoot:   c.ReadOnly,
		SeccompProfile: "",
	}
	if !pod {
		if c.Init {
			opts.PidMode = pb.NamespaceMode_POD
		}
		user, group := kv.Split(c.U_User, ":")
		opts.User = user
		opts.Group = group
		opts.Groups = c.GroupAdd
	}
	if c.Ipc == "host" {
		opts.IpcMode = pb.NamespaceMode_NODE
	}
	if c.Pid == "host" {
		opts.PidMode = pb.NamespaceMode_NODE
	}
	if c.Net == "host" {
		opts.NetMode = pb.NamespaceMode_NODE
	}

	return opts
}

func (c *Create) toContainerOptions(args []string) (*v1alpha1.ContainerOptions, error) {
	labels, err := kvfile.ReadKVMap(c.LabelFile, c.L_Label)
	if err != nil {
		return nil, err
	}

	env, err := kvfile.ReadEnv(c.EnvFile, c.E_Env)
	if err != nil {
		return nil, err
	}

	memory := int64(0)
	if c.M_Memory != "" {
		bytes, err := units.RAMInBytes(c.M_Memory)
		if err != nil {
			return nil, err
		}
		memory = bytes
	}

	containerName := c.Name
	if containerName == "" {
		containerName = name.Random()
	}

	var command []string
	if c.Entrypoint != "" {
		command = []string{c.Entrypoint}
	}

	cpus := int64(0)
	if c.Cpus != "" {
		f, err := strconv.ParseFloat(c.Cpus, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing %s", c.Cpus)
		}
		cpus = int64(f * 1000)
	}

	mounts, err := volumes.Parse(c.V_Volume)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.ContainerOptions{
		Labels:     labels,
		Name:       containerName,
		Attempt:    0,
		Command:    command,
		Args:       args,
		WorkingDir: c.W_Workdir,
		Envs:       env,
		Mounts:     mounts,
		Devices:    nil,
		Stdin:      c.I_Interactive,
		Tty:        c.T_Tty,
		MillisCpu:  cpus,
		LinuxResources: &pb.LinuxContainerResources{
			CpuPeriod:          int64(c.CpuPeriod),
			CpuQuota:           int64(c.CpuQuota),
			CpuShares:          int64(c.C_CpuShares),
			MemoryLimitInBytes: memory,
			OomScoreAdj:        int64(c.OomScoreAdj),
			CpusetCpus:         c.CpusetCpus,
			CpusetMems:         c.CpusetMems,
		},
		AddCapabilities:  c.CapAdd,
		DropCapabilities: c.CapDrop,
		ApparmorProfile:  nil,
		NoNewPrivs:       false,
		MaskedPaths:      nil,
		ReadonlyPaths:    nil,
		SecurityContext:  c.toSharedSecretOptions(false),
	}, nil
}

func (c *Create) Run(ctx *cli.Context) error {
	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	id, err := c.Create(ctx, client, false)
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func PullProgress(ctx context.Context, client client.Client, out io.Writer, image string) error {
	if out == nil {
		return nil
	}

	c, err := client.PullProgress(ctx, image)
	if err != nil {
		return err
	}

	return progress.Display(c, out)
}

func PullImage(ctx context.Context, c client.Client, image string, progress io.Writer, force bool) (string, error) {
	existingImage, err := c.GetImage(ctx, image)
	if err == client.ErrImageNotFound {
		existingImage = nil
	} else if err != nil {
		return "", err
	}

	if force {
		existingImage = nil
	}

	eg := errgroup.Group{}
	if existingImage == nil && progress != nil {
		eg.Go(func() error {
			return PullProgress(ctx, c, progress, image)
		})
	}

	if existingImage != nil && len(existingImage.Digests) > 0 {
		image = existingImage.Digests[0]
	} else {
		image, err = c.PullImage(ctx, image, auth.Lookup(image))
		if err != nil {
			return "", err
		}
	}

	return image, eg.Wait()
}

func (c *Create) Create(ctx *cli.Context, client client.Client, stdinOnce bool) (string, error) {
	if err := validate.NArgs(ctx, "create", 1, -1); err != nil {
		return "", err
	}

	image := ctx.Args().First()
	args := ctx.Args().Tail()

	_, err := PullImage(ctx.Context, client, image, os.Stderr, false)
	if err != nil {
		return "", err
	}

	podConfig, err := c.toPodOptions()
	if err != nil {
		return "", err
	}

	conConfig, err := c.toContainerOptions(args)
	if err != nil {
		return "", err
	}
	conConfig.StdinOnce = stdinOnce

	podID, err := client.CreatePod(ctx.Context, conConfig.Name, podConfig)
	if err != nil {
		return "", nil
	}

	id, err := client.CreateContainer(ctx.Context, podID, image, conConfig)
	if err != nil {
		return "", err
	}

	return id, nil
}
