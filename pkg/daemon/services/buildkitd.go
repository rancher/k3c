package services

import (
	"path/filepath"

	"github.com/containerd/cri/pkg/constants"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/daemon/services/embed/buildkitd"
	"github.com/sirupsen/logrus"
)

func startBuildkitd(containerdAddress string, serverCB buildkitd.ServerCallback, stateDir, rootDir string, opts *Opts) error {
	cni := cniDir(rootDir)
	cniConfig := filepath.Join(cni, "10-k3c.json")
	address := "unix://" + filepath.Join(stateDir, "k3c.sock")
	if err := writeJSONToFile(cniConfig, cniPlugin(opts)); err != nil {
		return errors.Wrap(err, "write cni conf")
	}

	args := []string{
		"buildkitd",
		"--root=" + filepath.Join(rootDir, "buildkitd"),
		"--addr=" + address,
		"--containerd-worker-namespace=" + constants.K8sContainerdNamespace,
		"--containerd-worker=true",
		"--containerd-worker-addr=" + containerdAddress,
		"--containerd-worker-net=cni",
		"--containerd-cni-config-path=" + cniConfig,
		"--containerd-cni-binary-dir=" + filepath.Join(rootDir, "bin"),
		"--containerd-worker-gc=true",
		"--oci-worker=false",
	}

	if opts.Group != "" {
		args = append(args, "--group="+opts.Group)
	}

	//if opts.ExtraConfig != "" {
	//	args = append(args, "--config="+opts.ExtraConfig)
	//}

	go func() {
		buildkitd.Run(serverCB, args...)
		logrus.Fatalf("buildkitd exited")
	}()

	return nil
}
