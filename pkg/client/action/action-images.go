package action

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/docker/go-units"
	"github.com/rancher/k3c/pkg/apis/services/images"
	imagesv1 "github.com/rancher/k3c/pkg/apis/services/images/v1alpha1"
	"github.com/rancher/k3c/pkg/client"
	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type ListImages struct {
	All     bool `usage:"Show all images (default hides tag-less images)" short:"a"`
	Digests bool `usage:"Show digests"`
	//Filter  string `usage:"Filter output based on conditions provided" short:"f"`
	//Format  string `usage:"Pretty-print images using a Go template"`
	NoTrunc bool `usage:"Don't truncate output"`
	Quiet   bool `usage:"Only show image IDs" short:"q"`
}

func (s *ListImages) Invoke(ctx context.Context, k8s *client.Interface, names []string) error {
	return DoImages(ctx, k8s, func(ctx context.Context, imagesClient imagesv1.ImagesClient) error {
		req := &imagesv1.ImageListRequest{}
		// TODO filtering not working as expected
		if len(names) > 0 {
			req.Filter = &criv1.ImageFilter{
				Image: &criv1.ImageSpec{
					Image: names[0],
				},
			}
		}
		res, err := imagesClient.List(ctx, req)
		if err != nil {
			return err
		}
		images.Sort(res.Images)

		// output in table format by default.
		display := newTableDisplay(20, 1, 3, ' ', 0)
		if !s.Quiet {
			if s.Digests {
				display.AddRow([]string{columnImage, columnTag, columnDigest, columnImageID, columnSize})
			} else {
				display.AddRow([]string{columnImage, columnTag, columnImageID, columnSize})
			}
		}
		for _, image := range res.Images {
			if s.Quiet {
				fmt.Printf("%s\n", image.Id)
				continue
			}
			imageName, repoDigest := images.NormalizeRepoDigest(image.RepoDigests)
			repoTagPairs := images.NormalizeRepoTagPair(image.RepoTags, imageName)
			size := units.HumanSizeWithPrecision(float64(image.GetSize_()), 3)
			id := image.Id
			if !s.NoTrunc {
				id = images.TruncateID(id, "sha256:", 13)
				repoDigest = images.TruncateID(repoDigest, "sha256:", 13)
			}
			for _, repoTagPair := range repoTagPairs {
				if !s.All && repoDigest == "<none>" {
					continue
				}
				if s.Digests {
					display.AddRow([]string{repoTagPair[0], repoTagPair[1], repoDigest, id, size})
				} else {
					display.AddRow([]string{repoTagPair[0], repoTagPair[1], id, size})
				}
			}
			continue
			fmt.Printf("ID: %s\n", image.Id)
			for _, tag := range image.RepoTags {
				fmt.Printf("RepoTags: %s\n", tag)
			}
			for _, digest := range image.RepoDigests {
				fmt.Printf("RepoDigests: %s\n", digest)
			}
			if image.Size_ != 0 {
				fmt.Printf("Size: %d\n", image.Size_)
			}
			if image.Uid != nil {
				fmt.Printf("Uid: %v\n", image.Uid)
			}
			if image.Username != "" {
				fmt.Printf("Username: %v\n", image.Username)
			}
			fmt.Printf("\n")
		}
		display.Flush()
		return nil
	})
}

const (
	columnImage   = "IMAGE"
	columnImageID = "IMAGE ID"
	columnSize    = "SIZE"
	columnTag     = "TAG"
	columnDigest  = "DIGEST"
)

// display use to output something on screen with table format.
type display struct {
	w *tabwriter.Writer
}

// newTableDisplay creates a display instance, and uses to format output with table.
func newTableDisplay(minwidth, tabwidth, padding int, padchar byte, flags uint) *display {
	w := tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, padchar, 0)
	return &display{w}
}

// AddRow add a row of data.
func (d *display) AddRow(row []string) {
	fmt.Fprintln(d.w, strings.Join(row, "\t"))
}

// Flush output all rows on screen.
func (d *display) Flush() error {
	return d.w.Flush()
}

// ClearScreen clear all output on screen.
func (d *display) ClearScreen() {
	fmt.Fprint(os.Stdout, "\033[2J")
	fmt.Fprint(os.Stdout, "\033[H")
}
