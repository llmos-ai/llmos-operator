package main

import (
	"os"
	"path/filepath"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/sirupsen/logrus"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/v2/pkg/controller-gen"
	"github.com/rancher/wrangler/v2/pkg/controller-gen/args"
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
		Package: "github.com/llmos-ai/llmos-controller/pkg/generated/ent",
		Features: []gen.Feature{
			gen.FeatureUpsert,
		},
	}
	if err = entc.Generate("./pkg/types/v1", config); err != nil {
		logrus.Fatalf("running database codegen: %v", err)
	}

	controllergen.Run(args.Options{
		OutputPackage: "github.com/llmos-ai/llmos-controller/pkg/generated",
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
		},
	})
}
