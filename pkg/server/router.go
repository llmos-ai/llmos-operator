package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rancher/apiserver/pkg/urlbuilder"

	"github.com/llmos-ai/llmos-controller/pkg/api/auth"
	"github.com/llmos-ai/llmos-controller/pkg/api/proxy"
	"github.com/llmos-ai/llmos-controller/pkg/api/publicui"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/server/ui"
)

type Router struct {
	mgmt *config.Management
}

func NewRouter(mgmt *config.Management) *Router {
	return &Router{
		mgmt: mgmt,
	}
}

// Routes adds additional customize routes to the default router
func (r *Router) Routes() http.Handler {
	m := mux.NewRouter()
	m.UseEncodedPath()
	m.StrictSlash(true)
	m.Use(urlbuilder.RedirectRewrite)

	// public auth handler
	authHandler := auth.NewAuthHandler(r.mgmt)
	m.Path("/v1-public/auth").Handler(authHandler)

	m.Handle("/", http.RedirectHandler("/dashboard/", http.StatusFound))

	vueUI := ui.Vue
	m.Handle("/dashboard/", vueUI.IndexFile())
	m.Handle("/dashboard", http.RedirectHandler("/dashboard/", http.StatusFound))
	m.PathPrefix("/dashboard/").Handler(vueUI.IndexFileOnNotFound())
	m.PathPrefix("/api-ui").Handler(vueUI.ServeAsset())
	m.Handle("/favicon.png", vueUI.ServeFaviconDashboard())
	m.Handle("/favicon.ico", vueUI.ServeFaviconDashboard())
	m.PathPrefix("/k8s/clusters/local").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		url := strings.TrimPrefix(req.URL.Path, "/k8s/clusters/local")
		if url == "" {
			url = "/"
		}
		http.Redirect(rw, req, url, http.StatusFound)
	})

	localLLMHandler := proxy.NewHandler()
	m.PathPrefix(proxy.LocalLLMApiPrefix).Handler(localLLMHandler)

	// public handlers
	publicHandler := publicui.NewPublicHandler()
	m.Path("/v1-public/ui").Handler(publicHandler)

	return m
}
