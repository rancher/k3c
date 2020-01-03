package containerd

import "github.com/sirupsen/logrus"

func Run(args ...string) {
	main(args)
	logrus.Fatal("containerd exited")
}
