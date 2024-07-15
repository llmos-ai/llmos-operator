package config

import (
	"context"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"github.com/rancher/wrangler/v2/pkg/start"
	"k8s.io/client-go/rest"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai"
	rayv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ray.io"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type Management struct {
	ctx         context.Context
	ReleaseName string
	RestConfig  *rest.Config

	MgmtFactory *mgmtv1.Factory
	RayFactory  *rayv1.Factory
	starters    []start.Starter
}

func SetupManagement(ctx context.Context, restConfig *rest.Config, releaseName string) (*Management, error) {
	mgmt := &Management{
		ctx:         ctx,
		RestConfig:  restConfig,
		ReleaseName: releaseName,
	}

	factory, err := controller.NewSharedControllerFactoryFromConfig(mgmt.RestConfig, config.Scheme)
	if err != nil {
		return nil, err
	}

	factoryOpts := &generic.FactoryOptions{
		SharedControllerFactory: factory,
	}

	mgmt.MgmtFactory, err = mgmtv1.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.starters = append(mgmt.starters, mgmt.MgmtFactory)

	kuberay, err := rayv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.RayFactory = kuberay
	mgmt.starters = append(mgmt.starters, kuberay)

	return mgmt, nil
}

func (m *Management) Start(threadiness int) error {
	return start.All(m.ctx, threadiness, m.starters...)
}
