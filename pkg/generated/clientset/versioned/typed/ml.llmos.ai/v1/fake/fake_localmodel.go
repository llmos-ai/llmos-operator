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
	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	mlllmosaiv1 "github.com/llmos-ai/llmos-operator/pkg/generated/clientset/versioned/typed/ml.llmos.ai/v1"
	gentype "k8s.io/client-go/gentype"
)

// fakeLocalModels implements LocalModelInterface
type fakeLocalModels struct {
	*gentype.FakeClientWithList[*v1.LocalModel, *v1.LocalModelList]
	Fake *FakeMlV1
}

func newFakeLocalModels(fake *FakeMlV1, namespace string) mlllmosaiv1.LocalModelInterface {
	return &fakeLocalModels{
		gentype.NewFakeClientWithList[*v1.LocalModel, *v1.LocalModelList](
			fake.Fake,
			namespace,
			v1.SchemeGroupVersion.WithResource("localmodels"),
			v1.SchemeGroupVersion.WithKind("LocalModel"),
			func() *v1.LocalModel { return &v1.LocalModel{} },
			func() *v1.LocalModelList { return &v1.LocalModelList{} },
			func(dst, src *v1.LocalModelList) { dst.ListMeta = src.ListMeta },
			func(list *v1.LocalModelList) []*v1.LocalModel { return gentype.ToPointerSlice(list.Items) },
			func(list *v1.LocalModelList, items []*v1.LocalModel) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
