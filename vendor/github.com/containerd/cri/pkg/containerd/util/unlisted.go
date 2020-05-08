/*
Copyright 2018 The containerd Authors.

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

package util

import (
	"context"

	"github.com/containerd/ttrpc"
	"google.golang.org/grpc/metadata"
)

type unlistedKey struct{}

// WithUnlisted adds the "unlisted" namespace to the context.
func WithUnlisted(ctx context.Context, unlisted string) context.Context {
	ctx = context.WithValue(ctx, unlistedKey{}, unlisted)
	return withUnlistedHeaderTTRPC(withUnlistedHeaderGRPC(ctx, unlisted), unlisted)
}

func Unlisted(ctx context.Context) (string, bool) {
	unlisted, ok := ctx.Value(unlistedKey{}).(string)
	if !ok {
		if unlisted, ok = fromUnlistedHeaderGRPC(ctx); !ok {
			return fromUnlistedHeaderTTRPC(ctx)
		}
	}
	return unlisted, ok
}

const (
	UnlistedLabel       = "containerd.io/cri-unlisted"
	UnlistedHeaderGRPC  = `cri-unlisted`
	UnlistedHeaderTTRPC = UnlistedHeaderGRPC + `-ttrpc`
)

func withUnlistedHeaderGRPC(ctx context.Context, unlisted string) context.Context {
	// also store on the grpc headers so it gets picked up by any clients that
	// are using this.
	nsheader := metadata.Pairs(UnlistedHeaderGRPC, unlisted)
	md, ok := metadata.FromOutgoingContext(ctx) // merge with outgoing context.
	if !ok {
		md = nsheader
	} else {
		// order ensures the latest is first in this list.
		md = metadata.Join(nsheader, md)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

func fromUnlistedHeaderGRPC(ctx context.Context) (string, bool) {
	// try to extract for use in grpc servers.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// TODO(stevvooe): Check outgoing context?
		return "", false
	}

	values := md[UnlistedHeaderGRPC]
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func copyMetadataTTRPC(src ttrpc.MD) ttrpc.MD {
	md := ttrpc.MD{}
	for k, v := range src {
		md[k] = append(md[k], v...)
	}
	return md
}

func withUnlistedHeaderTTRPC(ctx context.Context, unlisted string) context.Context {
	md, ok := ttrpc.GetMetadata(ctx)
	if !ok {
		md = ttrpc.MD{}
	} else {
		md = copyMetadataTTRPC(md)
	}
	md.Set(UnlistedHeaderTTRPC, unlisted)
	return ttrpc.WithMetadata(ctx, md)
}

func fromUnlistedHeaderTTRPC(ctx context.Context) (string, bool) {
	return ttrpc.GetMetadataValue(ctx, UnlistedHeaderTTRPC)
}
