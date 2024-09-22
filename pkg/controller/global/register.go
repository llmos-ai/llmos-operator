package global

import (
	"context"

	steve "github.com/rancher/steve/pkg/server"

	"github.com/llmos-ai/llmos-operator/pkg/controller/global/settings"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type registerFunc func(context.Context, *config.Scaled) error

var registerFuncs = []registerFunc{
	settings.Register,
}

func Register(ctx context.Context, _ *steve.Controllers, _ config.Options) error {
	scaled := config.ScaledWithContext(ctx)
	for _, f := range registerFuncs {
		if err := f(ctx, scaled); err != nil {
			return err
		}
	}
	return nil
}
