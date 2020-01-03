package build

import (
	"context"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/pkg/errors"
	"github.com/rancher/k3c/pkg/kvfile"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type ProgressStyle string

var (
	ProgressStyleAuto  = ProgressStyle("auto")
	ProgressStyleNone  = ProgressStyle("none")
	ProgressStylePlain = ProgressStyle("plain")
	// what's the point? this is basically the same as auto
	ProgressStyleTTY = ProgressStyle("tty")
)

type BuildOpts struct {
	CacheFromImages []string
	Args            map[string]string
	Label           map[string]string
	Target          string
	Tag             []string
	NoCache         bool
	AddHosts        []string
	Secrets         []string
	SSH             []string
	Dockerfile      string
	Progress        ProgressStyle
	Output          string
	Pull            bool
}

func (b *buildkitClient) Build(ctx context.Context, contextDir string, opts *BuildOpts) (string, error) {
	if opts == nil {
		opts = &BuildOpts{}
	}

	tag := ""
	if len(opts.Tag) > 0 {
		tag = opts.Tag[0]
	}

	if tag != "" {
		ref, err := reference.ParseDockerRef(tag)
		if err != nil {
			return "", errors.Wrapf(err, "parsing %s", tag)
		}
		tag = ref.String()
	}

	logrus.Debugf("Building contextDir [%s] opts [%#v]", contextDir, opts)

	session, err := opts.Session()
	if err != nil {
		return "", err
	}

	solveOpts := client.SolveOpt{
		Session:       session,
		Exports:       opts.Exports(tag),
		LocalDirs:     opts.LocalDirs(contextDir),
		Frontend:      opts.Frontend(),
		FrontendAttrs: opts.FrontendAttrs(),
		CacheImports:  opts.CacheImports(),
	}

	eg := errgroup.Group{}
	resp, err := b.client.Solve(ctx, nil, solveOpts, progress(eg, opts.Progress))
	if err != nil {
		return "", err
	}

	if err := eg.Wait(); err != nil {
		return "", err
	}

	digest := resp.ExporterResponse["containerimage.digest"]
	if len(opts.Tag) > 1 {
		return digest, b.c.TagImage(ctx, opts.Tag[0], opts.Tag[1:]...)
	}

	return digest, nil
}

func progress(group errgroup.Group, style ProgressStyle) chan *client.SolveStatus {
	var (
		c   console.Console
		err error
	)

	switch style {
	case ProgressStyleNone:
		return nil
	case ProgressStylePlain:
	default:
		c, err = console.ConsoleFromFile(os.Stderr)
		if err != nil {
			c = nil
		}
	}

	ch := make(chan *client.SolveStatus, 1)
	group.Go(func() error {
		return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, ch)
	})
	return ch
}

func (b *BuildOpts) LocalDirs(contextDir string) map[string]string {
	locals := map[string]string{
		"context": contextDir,
	}
	if b.Dockerfile == "" {
		locals["dockerfile"] = contextDir
	} else {
		locals["dockerfile"] = filepath.Dir(b.Dockerfile)
	}

	return locals
}

func (b *BuildOpts) defaultExporter(tag string) []client.ExportEntry {
	exp := client.ExportEntry{
		Type:  client.ExporterImage,
		Attrs: map[string]string{},
	}
	if tag != "" {
		exp.Attrs["name"] = tag
		exp.Attrs["name-canonical"] = ""
	}
	return []client.ExportEntry{exp}
}

func (b *BuildOpts) Exports(tag string) []client.ExportEntry {
	switch b.Output {
	case "":
		return b.defaultExporter(tag)
	case "-":
		return []client.ExportEntry{
			{
				Type: "tar",
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return os.Stdout, nil
				},
			},
		}
	}

	var (
		export client.ExportEntry
		err    error
	)

	export.Type, export.Attrs, err = kvfile.ParseTypeAndKVMap(b.Output)
	if err != nil {
		logrus.Errorf("failed to parse output [%s]: %v", b.Output, err)
		return b.defaultExporter(tag)
	}

	dest := export.Attrs["dest"]

	switch export.Type {
	case "":
		for k := range export.Attrs {
			export.Type = "local"
			export.OutputDir = k
		}
	case client.ExporterLocal:
		if dest == "" {
			dest = "./output"
		}
		export.OutputDir = dest
	case client.ExporterDocker, client.ExporterTar, client.ExporterOCI:
		export.Output = fileOutput(dest)
	}

	if export.Type == "" {
		return b.defaultExporter(tag)
	}

	return []client.ExportEntry{export}
}

func fileOutput(dest string) func(map[string]string) (io.WriteCloser, error) {
	if dest == "" {
		dest = "image.tar"
	}

	return func(_ map[string]string) (io.WriteCloser, error) {
		_ = os.MkdirAll(filepath.Dir(dest), 0700)
		return os.Create(dest)
	}
}

func (b *BuildOpts) CacheImports() (result []client.CacheOptionsEntry) {
	exists := map[string]bool{}
	for _, s := range b.CacheFromImages {
		if exists[s] {
			continue
		}
		exists[s] = true

		result = append(result, client.CacheOptionsEntry{
			Type: "registry",
			Attrs: map[string]string{
				"ref": s,
			},
		})
	}

	return
}

func (b *BuildOpts) Frontend() string {
	return "dockerfile.v0"
}

func (b *BuildOpts) FrontendAttrs() map[string]string {
	result := map[string]string{
		"target": b.Target,
	}

	for k, v := range b.Args {
		result["build-arg:"+k] = v
	}

	for k, v := range b.Label {
		result["label:"+k] = v
	}

	hosts := toCSV(b.AddHosts)
	if hosts != "" {
		result["add-hosts"] = hosts
	}

	if b.Dockerfile == "" {
		result["filename"] = "Dockerfile"
	} else {
		result["filename"] = filepath.Base(b.Dockerfile)
	}

	if b.Pull {
		result["image-resolve-mode"] = "pull"
	}

	return result
}

func toCSV(items []string) string {
	buf := &strings.Builder{}
	w := csv.NewWriter(buf)
	// I don't believe this can ever fail
	_ = w.Write(items)
	return buf.String()
}
