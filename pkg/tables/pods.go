package tables

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/rancher/k3c/pkg/table"
	"github.com/urfave/cli/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ContainerData struct {
	Pod       v1.Pod
	Container v1.Container
	Status    v1.ContainerStatus
	Info      map[string]interface{}
}

func NewPods(cli *cli.Context) table.Writer {
	cols := [][]string{
		{"CONTAINER ID", "Status.containerID | id"},
		{"IMAGE", "Container.image | formatImage"},
		{"COMMAND", "Info.runtimeSpec.process.args | join \" \" | formatCommand"},
		{"CREATED", "Typed.Pod.CreationTimestamp | ago"},
		{"STATUS", "Typed.Status | containerStatus"},
		{"PORTS", "Typed.Container.Ports | formatPorts"},
		{"NAMES", "Container.name"},
	}

	w := table.NewWriter(cols, config(cli, "CONTAINER ID"))
	w.AddFormatFunc("formatPorts", formatPorts)
	w.AddFormatFunc("formatImage", formatImage)
	w.AddFormatFunc("formatCommand", formatCommandFunc(cli))
	w.AddFormatFunc("ago", ago)
	w.AddFormatFunc("containerStatus", containerStatus)
	return w
}

func formatCommandFunc(cli *cli.Context) table.FormatFunc {
	return func(command string) (string, error) {
		if !cli.Bool("no-trunc") && len(command) > 40 {
			return command[:37] + `...`, nil
		}
		return command, nil
	}
}

func formatPorts(ports []v1.ContainerPort) (string, error) {
	var result strings.Builder
	for _, port := range ports {
		if result.Len() > 0 {
			result.WriteString(", ")
		}
		if port.HostPort > 0 {
			if port.HostIP != "" {
				result.WriteString(port.HostIP)
				result.WriteString(":")
			}
			result.WriteString(strconv.Itoa(int(port.HostPort)))
			result.WriteString("->")
		}
		result.WriteString(strconv.Itoa(int(port.ContainerPort)))
		if port.Protocol == v1.ProtocolUDP {
			result.WriteString("/udp")
		} else {
			result.WriteString("/tcp")
		}
	}

	return result.String(), nil
}

func formatImage(image string) (string, error) {
	image = strings.TrimPrefix(strings.SplitN(image, "@", 2)[0], "docker.io/library/")
	return strings.TrimPrefix(strings.SplitN(image, "@", 2)[0], "docker.io/"), nil
}

func containerStatus(status v1.ContainerStatus) (string, error) {
	if status.State.Waiting != nil {
		return "Created", nil
	} else if status.State.Running != nil {
		return ago(status.State.Running.StartedAt)
	} else if status.State.Terminated != nil {
		ago, err := ago(status.State.Terminated.FinishedAt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Exited (%d) %s", status.State.Terminated.ExitCode, ago), nil
	}
	return "Unknown", nil
}

func ago(t metav1.Time) (string, error) {
	return units.HumanDuration(time.Now().UTC().Sub(t.Time)) + " ago", nil
}
