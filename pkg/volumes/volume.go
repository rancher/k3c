package volumes

import (
	"fmt"
	"strings"

	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func Parse(volumes []string) (result []*v1alpha2.Mount, err error) {
	for _, volume := range volumes {
		parts := strings.SplitN(volume, ":", 3)
		ro := false
		src := ""
		dest := ""
		if len(parts) > 1 {
			switch parts[len(parts)-1] {
			case "ro":
				ro = true
				fallthrough
			case "rw":
				parts = parts[:len(parts)-1]
			}
		}

		switch len(parts) {
		case 3:
			return nil, fmt.Errorf("invalid option: %s in volume definition: %s", parts[2], volume)
		case 2:
			src = parts[0]
			dest = parts[1]
		case 1:
			dest = parts[0]
		}
		if len(parts) == 3 {
		}

		result = append(result, &v1alpha2.Mount{
			ContainerPath: dest,
			HostPath:      src,
			Readonly:      ro,
			Propagation:   v1alpha2.MountPropagation_PROPAGATION_PRIVATE,
		})
	}

	return
}
