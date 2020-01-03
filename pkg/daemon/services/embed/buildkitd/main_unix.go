// +build !windows

package buildkitd

import (
	"syscall"
)

func init() {
	syscall.Umask(0)
}
