package build

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/client/build"
	"github.com/rancher/k3c/pkg/kvfile"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/urfave/cli/v2"
)

type Build struct {
	AddHost   []string `usage:"Add a custom host-to-IP mapping (host:ip)"`
	BuildArg  []string `usage:"Set build-time variables"`
	CacheFrom []string `usage:"Images to consider as cache sources"`
	F_File    string   `usage:"Name of the Dockerfile (Default is 'PATH/Dockerfile')"`
	Label     []string `usage:"Set metadata for an image"`
	NoCache   bool     `usage:"Do not use cache when building the image"`
	O_Output  string   `usage:"Output directory or - for stdout. (adv. format: type=local,dest=path)"`
	Progress  string   `usage:"Set type of progress output (auto, plain, tty). Use plain to show container output" default:"auto"`
	Q_Quiet   bool     `usage:"Suppress the build output and print image ID on success"`
	Secret    []string `usage:"Secret file to expose to the build (only if BuildKit enabled): id=mysecret,src=/local/secret"`
	T_Tag     []string `usage:"Name and optionally a tag in the 'name:tag' format"`
	Target    string   `usage:"Set the target build stage to build."`
	Ssh       []string `usage:"SSH agent socket or keys to expose to the build (only if BuildKit enabled) (format: default|<id>[=<socket>|<key>[,<key>]])"`
	Pull      bool     `usage:"Always attempt to pull a newer version of the image"`
}

func (b *Build) Run(ctx *cli.Context) error {
	if err := validate.NArgs(ctx, "build", 1, 1); err != nil {
		return err
	}

	args, err := kvfile.ReadKVMap(nil, b.BuildArg)
	if err != nil {
		return errors.Wrap(err, "parsing build-arg")
	}

	labels, err := kvfile.ReadKVMap(nil, b.Label)
	if err != nil {
		return errors.Wrap(err, "parsing label")
	}

	//if b.T_Tag != "" && b.O_Output != "" {
	//	return fmt.Errorf("--tag and --output can not be combined")
	//}

	opts := &build.Opts{
		CacheFromImages: b.CacheFrom,
		Pull:            b.Pull,
		Args:            args,
		Label:           labels,
		Tag:             b.T_Tag,
		Target:          b.Target,
		SSH:             b.Ssh,
		NoCache:         b.NoCache,
		AddHosts:        b.AddHost,
		Dockerfile:      b.F_File,
		Progress:        build.ProgressStyle(b.Progress),
		Secrets:         b.Secret,
		Output:          b.O_Output,
	}

	if b.Q_Quiet {
		opts.Progress = build.ProgressStyleNone
	}

	client, err := cliclient.NewBuilder(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	contextDir := ctx.Args().First()

	var closer io.Closer
	opts.Dockerfile, contextDir, closer, err = readStdinContent(opts.Dockerfile, contextDir)
	if err != nil {
		return err
	}
	defer closer.Close()

	digest, err := client.Build(ctx.Context, contextDir, opts)
	if err != nil {
		return err
	}

	if digest != "" {
		fmt.Println(digest)
	}

	return nil
}

func readStdinContent(dockerfile, contextDir string) (string, string, io.Closer, error) {
	if dockerfile != "-" && contextDir != "-" {
		return dockerfile, contextDir, (*dirCleanup)(nil), nil
	}

	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return "", "", nil, err
	}

	tempDir, err := ioutil.TempDir("", "k3c-build")
	if err != nil {
		return "", "", nil, err
	}

	tempDockerfile := filepath.Join(tempDir, "Dockerfile")
	if err := ioutil.WriteFile(tempDockerfile, bytes, 0600); err != nil {
		return "", "", nil, err
	}

	if contextDir == "-" {
		return tempDockerfile, tempDir, &dirCleanup{Dir: tempDir}, nil
	}

	return tempDockerfile, contextDir, &dirCleanup{Dir: tempDir}, nil
}

type dirCleanup struct {
	Dir string
}

func (d *dirCleanup) Close() error {
	if d == nil {
		return nil
	}
	return os.RemoveAll(d.Dir)
}
