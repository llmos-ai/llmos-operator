package main

import (
	"os"
	"path/filepath"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/sirupsen/logrus"

	nvidiav1 "github.com/NVIDIA/gpu-operator/api/v1"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/v2/pkg/controller-gen"
	"github.com/rancher/wrangler/v2/pkg/controller-gen/args"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
)

const (
	kubeRayGV = "ray.io"
	nvidiaGV  = "nvidia.com"
)

func main() {
	_ = os.Unsetenv("GOPATH")

	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed getting pwd: %v", err)
	}

	header, err := os.ReadFile(filepath.Join(pwd, "/hack/boilerplate.go.txt"))
	if err != nil {
		logrus.Fatalf("failed reading header: %v", err)
	}

	config := &gen.Config{
		Header:  string(header),
		Target:  "./pkg/generated/ent",
		Package: "github.com/llmos-ai/llmos-operator/pkg/generated/ent",
		Features: []gen.Feature{
			gen.FeatureUpsert,
		},
	}
	if err = entc.Generate("./pkg/types/v1", config); err != nil {
		logrus.Fatalf("running database codegen: %v", err)
	}

	controllergen.Run(args.Options{
		OutputPackage: "github.com/llmos-ai/llmos-operator/pkg/generated",
		Boilerplate:   "hack/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"management.llmos.ai": {
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/management.llmos.ai/v1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
			"ml.llmos.ai": {
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/ml.llmos.ai/v1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
			upgradev1.SchemeGroupVersion.Group: {
				Types: []interface{}{
					upgradev1.Plan{},
				},
				GenerateClients: true,
			},
			kubeRayGV: {
				PackageName: kubeRayGV,
				Types: []interface{}{
					rayv1.RayCluster{},
					rayv1.RayJob{},
					rayv1.RayService{},
				},
				GenerateTypes:   false,
				GenerateClients: true,
			},
			nvidiaGV: {
				PackageName: nvidiaGV,
				Types: []interface{}{
					nvidiav1.ClusterPolicy{},
				},
				GenerateTypes:   false,
				GenerateClients: true,
			},
		},
	})
}
