// +build !linux

package action

import "context"

func (s *Agent) Run(ctx context.Context) error {
	panic("not supported on this platform")
}
