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

// adapted from https://github.com/kubernetes-sigs/cri-tools/blob/1bcad62d514c1166c9fd49557d2c5de2b05368aa/cmd/crictl/image.go

package images

import (
	"sort"
	"strings"

	criv1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type byId []*criv1.Image

func (a byId) Len() int      { return len(a) }
func (a byId) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byId) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}

type byDigest []*criv1.Image

func (a byDigest) Len() int      { return len(a) }
func (a byDigest) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byDigest) Less(i, j int) bool {
	return strings.Join(a[i].RepoDigests, `_`) < strings.Join(a[j].RepoDigests, `_`)
}

func Sort(refs []*criv1.Image) {
	sort.Sort(byId(refs))
	sort.Sort(byDigest(refs))
}

func TruncateID(id, prefix string, n int) string {
	id = strings.TrimPrefix(id, prefix)
	if len(id) > n {
		id = id[:n]
	}
	return id
}

func NormalizeRepoDigest(repoDigests []string) (string, string) {
	if len(repoDigests) == 0 {
		return "<none>", "<none>"
	}
	repoDigestPair := strings.Split(repoDigests[0], "@")
	if len(repoDigestPair) != 2 {
		return "errorName", "errorRepoDigest"
	}
	return repoDigestPair[0], repoDigestPair[1]
}

func NormalizeRepoTagPair(repoTags []string, imageName string) (repoTagPairs [][]string) {
	const none = "<none>"
	if len(repoTags) == 0 {
		repoTagPairs = append(repoTagPairs, []string{imageName, none})
		return
	}
	for _, repoTag := range repoTags {
		idx := strings.LastIndex(repoTag, ":")
		if idx == -1 {
			repoTagPairs = append(repoTagPairs, []string{"errorRepoTag", "errorRepoTag"})
			continue
		}
		name := repoTag[:idx]
		if name == none {
			name = imageName
		}
		repoTagPairs = append(repoTagPairs, []string{name, repoTag[idx+1:]})
	}
	return
}
