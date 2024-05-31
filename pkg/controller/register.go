package controller

import (
	"context"

	"github.com/rancher/wrangler/v2/pkg/leader"

	"github.com/llmos-ai/llmos-controller/pkg/controller/modelfile"
	"github.com/llmos-ai/llmos-controller/pkg/controller/setting"
	"github.com/llmos-ai/llmos-controller/pkg/controller/upgrade"
	"github.com/llmos-ai/llmos-controller/pkg/controller/user"
	"github.com/llmos-ai/llmos-controller/pkg/indexeres"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
)

type registerFunc func(context.Context, *config.Management) error

var registerFuncs = []registerFunc{
	indexeres.Register,
	upgrade.Register,
	setting.Register,
	modelfile.Register,
	user.Register,
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
	go leader.RunOrDie(ctx, "", "llmos-controller-leader", mgmt.ClientSet, func(ctx context.Context) {
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
