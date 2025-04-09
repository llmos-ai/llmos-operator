package config

import (
	"context"

	"github.com/rancher/wrangler/v3/pkg/apply"
	appsv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/apps"
	batchv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch"
	corev1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	rbacv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/llmos-ai/llmos-operator/pkg/config"
	rookv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ceph.rook.io"
	helmv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/helm.cattle.io"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai"
	nvidiav1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/nvidia.com"
	kuberayv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ray.io"
	storagev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/storage.k8s.io"
	upgradev1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/upgrade.cattle.io"
	"github.com/llmos-ai/llmos-operator/pkg/generated/ent"
)

// Options define the api server options
type Options struct {
	Context         context.Context
	HTTPListenPort  int
	HTTPSListenPort int
	Threadiness     int
	config.CommonOptions
}

type (
	_scaledKey struct{}
)

type Scaled struct {
	Ctx        context.Context
	ClientSet  *kubernetes.Clientset
	Management *Management

	CoreFactory *corev1.Factory
	MgmtFactory *ctlmgmtv1.Factory
	starters    []start.Starter
}

type Management struct {
	Ctx       context.Context
	ClientSet *kubernetes.Clientset
	Apply     apply.Apply
	EntClient *ent.Client

	CoreFactory    *corev1.Factory
	AppsFactory    *appsv1.Factory
	RbacFactory    *rbacv1.Factory
	BatchFactory   *batchv1.Factory
	StorageFactory *storagev1.Factory
	MgmtFactory    *ctlmgmtv1.Factory
	UpgradeFactory *upgradev1.Factory
	LLMFactory     *ctlmlv1.Factory
	KubeRayFactory *kuberayv1.Factory
	NvidiaFactory  *nvidiav1.Factory
	RookFactory    *rookv1.Factory
	HelmFactory    *helmv1.Factory

	starters []start.Starter
}

func SetupScaled(ctx context.Context, restConfig *rest.Config, opts *generic.FactoryOptions) (
	context.Context, *Scaled, error) {
	scaled := &Scaled{
		Ctx: ctx,
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	scaled.ClientSet = clientSet

	mgmt, err := ctlmgmtv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, nil, err
	}
	scaled.MgmtFactory = mgmt
	scaled.starters = append(scaled.starters, mgmt)

	core, err := corev1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, nil, err
	}
	scaled.CoreFactory = core
	scaled.starters = append(scaled.starters, core)

	scaled.Management, err = setupManagement(ctx, restConfig, opts)
	if err != nil {
		return nil, nil, err
	}

	return context.WithValue(scaled.Ctx, _scaledKey{}, scaled), scaled, nil
}

func setupManagement(ctx context.Context, restConfig *rest.Config, opts *generic.FactoryOptions) (*Management, error) {
	mgmt := &Management{
		Ctx: ctx,
	}

	apply, err := apply.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.Apply = apply

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	mgmt.ClientSet = clientSet

	core, err := corev1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.CoreFactory = core
	mgmt.starters = append(mgmt.starters, core)

	apps, err := appsv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.AppsFactory = apps
	mgmt.starters = append(mgmt.starters, apps)

	rbac, err := rbacv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.RbacFactory = rbac
	mgmt.starters = append(mgmt.starters, rbac)

	batch, err := batchv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.BatchFactory = batch
	mgmt.starters = append(mgmt.starters, batch)

	storage, err := storagev1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.StorageFactory = storage
	mgmt.starters = append(mgmt.starters, storage)

	llmosMgmt, err := ctlmgmtv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.MgmtFactory = llmosMgmt
	mgmt.starters = append(mgmt.starters, llmosMgmt)

	llm, err := ctlmlv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.LLMFactory = llm
	mgmt.starters = append(mgmt.starters, llm)

	upgrade, err := upgradev1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.UpgradeFactory = upgrade
	mgmt.starters = append(mgmt.starters, upgrade)

	kubeRay, err := kuberayv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.KubeRayFactory = kubeRay
	mgmt.starters = append(mgmt.starters, kubeRay)

	nvidia, err := nvidiav1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.NvidiaFactory = nvidia
	mgmt.starters = append(mgmt.starters, nvidia)

	rook, err := rookv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.RookFactory = rook
	mgmt.starters = append(mgmt.starters, rook)

	helm, err := helmv1.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	mgmt.HelmFactory = helm
	mgmt.starters = append(mgmt.starters, helm)

	return mgmt, nil
}

func ScaledWithContext(ctx context.Context) *Scaled {
	return ctx.Value(_scaledKey{}).(*Scaled)
}

func (s *Scaled) Start(threads int) error {
	return start.All(s.Ctx, threads, s.starters...)
}

func (m *Management) Start(threads int) error {
	return start.All(m.Ctx, threads, m.starters...)
}

func (m *Management) SetEntClient(client *ent.Client) {
	m.EntClient = client
}

func (m *Management) GetEntClient() *ent.Client {
	return m.EntClient
}
