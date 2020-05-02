module github.com/containerd/cri

go 1.13

replace (
	github.com/containerd/containerd => github.com/dweomer/containerd v1.3.5-0.20200416225531-efdd59c500bf // v1.3.4+k3c.1
	github.com/docker/distribution => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc10
	k8s.io/api => k8s.io/api v0.16.6 // v0.16.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.6 // v1.16.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.6 // v1.16.6
	k8s.io/apiserver => k8s.io/apiserver v0.16.6 // v1.16.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.6 // v1.16.6
	k8s.io/client-go => k8s.io/client-go v0.16.6 // v1.16.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.6 // v1.16.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.6 // v1.16.6
	k8s.io/code-generator => k8s.io/code-generator v0.16.6 // v1.16.6
	k8s.io/component-base => k8s.io/component-base v0.16.6 // v1.16.6
	k8s.io/cri-api => k8s.io/cri-api v0.16.6 // v1.16.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.6 // v1.16.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.6 // v1.16.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.6 // v1.16.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.6 // v1.16.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.6 // v1.16.6
	k8s.io/kubectl => k8s.io/kubectl v0.16.6 // v1.16.6
	k8s.io/kubelet => k8s.io/kubelet v0.16.6 // v1.16.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.16.6 // v1.16.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.6 // v1.16.6
	k8s.io/metrics => k8s.io/metrics v0.16.6 // v1.16.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.6 // v1.16.6
)

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/containerd/cgroups v0.0.0-20200413225007-9f1c62dddf4b
	github.com/containerd/containerd v1.3.4
	github.com/containerd/continuity v0.0.0-20190815185530-f2a389ac0a02
	github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	github.com/containerd/go-cni v0.0.0-20190813230227-49fbd9b210f3
	github.com/containerd/typeurl v1.0.0
	github.com/containernetworking/plugins v0.7.6
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20171019062838-86f080cff091
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/protobuf v1.3.1
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20180430190053-c9281466c8b2
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc9
	github.com/opencontainers/runtime-spec v1.0.2-0.20190207185410-29686dbc5559
	github.com/opencontainers/selinux v1.3.1-0.20190929122143-5215b1806f52
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e
	google.golang.org/grpc v1.23.0
	k8s.io/apimachinery v0.16.6
	k8s.io/client-go v0.16.6
	k8s.io/cri-api v0.16.6
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.16.6
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f
)
