package main

import (
	"os"

	upgradev1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/v2/pkg/controller-gen"
	"github.com/rancher/wrangler/v2/pkg/controller-gen/args"
)

func main() {
	_ = os.Unsetenv("GOPATH")
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
			upgradev1.SchemeGroupVersion.Group: {
				Types: []interface{}{
					upgradev1.Plan{},
				},
				GenerateClients: true,
			},
		},
	})
}
