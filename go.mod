module github.com/rancher/k3c

go 1.13

replace (
	github.com/containerd/containerd => github.com/rancher/containerd v1.3.3-k3s2
	github.com/containerd/cri => github.com/dweomer/cri v1.11.1-0.20200310230101-bc6bc8f7a60e // k3c
	github.com/docker/distribution => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/hashicorp/go-immutable-radix => github.com/tonistiigi/go-immutable-radix v0.0.0-20170803185627-826af9ccf0fe
	github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305
	github.com/moby/buildkit => github.com/dweomer/buildkit v0.6.2-0.20200111044149-7562b84375e4 // k3c
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc10

	k8s.io/api => github.com/rancher/kubernetes/staging/src/k8s.io/api v1.17.3-k3s1
	k8s.io/apiextensions-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiextensions-apiserver v1.17.3-k3s1
	k8s.io/apimachinery => github.com/rancher/kubernetes/staging/src/k8s.io/apimachinery v1.17.3-k3s1
	k8s.io/apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiserver v1.17.3-k3s1
	k8s.io/cli-runtime => github.com/rancher/kubernetes/staging/src/k8s.io/cli-runtime v1.17.3-k3s1
	k8s.io/client-go => github.com/rancher/kubernetes/staging/src/k8s.io/client-go v1.17.3-k3s1
	k8s.io/cloud-provider => github.com/rancher/kubernetes/staging/src/k8s.io/cloud-provider v1.17.3-k3s1
	k8s.io/cluster-bootstrap => github.com/rancher/kubernetes/staging/src/k8s.io/cluster-bootstrap v1.17.3-k3s1
	k8s.io/code-generator => github.com/rancher/kubernetes/staging/src/k8s.io/code-generator v1.17.3-k3s1
	k8s.io/component-base => github.com/rancher/kubernetes/staging/src/k8s.io/component-base v1.17.3-k3s1
	k8s.io/cri-api => github.com/rancher/kubernetes/staging/src/k8s.io/cri-api v1.17.3-k3s1
	k8s.io/csi-translation-lib => github.com/rancher/kubernetes/staging/src/k8s.io/csi-translation-lib v1.17.3-k3s1
	k8s.io/kube-aggregator => github.com/rancher/kubernetes/staging/src/k8s.io/kube-aggregator v1.17.3-k3s1
	k8s.io/kube-controller-manager => github.com/rancher/kubernetes/staging/src/k8s.io/kube-controller-manager v1.17.3-k3s1
	k8s.io/kube-proxy => github.com/rancher/kubernetes/staging/src/k8s.io/kube-proxy v1.17.3-k3s1
	k8s.io/kube-scheduler => github.com/rancher/kubernetes/staging/src/k8s.io/kube-scheduler v1.17.3-k3s1
	k8s.io/kubectl => github.com/rancher/kubernetes/staging/src/k8s.io/kubectl v1.17.3-k3s1
	k8s.io/kubelet => github.com/rancher/kubernetes/staging/src/k8s.io/kubelet v1.17.3-k3s1
	k8s.io/kubernetes => github.com/rancher/kubernetes v1.17.3-k3s1
	k8s.io/legacy-cloud-providers => github.com/rancher/kubernetes/staging/src/k8s.io/legacy-cloud-providers v1.17.3-k3s1
	k8s.io/metrics => github.com/rancher/kubernetes/staging/src/k8s.io/metrics v1.17.3-k3s1
	k8s.io/node-api => github.com/rancher/kubernetes/staging/src/k8s.io/node-api v1.17.3-k3s1
	k8s.io/sample-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/sample-apiserver v1.17.3-k3s1
	k8s.io/sample-cli-plugin => github.com/rancher/kubernetes/staging/src/k8s.io/sample-cli-plugin v1.17.3-k3s1
	k8s.io/sample-controller => github.com/rancher/kubernetes/staging/src/k8s.io/sample-controller v1.17.3-k3s1
)

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/containerd/aufs v0.0.0-20190114185352-f894a800659b
	github.com/containerd/console v0.0.0-20191219165238-8375c3424e4d
	github.com/containerd/containerd v1.3.2
	github.com/containerd/cri v1.11.1-0.20191202213601-b1bef15fbeb6
	github.com/containerd/typeurl v0.0.0-20200205145503-b45ef1f1f737
	github.com/containerd/zfs v0.0.0-20190829050200-2ceb2dbb8154
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.4 // indirect
	github.com/coreos/go-systemd/v22 v22.0.0
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0
	github.com/gofrs/flock v0.7.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.3
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/moby/buildkit v0.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc9
	github.com/opencontainers/runtime-spec v1.0.2-0.20200307132014-f49fed0d6290 // indirect
	github.com/opentracing/opentracing-go v0.0.0-20171003133519-1361b9cd60be
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/rancher/norman/v2 v2.0.0-20200111044641-76fd7a67396a
	github.com/rancher/wrangler v0.5.0
	github.com/sirupsen/logrus v1.4.2
	github.com/tchap/go-patricia v2.3.0+incompatible // indirect
	github.com/uber/jaeger-client-go v0.0.0-20180103221425-e02c85f9069e
	github.com/urfave/cli v1.22.2
	github.com/urfave/cli/v2 v2.1.1
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.23.1
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/cri-api v0.17.3
	k8s.io/kubectl v0.17.3
	k8s.io/kubernetes v1.17.3
	sigs.k8s.io/yaml v1.1.0
)
