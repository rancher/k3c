package name

import (
	"github.com/docker/docker/pkg/namesgenerator"
	"strings"
)

func Random() string {
	return strings.Replace(namesgenerator.GetRandomName(0), "_", "-", -1)
}

