package server

import (
	"context"

	wc "github.com/oneblock-ai/webhook/pkg/config"
	ws "github.com/oneblock-ai/webhook/pkg/server"
	"github.com/rancher/wrangler/v3/pkg/k8scheck"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"k8s.io/client-go/rest"

	"github.com/llmos-ai/llmos-operator/pkg/config"
	sserver "github.com/llmos-ai/llmos-operator/pkg/server"
	"github.com/llmos-ai/llmos-operator/pkg/webhook"
	wconfig "github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

// WebhookServer defines the webhook webhookServer types
type WebhookServer struct {
	ctx           context.Context
	webhookServer *ws.WebhookServer
	restConfig    *rest.Config
}

// Options define the api webhookServer options
type Options struct {
	Context         context.Context
	KubeConfig      string
	HTTPSListenPort int
	Threadiness     int
	Namespace       string
	ReleaseName     string
	DevMode         bool
	DevURL          string

	config.CommonOptions
}

func NewServer(opts Options) (*WebhookServer, error) {
	s := &WebhookServer{
		ctx: opts.Context,
	}

	clientConfig, err := sserver.GetConfig(opts.KubeConfig)
	if err != nil {
		return s, err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return s, err
	}
	restConfig.RateLimiter = ratelimit.None
	s.restConfig = restConfig

	err = k8scheck.Wait(s.ctx, *restConfig)
	if err != nil {
		return nil, err
	}

	// set up a new webhook webhookServer
	webhookName := wconfig.GetWebhookName(opts.ReleaseName)
	s.webhookServer = ws.NewWebhookServer(opts.Context, restConfig, webhookName, &wc.Options{
		Namespace:       opts.Namespace,
		Threadiness:     opts.Threadiness,
		HTTPSListenPort: opts.HTTPSListenPort,
		DevMode:         opts.DevMode,
		DevURL:          opts.DevURL,
	})

	if err := webhook.Register(opts.Context, restConfig, s.webhookServer, opts.ReleaseName, opts.Threadiness); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *WebhookServer) ListenAndServe() error {
	if err := s.webhookServer.Start(); err != nil {
		return err
	}

	<-s.ctx.Done()
	return nil
}
