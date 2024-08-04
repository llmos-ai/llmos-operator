package controller

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/leader"

	"github.com/llmos-ai/llmos-operator/pkg/controller/managedaddon"
	"github.com/llmos-ai/llmos-operator/pkg/controller/modelfile"
	"github.com/llmos-ai/llmos-operator/pkg/controller/notebook"
	"github.com/llmos-ai/llmos-operator/pkg/controller/raycluster"
	"github.com/llmos-ai/llmos-operator/pkg/controller/setting"
	"github.com/llmos-ai/llmos-operator/pkg/controller/upgrade"
	"github.com/llmos-ai/llmos-operator/pkg/controller/user"
	"github.com/llmos-ai/llmos-operator/pkg/indexeres"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type registerFunc func(context.Context, *config.Management) error

var registerFuncs = []registerFunc{
	indexeres.Register,
	upgrade.Register,
	setting.Register,
	modelfile.Register,
	user.Register,
	notebook.Register,
	raycluster.Register,
	managedaddon.Register,
	//storage.Register,
}

func register(ctx context.Context, mgmt *config.Management) error {
	for _, f := range registerFuncs {
		if err := f(ctx, mgmt); err != nil {
			return err
		}
	}
	return nil
}

func Register(ctx context.Context, mgmt *config.Management, threadiness int) error {
	go leader.RunOrDie(ctx, "", "llmos-operator-leader", mgmt.ClientSet, func(ctx context.Context) {
		if err := register(ctx, mgmt); err != nil {
			panic(err)
		}
		if err := mgmt.Start(threadiness); err != nil {
			panic(err)
		}
		<-ctx.Done()
	})
	return nil
}
