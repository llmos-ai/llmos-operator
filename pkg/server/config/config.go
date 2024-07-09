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
	ctlmlv1 "github.com/llmos-ai/llmos-controller/pkg/generated/controllers/ml.llmos.ai"
	"github.com/llmos-ai/llmos-controller/pkg/generated/controllers/upgrade.cattle.io"
	"github.com/llmos-ai/llmos-controller/pkg/generated/ent"
)

type Management struct {
	Ctx        context.Context
	Namespace  string
	ClientSet  *kubernetes.Clientset
	RestConfig *rest.Config
	Apply      apply.Apply
	Scheme     *runtime.Scheme
	EntClient  *ent.Client

	CoreFactory    *corev1.Factory
	AppsFactory    *appsv1.Factory
	RbacFactory    *rbacv1.Factory
	MgmtFactory    *ctlmgmtv1.Factory
	UpgradeFactory *upgrade.Factory
	LLMFactory     *ctlmlv1.Factory

	starters []start.Starter
}

func SetupManagement(ctx context.Context, restConfig *rest.Config,
	namespace string) (*Management, error) {
	mgmt := &Management{
		Ctx:       ctx,
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

	apps, err := appsv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.AppsFactory = apps
	mgmt.starters = append(mgmt.starters, apps)

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
	mgmt.MgmtFactory = llmosMgmt
	mgmt.starters = append(mgmt.starters, llmosMgmt)

	llm, err := ctlmlv1.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.LLMFactory = llm
	mgmt.starters = append(mgmt.starters, llm)

	upgrade, err := upgrade.NewFactoryFromConfigWithOptions(restConfig, factoryOpts)
	if err != nil {
		return nil, err
	}
	mgmt.UpgradeFactory = upgrade
	mgmt.starters = append(mgmt.starters, upgrade)

	return mgmt, nil
}

func (m *Management) Start(threadiness int) error {
	return start.All(m.Ctx, threadiness, m.starters...)
}

func (m *Management) SetEntClient(client *ent.Client) {
	m.EntClient = client
}

func (m *Management) GetEntClient() *ent.Client {
	return m.EntClient
}
