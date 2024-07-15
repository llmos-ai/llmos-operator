/*
Copyright 2024 llmos.ai.

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

package v1

import (
	v1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"github.com/rancher/wrangler/v2/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	schemes.Register(v1.AddToScheme)
}

type Interface interface {
	ModelFile() ModelFileController
	Notebook() NotebookController
}

func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &version{
		controllerFactory: controllerFactory,
	}
}

type version struct {
	controllerFactory controller.SharedControllerFactory
}

func (v *version) ModelFile() ModelFileController {
	return generic.NewNonNamespacedController[*v1.ModelFile, *v1.ModelFileList](schema.GroupVersionKind{Group: "ml.llmos.ai", Version: "v1", Kind: "ModelFile"}, "modelfiles", v.controllerFactory)
}

func (v *version) Notebook() NotebookController {
	return generic.NewController[*v1.Notebook, *v1.NotebookList](schema.GroupVersionKind{Group: "ml.llmos.ai", Version: "v1", Kind: "Notebook"}, "notebooks", true, v.controllerFactory)
}
