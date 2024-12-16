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
	defaultImgRepo    = "ghcr.io/oneblock-ai"
	defaultImgVersion = "v0.1.0"
)

// SetDefaultNotebookImages set default notebook images
// resources please refer to: https://www.kubeflow.org/docs/components/notebooks/container-images/
func setDefaultNotebookImages() string {
	defaultImgs := map[string][]Image{
		"jupyter": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-scipy", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
				Default:        true,
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-pytorch", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-pytorch-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-pytorch-cuda", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-pytorch-cuda-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-tensorflow", defaultImgVersion),
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-tensorflow-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-tensorflow-cuda", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-tensorflow-cuda-full", defaultImgVersion),
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "jupyter-pipeline", defaultImgVersion),
				Description:    "JupyterLab + Elyra Pipeline",
			},
		},
		"code-server": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "code-server-python", defaultImgVersion),
				Description:    "Visual Studio Code + Conda Python",
				Default:        true,
			},
		},
		"rstudio": {
			{
				ContainerImage: fmt.Sprintf("%s/%s:%s", defaultImgRepo, "rstudio-tidyverse", defaultImgVersion),
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
