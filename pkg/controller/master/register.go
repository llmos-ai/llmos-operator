package master

import (
	"context"

	steve "github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/leader"

	"github.com/llmos-ai/llmos-operator/pkg/controller/master/dataset"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/globalrole"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/managedaddon"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/model"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/modelservice"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/monitoring"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/namespace"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/node"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/notebook"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/raycluster"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/registry"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/roletemplate"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/roletemplatebinding"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/setting"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/token"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/upgrade"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master/user"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type registerFunc func(context.Context, *config.Management, config.Options) error

var registerFuncs = []registerFunc{
	upgrade.Register,
	setting.Register,
	user.Register,
	notebook.Register,
	raycluster.Register,
	managedaddon.Register,
	modelservice.Register,
	token.Register,
	globalrole.Register,
	roletemplate.Register,
	roletemplatebinding.Register,
	namespace.Register,
	node.Register,
	monitoring.Register,
	registry.Register,
	dataset.Register,
	model.Register,
}

func register(ctx context.Context, mgmt *config.Management, opts config.Options) error {
	for _, f := range registerFuncs {
		if err := f(ctx, mgmt, opts); err != nil {
			return err
		}
	}
	return nil
}

func Register(ctx context.Context, controllers *steve.Controllers, opts config.Options) error {
	scaled := config.ScaledWithContext(ctx)
	go leader.RunOrDie(ctx, "", "llmos-operator-leader", controllers.K8s, func(ctx context.Context) {
		if err := register(ctx, scaled.Management, opts); err != nil {
			panic(err)
		}
		if err := scaled.Management.Start(opts.Threadiness); err != nil {
			panic(err)
		}
		<-ctx.Done()
	})
	return nil
}
