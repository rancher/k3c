package ports

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func Parse(ports []string) (result []*pb.PortMapping, err error) {
	for _, p := range ports {
		hostPort := 0
		containerPort := 0
		proto := pb.Protocol_TCP
		if strings.HasSuffix(p, "/tcp") {
			p = p[:len(p)-4]
		} else if strings.HasSuffix(p, "/udp") {
			proto = pb.Protocol_UDP
			p = p[:len(p)-4]
		}

		parts := strings.SplitN(p, ":", 2)
		if len(parts) == 1 {
			containerPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, errors.Wrapf(err, "parsing %s", parts[0])
			}
		} else if len(parts) == 2 {
			hostPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, errors.Wrapf(err, "parsing %s", parts[0])
			}
			containerPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return nil, errors.Wrapf(err, "parsing %s", parts[1])
			}
		} else {
			return nil, errors.Wrapf(err, "invalid port definition: %s", p)
		}

		result = append(result, &pb.PortMapping{
			Protocol:      proto,
			ContainerPort: int32(containerPort),
			HostPort:      int32(hostPort),
		})
	}

	return result, nil
}
