package images

import (
	"strings"

	"github.com/rancher/k3c/pkg/cliclient"
	"github.com/rancher/k3c/pkg/tables"
	"github.com/rancher/k3c/pkg/validate"
	"github.com/rancher/wrangler/pkg/kv"
	"github.com/urfave/cli/v2"
)

const (
	none = "<none>"
)

type Images struct {
	A_All   bool   `usage:"Show all images (default hides intermediate images)"`
	Digests bool   `usage:"Show digests"`
	Format  string `usage:"Pretty-print images using a Go template"`
	NoTrunc bool   `usage:"Don't truncate output"`
	Q_Quiet bool   `usage:"Only show numeric IDs"`
}

func (i *Images) Run(ctx *cli.Context) (err error) {
	if err := validate.NArgs(ctx, "images", 0, 1); err != nil {
		return err
	}

	client, err := cliclient.New(ctx)
	if err != nil {
		return err
	}

	images, err := client.ListImages(ctx.Context)
	if err != nil {
		return err
	}

	filterRepo, filterTag := kv.SplitLast(ctx.Args().First(), ":")
	w := tables.NewImages(ctx, i.Digests)

	for _, image := range images {
		for _, tag := range defValue(image.Tags, "") {
			id, repo, tag, digest := idRepoTagDigest(image.ID, tag, image.Digests)
			if repo == none && !i.A_All {
				continue
			}
			if filterRepo != "" && filterRepo != repo {
				continue
			}
			if filterTag != "" && filterTag != tag {
				continue
			}
			w.Write(tables.ImageData{
				ID:     id,
				Repo:   repo,
				Tag:    tag,
				Digest: digest,
				Size:   image.Size,
			})
		}
	}

	return w.Close()
}

func idRepoTagDigest(imageID, imageTag string, imageDigests []string) (id, repo, tag, digest string) {
	repo, digest = kv.Split(defValue(imageDigests, "")[0], "@")
	repoFromTag, tag := kv.SplitLast(imageTag, ":")

	if repo == "" || tag != "" {
		repo = repoFromTag
	}

	_, id = kv.Split(imageID, ":")
	if id == "" {
		id = imageID
	}

	repo = strings.TrimPrefix(repo, "docker.io/library/")
	repo = strings.TrimPrefix(repo, "docker.io/")

	return id, orNone(repo), orNone(tag), orNone(digest)
}

func orNone(val string) string {
	if val == "" {
		return none
	}
	return val
}

func defValue(s []string, defValue string) []string {
	if len(s) == 0 {
		return []string{defValue}
	}
	return s
}
