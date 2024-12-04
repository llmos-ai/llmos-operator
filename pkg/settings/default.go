package settings

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type Image struct {
	ContainerImage string `json:"containerImage,omitempty"`
	Description    string `json:"description,omitempty"`
	Default        bool   `json:"default,omitempty"`
}

const defaultImgVersion = "v0.1.0"

// SetDefaultNotebookImages set default notebook images
// resources please refer to: https://www.kubeflow.org/docs/components/notebooks/container-images/
func setDefaultNotebookImages() string {
	defaultImgs := map[string][]Image{
		"jupyter": {
			{
				ContainerImage: "oneblockai/jupyter-scipy:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch",
				Default:        true,
			},
			{
				ContainerImage: "oneblockai/jupyter-pytorch:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: "oneblockai/jupyter-pytorch-full:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: "oneblockai/jupyter-pytorch-cuda:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: "oneblockai/jupyter-pytorch-cuda-full:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: "oneblockai/jupyter-tensorflow:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch",
			},
			{
				ContainerImage: "oneblockai/jupyter-tensorflow-full:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + Common Packages",
			},
			{
				ContainerImage: "oneblockai/jupyter-tensorflow-cuda:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + CUDA",
			},
			{
				ContainerImage: "oneblockai/jupyter-tensorflow-cuda-full:" + defaultImgVersion,
				Description:    "JupyterLab + PyTorch + CUDA + Common Packages",
			},
			{
				ContainerImage: "oneblockai/jupyter-pipeline:" + defaultImgVersion,
				Description:    "JupyterLab + Elyra Pipeline",
			},
		},
		"code-server": {
			{
				ContainerImage: "oneblockai/codeserver-python:" + defaultImgVersion,
				Description:    "Visual Studio Code + Conda Python",
				Default:        true,
			},
		},
		"rstudio": {
			{
				ContainerImage: "oneblockai/rstudio-tidyverse:" + defaultImgVersion,
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
