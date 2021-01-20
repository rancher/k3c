package action

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	buildkit "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/rancher/k3c/pkg/client"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type BuildImage struct {
	AddHost  []string `usage:"Add a custom host-to-IP mapping (host:ip)"`
	BuildArg []string `usage:"Set build-time variables"`
	//CacheFrom []string `usage:"Images to consider as cache sources"`
	File  string   `usage:"Name of the Dockerfile (Default is 'PATH/Dockerfile')" short:"f"`
	Label []string `usage:"Set metadata for an image"`
	//NoCache   bool     `usage:"Do not use cache when building the image"`
	//Output    string   `usage:"Output directory or - for stdout. (adv. format: type=local,dest=path)" short:"o"`
	Progress string `usage:"Set type of progress output (auto, plain, tty). Use plain to show container output" default:"auto"`
	//Quiet     bool     `usage:"Suppress the build output and print image ID on success" short:"q"`
	//Secret    []string `usage:"Secret file to expose to the build (only if Buildkit enabled): id=mysecret,src=/local/secret"`
	Tag    []string `usage:"Name and optionally a tag in the 'name:tag' format" short:"t"`
	Target string   `usage:"Set the target build stage to build."`
	//Ssh       []string `usage:"SSH agent socket or keys to expose to the build (only if Buildkit enabled) (format: default|<id>[=<socket>|<key>[,<key>]])"`
	Pull bool `usage:"Always attempt to pull a newer version of the image"`
}

func (s *BuildImage) Invoke(ctx context.Context, k8s *client.Interface, path string) error {
	return DoControl(ctx, k8s, func(ctx context.Context, bkc *buildkit.Client) error {
		options := buildkit.SolveOpt{
			Frontend:      s.Frontend(),
			FrontendAttrs: s.FrontendAttrs(),
			LocalDirs:     s.LocalDirs(path),
		}
		if len(s.Tag) > 0 {
			options.Exports = defaultExporter(s.Tag[0])
		}
		eg := errgroup.Group{}
		res, err := bkc.Solve(ctx, nil, options, s.progress(&eg))
		if err != nil {
			return err
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		logrus.Debugf("%#v", res)
		return nil
	})
}

func (s *BuildImage) Frontend() string {
	return "dockerfile.v0"
}

func (s *BuildImage) FrontendAttrs() map[string]string {
	// --target
	m := map[string]string{
		"target": s.Target,
	}
	// --build-arg
	for _, b := range s.BuildArg {
		p := strings.SplitN(b, "=", 2)
		k := fmt.Sprintf("build-arg:%s", p[0])
		v := strings.Join(p[1:], "=")
		m[k] = v
	}
	// --label
	for _, l := range s.Label {
		p := strings.SplitN(l, "=", 2)
		k := fmt.Sprintf("label:%s", p[0])
		v := strings.Join(p[1:], "=")
		m[k] = v
	}
	// --add-host
	h := strings.Join(s.AddHost, ",")
	if h != "" {
		m["add-hosts"] = h
	}
	// --file
	if s.File == "" {
		m["filename"] = "Dockerfile"
	} else {
		m["filename"] = filepath.Base(s.File)
	}
	// --pull
	if s.Pull {
		m["image-resolve-mode"] = "pull"
	}
	return m
}

func (s *BuildImage) LocalDirs(path string) map[string]string {
	m := map[string]string{
		"context": path,
	}
	if s.File == "" {
		m["dockerfile"] = path
	} else {
		m["dockerfile"] = filepath.Dir(s.File)
	}
	return m
}

func (s *BuildImage) progress(group *errgroup.Group) chan *buildkit.SolveStatus {
	var (
		c   console.Console
		err error
	)

	switch s.Progress {
	case "none":
		return nil
	case "plain":
	default:
		c, err = console.ConsoleFromFile(os.Stderr)
		if err != nil {
			c = nil
		}
	}

	ch := make(chan *buildkit.SolveStatus, 1)
	group.Go(func() error {
		return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, ch)
	})
	return ch
}

func defaultExporter(tag string) []buildkit.ExportEntry {
	exp := buildkit.ExportEntry{
		Type:  buildkit.ExporterImage,
		Attrs: map[string]string{},
	}
	if tag != "" {
		exp.Attrs["name"] = tag
		exp.Attrs["name-canonical"] = ""
	}
	return []buildkit.ExportEntry{exp}
}
