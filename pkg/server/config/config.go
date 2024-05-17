package config

import (
	"context"

	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v2/pkg/apply"
	appsv1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/apps"
	corev1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/core"
	rbacv1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/rbac"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"github.com/rancher/wrangler/v2/pkg/start"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	ctlmgmtv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/management.llmos.ai"
	nvidiav1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/nvidia.com"
	"github.com/llmos-ai/llmos-controller/pkg/generated/controllers/upgrade.cattle.io"
)

type Management struct {
	ctx        context.Context
	Namespace  string
	ClientSet  *kubernetes.Clientset
	RestConfig *rest.Config
	Apply      apply.Apply
	Scheme     *runtime.Scheme

	CoreFactory      *corev1.Factory
	AppsFactory      *appsv1.Factory
	RbacFactory      *rbacv1.Factory
	LLMOSMgmtFactory *ctlmgmtv1.Factory
	UpgradeFactory   *upgrade.Factory
	NvidiaFactory    *nvidiav1.Factory

	starters []start.Starter
}

func SetupManagement(ctx context.Context, restConfig *rest.Config, namespace string) (*Management, error) {
	mgmt := &Management{
		ctx:       ctx,
		Namespace: namespace,
		Scheme:    Scheme,
	}

	apply, err := apply.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.Apply = apply

	mgmt.RestConfig = restConfig
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.ClientSet = clientSet

	factory, err := controller.NewSharedControllerFactoryFromConfig(mgmt.RestConfig, Scheme)
	if err != nil {
		return nil, err
	}

	factoryOpts := &generic.FactoryOptions{
		SharedControllerFactory: factory,
	}

	core, err := corev1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.CoreFactory = core
	mgmt.starters = append(mgmt.starters, core)

	rbac, err := rbacv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.RbacFactory = rbac
	mgmt.starters = append(mgmt.starters, rbac)

	llmosMgmt, err := ctlmgmtv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.LLMOSMgmtFactory = llmosMgmt
	mgmt.starters = append(mgmt.starters, llmosMgmt)

	upgrade, err := upgrade.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.UpgradeFactory = upgrade
	mgmt.starters = append(mgmt.starters, upgrade)

	nvidia, err := nvidiav1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.NvidiaFactory = nvidia
	mgmt.starters = append(mgmt.starters, nvidia)

	return mgmt, nil
}

func (m *Management) Start(threadiness int) error {
	return start.All(m.ctx, threadiness, m.starters...)
}
