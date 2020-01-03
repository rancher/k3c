package buildkitd

import (
	"google.golang.org/grpc"
)

type ServerCallback func(server *grpc.Server) error

func Run(cb ServerCallback, args ...string) {
	main(args, cb)
}
