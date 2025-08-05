package settings

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

type Image struct {
	ContainerImage string `json:"containerImage,omitempty"`
	Description    string `json:"description,omitempty"`
	Default        bool   `json:"default,omitempty"`
}

const (
	defaultImgVersion = "v0.1.0"
)

// SetDefaultNotebookImages set default notebook images
// resources please refer to: https://www.kubeflow.org/docs/components/notebooks/container-images/
func setDefaultNotebookImages() string {
	registry := GlobalSystemImageRegistry.Get()
	defaultImgs := map[string][]Image{
		"jupyter": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-scipy", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
				Default:        true,
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-pytorch", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-pytorch-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-pytorch-cuda", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-pytorch-cuda-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-tensorflow", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-tensorflow-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-tensorflow-cuda", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-tensorflow-cuda-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/jupyter-pipeline", defaultImgVersion),
				Description:    "JupyterLab + Elyra Pipeline",
			},
		},
		"code-server": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/codeserver-python", defaultImgVersion),
				Description:    "Visual Studio Code + Conda Python",
				Default:        true,
			},
		},
		"rstudio": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", registry, "oneblock-ai/rstudio-tidyverse", defaultImgVersion),
				Description:    "RStudio + Tidyverse",
				Default:        true,
			},
		},
	}
	stringImg, err := json.Marshal(defaultImgs)
	if err != nil {
		logrus.Fatal(err)
	}
	return string(stringImg)
}
