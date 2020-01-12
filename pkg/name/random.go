package name

import (
	"strings"

	"github.com/docker/docker/pkg/namesgenerator"
)

func Random() string {
	return strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
}
