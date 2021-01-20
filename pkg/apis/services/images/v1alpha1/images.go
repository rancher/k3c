// support protobuf code generation

package images

import (
	// vendor-time imports supporting protoc imports
	_ "github.com/gogo/googleapis/google/rpc"
	_ "github.com/gogo/protobuf/gogoproto"
	_ "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)
