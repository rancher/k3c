/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package stream

// Copied from cri-tools project

import (
	"fmt"
	"net/url"

	dockerterm "github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
	restclient "k8s.io/client-go/rest"
	remoteclient "k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/util/term"
)

func Stream(in, tty bool, urlString string) error {
	logrus.Debugf("streaming %s stdin=%v tty=%v", urlString, in, tty)
	url, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	executor, err := remoteclient.NewSPDYExecutor(&restclient.Config{TLSClientConfig: restclient.TLSClientConfig{Insecure: true}}, "POST", url)
	if err != nil {
		return err
	}

	stdin, stdout, stderr := dockerterm.StdStreams()
	streamOptions := remoteclient.StreamOptions{
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	}
	if in {
		streamOptions.Stdin = stdin
	}
	logrus.Debugf("StreamOptions: %v", streamOptions)
	if !tty {
		return executor.Stream(streamOptions)
	}
	if !in {
		return fmt.Errorf("tty=true must be specified with interactive=true")
	}
	t := term.TTY{
		In:  stdin,
		Out: stdout,
		Raw: true,
	}
	if !t.IsTerminalIn() {
		return fmt.Errorf("input is not a terminal")
	}
	streamOptions.TerminalSizeQueue = t.MonitorSize(t.GetSize())
	return t.Safe(func() error { return executor.Stream(streamOptions) })
}
