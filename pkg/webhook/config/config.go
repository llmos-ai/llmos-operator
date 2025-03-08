package config

import (
	"context"
	"fmt"

	"github.com/rancher/lasso/pkg/controller"
	appsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/rest"

	helmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/helm.cattle.io"
	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai"
	rayv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ray.io"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type Management struct {
	ctx         context.Context
	ReleaseName string
	RestConfig  *rest.Config

	MgmtFactory *mgmtv1.Factory
	LLMFactory  *mlv1.Factory
	CoreFactory *corev1.Factory
	AppsFactory *appsv1.Factory
	RayFactory  *rayv1.Factory
	HelmFactory *helmv1.Factory
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

	mgmt.LLMFactory, err = mlv1.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.starters = append(mgmt.starters, mgmt.LLMFactory)

	core, err := corev1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.CoreFactory = core
	mgmt.starters = append(mgmt.starters, core)

	apps, err := appsv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.AppsFactory = apps
	mgmt.starters = append(mgmt.starters, apps)

	kuberay, err := rayv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.RayFactory = kuberay
	mgmt.starters = append(mgmt.starters, kuberay)

	helm, err := helmv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.HelmFactory = helm
	mgmt.starters = append(mgmt.starters, helm)

	return mgmt, nil
}

func (m *Management) Start(threadiness int) error {
	return start.All(m.ctx, threadiness, m.starters...)
}

func GetWebhookName(releaseName string) string {
	return fmt.Sprintf("%s-webhook", releaseName)
}
