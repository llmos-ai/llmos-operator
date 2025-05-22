package modelservice

import (
	"reflect"
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

func TestBuildArgs(t *testing.T) {
	ms := &mlv1.ModelService{}
	ms.Spec.ModelName = "test-model"
	ms.Spec.ServedModelName = "served-test-model"
	ms.Spec.Template.Spec.Containers = []v1.Container{
		{
			Args: []string{"--some-arg=value", "--model=old-model"},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					vGPUNumber: resource.MustParse("1"),
				},
			},
		},
	}

	expectedArgs := []string{
		"--some-arg=value",
		"--model=test-model",
		"--served-model-name=served-test-model",
		"--tensor-parallel-size=" + strconv.Itoa(getVGPUNumber(ms)),
	}

	result := buildArgs(ms)

	if !utils.EqualIgnoreOrder(result, expectedArgs) {
		t.Errorf("Expected %v, got %v", expectedArgs, result)
	}
}

func TestBuildArgs_WithoutServedModelName(t *testing.T) {
	ms := &mlv1.ModelService{}
	ms.Spec.ModelName = "test-model"
	ms.Spec.ServedModelName = ""
	ms.Spec.Template.Spec.Containers = []v1.Container{
		{
			Args: []string{"--some-arg=value", "--model=old-model", "--served-model-name=my-name"},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					vGPUNumber: resource.MustParse("1"),
				},
			},
		},
	}

	expectedArgs := []string{
		"--some-arg=value",
		"--model=test-model",
		"--served-model-name=my-name",
		"--tensor-parallel-size=" + strconv.Itoa(getVGPUNumber(ms)),
	}

	result := buildArgs(ms)

	if !reflect.DeepEqual(result, expectedArgs) {
		t.Errorf("Expected %v, got %v", expectedArgs, result)
	}
}
