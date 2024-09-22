package server

import (
	"context"

	dlserver "github.com/rancher/dynamiclistener/server"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/steve/pkg/accesscontrol"
	steve "github.com/rancher/steve/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/k8scheck"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/llmos-ai/llmos-operator/pkg/api"
	"github.com/llmos-ai/llmos-operator/pkg/auth"
	"github.com/llmos-ai/llmos-operator/pkg/controller/global"
	"github.com/llmos-ai/llmos-operator/pkg/controller/master"
	"github.com/llmos-ai/llmos-operator/pkg/data"
	"github.com/llmos-ai/llmos-operator/pkg/indexeres"
	sconfig "github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/server/ui"
)

type APIServer struct {
	ctx            context.Context
	clientSet      *kubernetes.Clientset
	scaled         *sconfig.Scaled
	steveServer    *steve.Server
	controllers    *steve.Controllers
	restConfig     *rest.Config
	startHooks     []StartHook
	postStartHooks []PostStartHook
}

type StartHook func(context.Context, *steve.Controllers, sconfig.Options) error
type PostStartHook func(int) error

func NewServer(opts sconfig.Options) (*APIServer, error) {
	s := &APIServer{
		ctx: opts.Context,
	}

	var err error
	kubeConfig, err := GetConfig(opts.KubeConfig)
	if err != nil {
		return nil, err
	}

	s.restConfig, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	s.restConfig.RateLimiter = ratelimit.None

	// Wait for k8s to be ready first
	if err = k8scheck.Wait(s.ctx, *s.restConfig); err != nil {
		return nil, err
	}

	s.clientSet, err = kubernetes.NewForConfig(s.restConfig)
	if err != nil {
		return nil, err
	}

	if err = s.setDefaults(opts); err != nil {
		return nil, err
	}

	// Configure the api ui
	ui.ConfigureAPIUI(s.steveServer.APIServer)

	s.startHooks = []StartHook{
		indexeres.Register,
		master.Register,
		global.Register,
	}

	s.postStartHooks = []PostStartHook{
		s.scaled.Start,
	}

	return s, s.start(opts)
}

func (s *APIServer) start(opts sconfig.Options) error {
	var err error
	for _, hook := range s.startHooks {
		if err = hook(s.ctx, s.controllers, opts); err != nil {
			return err
		}
	}

	// Register api schemas formatter
	if err = api.Register(s.ctx, s.steveServer); err != nil {
		return err
	}

	if err = s.controllers.Start(s.ctx); err != nil {
		return err
	}

	for _, hook := range s.postStartHooks {
		if err = hook(opts.Threadiness); err != nil {
			return err
		}
	}

	return nil

}

func (s *APIServer) setDefaults(opts sconfig.Options) error {
	factory, err := controller.NewSharedControllerFactoryFromConfig(s.restConfig, sconfig.Scheme)
	if err != nil {
		return err
	}

	factoryOpts := &generic.FactoryOptions{
		SharedControllerFactory: factory,
	}

	// Set up scaled config
	s.ctx, s.scaled, err = sconfig.SetupScaled(s.ctx, s.restConfig, factoryOpts)
	if err != nil {
		return err
	}

	s.controllers, err = steve.NewController(s.restConfig, factoryOpts)
	if err != nil {
		return err
	}

	asl := accesscontrol.NewAccessStore(s.ctx, true, s.controllers.RBAC)

	// Define the route handler after the scaled is set up
	r := NewRouter(s.scaled)

	// Define the auth middleware
	auth := auth.NewMiddleware(s.scaled)

	// Wait for webhooks to be registered before proceeding controller operations
	if err = WaitingWebhooks(s.ctx, s.clientSet, opts.ReleaseName); err != nil {
		return err
	}

	if err = data.Init(s.scaled.Management); err != nil {
		return err
	}

	// Set up a new API server
	s.steveServer, err = steve.New(s.ctx, s.restConfig, &steve.Options{
		Controllers:     s.controllers,
		Next:            r.Routes(),
		AccessSetLookup: asl,
		AuthMiddleware:  auth.AuthMiddleware,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *APIServer) ListenAndServe(opts sconfig.Options, listenOpts *dlserver.ListenOpts) error {
	return s.steveServer.ListenAndServe(s.ctx, opts.HTTPSListenPort, opts.HTTPListenPort, listenOpts)
}
