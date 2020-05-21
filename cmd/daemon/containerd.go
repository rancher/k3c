// +build !linux

package daemon

import (
	"os"

	"github.com/sirupsen/logrus"
)

func Containerd() {
	logrus.Fatalf("%s only supported on Linux", os.Args[0])
}
