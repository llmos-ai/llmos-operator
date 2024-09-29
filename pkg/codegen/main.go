package main

import (
	"os"
	"path/filepath"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	nvidiav1 "github.com/NVIDIA/gpu-operator/api/nvidia/v1"
	helmv1 "github.com/k3s-io/helm-controller/pkg/apis/helm.cattle.io/v1"
	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	rookv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/sirupsen/logrus"
	storagev1 "k8s.io/api/storage/v1"
)

const (
	kubeRayGV = "ray.io"
	nvidiaGV  = "nvidia.com"
	rookGV    = "ceph.rook.io"
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
				PackageName: "management.llmos.ai",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/management.llmos.ai/v1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
			"ml.llmos.ai": {
				PackageName: "ml.llmos.ai",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/ml.llmos.ai/v1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
			upgradev1.SchemeGroupVersion.Group: {
				PackageName: upgradev1.SchemeGroupVersion.Group,
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
			rookGV: {
				PackageName: rookGV,
				Types: []interface{}{
					rookv1.CephCluster{},
					rookv1.CephBlockPool{},
					rookv1.CephFilesystem{},
					rookv1.CephFilesystemSubVolumeGroup{},
				},
				GenerateTypes:   false,
				GenerateClients: true,
			},
			storagev1.SchemeGroupVersion.Group: {
				PackageName: storagev1.GroupName,
				Types: []interface{}{
					storagev1.StorageClass{},
				},
				GenerateTypes:   false,
				GenerateClients: true,
			},
			helmv1.SchemeGroupVersion.Group: {
				PackageName: helmv1.SchemeGroupVersion.Group,
				Types: []interface{}{
					helmv1.HelmChart{},
				},
				GenerateTypes:   false,
				GenerateClients: true,
			},
		},
	})
}
