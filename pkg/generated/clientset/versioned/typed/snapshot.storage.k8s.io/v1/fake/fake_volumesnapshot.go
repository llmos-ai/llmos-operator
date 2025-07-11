/*
Copyright 2025 llmos.ai.

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
// Code generated by main. DO NOT EDIT.

package fake

import (
	v1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotstoragek8siov1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/snapshot.storage.k8s.io/v1"
	gentype "k8s.io/client-go/gentype"
)

// fakeVolumeSnapshots implements VolumeSnapshotInterface
type fakeVolumeSnapshots struct {
	*gentype.FakeClientWithList[*v1.VolumeSnapshot, *v1.VolumeSnapshotList]
	Fake *FakeSnapshotV1
}

func newFakeVolumeSnapshots(fake *FakeSnapshotV1, namespace string) snapshotstoragek8siov1.VolumeSnapshotInterface {
	return &fakeVolumeSnapshots{
		gentype.NewFakeClientWithList[*v1.VolumeSnapshot, *v1.VolumeSnapshotList](
			fake.Fake,
			namespace,
			v1.SchemeGroupVersion.WithResource("volumesnapshots"),
			v1.SchemeGroupVersion.WithKind("VolumeSnapshot"),
			func() *v1.VolumeSnapshot { return &v1.VolumeSnapshot{} },
			func() *v1.VolumeSnapshotList { return &v1.VolumeSnapshotList{} },
			func(dst, src *v1.VolumeSnapshotList) { dst.ListMeta = src.ListMeta },
			func(list *v1.VolumeSnapshotList) []*v1.VolumeSnapshot { return gentype.ToPointerSlice(list.Items) },
			func(list *v1.VolumeSnapshotList, items []*v1.VolumeSnapshot) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
