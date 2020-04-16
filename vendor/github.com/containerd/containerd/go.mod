module github.com/containerd/containerd

go 1.13

replace (
	github.com/docker/distribution => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
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
	github.com/Microsoft/go-winio v0.4.14
	github.com/Microsoft/hcsshim v0.8.7-0.20190820203702-9e921883ac92
	github.com/containerd/aufs v0.0.0-20190114185352-f894a800659b
	github.com/containerd/btrfs v0.0.0-20181101203652-af5082808c83
	github.com/containerd/cgroups v0.0.0-20200413225007-9f1c62dddf4b
	github.com/containerd/console v1.0.0
	github.com/containerd/continuity v0.0.0-20190815185530-f2a389ac0a02
	github.com/containerd/cri v1.11.1-0.20200320165605-f864905c93b9
	github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
	github.com/containerd/go-cni v0.0.0-20190813230227-49fbd9b210f3 // indirect
	github.com/containerd/go-runc v0.0.0-20190911050354-e029b79d8cda
	github.com/containerd/ttrpc v1.0.0
	github.com/containerd/typeurl v1.0.0
	github.com/containerd/zfs v0.0.0-20190829050200-2ceb2dbb8154
	github.com/containernetworking/plugins v0.7.6 // indirect
	github.com/coreos/go-systemd v0.0.0-20181012123002-c6f51f82210d
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20171019062838-86f080cff091 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916
	github.com/docker/go-units v0.4.0
	github.com/gogo/googleapis v1.2.0
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/google/go-cmp v0.3.0
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.7
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20180430190053-c9281466c8b2
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc2.0.20190611121236-6cc515888830
	github.com/opencontainers/runtime-spec v1.0.2-0.20190207185410-29686dbc5559
	github.com/opencontainers/selinux v1.3.1-0.20190929122143-5215b1806f52
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/sirupsen/logrus v1.4.2
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2
	github.com/tchap/go-patricia v2.2.6+incompatible // indirect
	github.com/urfave/cli v1.22.0
	go.etcd.io/bbolt v1.3.3
	golang.org/x/crypto v0.0.0-20200128174031-69ecbb4d6d5d // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e
	google.golang.org/grpc v1.23.0
	gopkg.in/yaml.v2 v2.2.8 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/cri-api v0.16.6 // indirect
	k8s.io/kubernetes v1.16.6 // indirect
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f // indirect
)
