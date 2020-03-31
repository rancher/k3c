package daemon

import (
	// containerd builtins
	_ "github.com/containerd/containerd/diff/walking/plugin"
	_ "github.com/containerd/containerd/gc/scheduler"
	_ "github.com/containerd/containerd/metrics/cgroups"
	_ "github.com/containerd/containerd/runtime/restart/monitor"
	_ "github.com/containerd/containerd/runtime/v1/linux"
	_ "github.com/containerd/containerd/runtime/v2"
	_ "github.com/containerd/containerd/runtime/v2/runc/options"
	_ "github.com/containerd/containerd/services/containers"
	_ "github.com/containerd/containerd/services/content"
	_ "github.com/containerd/containerd/services/diff"
	_ "github.com/containerd/containerd/services/events"
	_ "github.com/containerd/containerd/services/healthcheck"
	_ "github.com/containerd/containerd/services/images"
	_ "github.com/containerd/containerd/services/introspection"
	_ "github.com/containerd/containerd/services/leases"
	_ "github.com/containerd/containerd/services/namespaces"
	_ "github.com/containerd/containerd/services/opt"
	_ "github.com/containerd/containerd/services/snapshots"
	_ "github.com/containerd/containerd/services/tasks"
	_ "github.com/containerd/containerd/services/version"
	_ "github.com/containerd/containerd/snapshots/native"
	_ "github.com/containerd/containerd/snapshots/overlay"

	// cri
	_ "github.com/containerd/cri"

	// snapshotters
	_ "github.com/containerd/aufs"
	_ "github.com/containerd/zfs"
)
