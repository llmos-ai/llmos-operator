package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rancher/steve/pkg/ui"

	"github.com/llmos-ai/llmos-controller/pkg/api/publicui"
	"github.com/llmos-ai/llmos-controller/pkg/server/config"
	"github.com/llmos-ai/llmos-controller/pkg/settings"
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
	vue := ui.NewUIHandler(&ui.Options{
		Index:          settings.UIIndex.Get,
		Path:           settings.UIPath.Get,
		Offline:        IsOffline,
		ReleaseSetting: settings.IsRelease,
	})

	m := mux.NewRouter()
	m.UseEncodedPath()

	m.Handle("/", http.RedirectHandler("/dashboard/", http.StatusFound))
	m.Handle("/dashboard", http.RedirectHandler("/dashboard/", http.StatusFound))
	m.Handle("/dashboard/", vue.IndexFile())
	m.Handle("/favicon.png", vue.ServeFaviconDashboard())
	m.Handle("/favicon.ico", vue.ServeFaviconDashboard())
	m.PathPrefix("/dashboard/").Handler(vue.IndexFileOnNotFound())
	m.PathPrefix("/k8s/clusters/local").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		url := strings.TrimPrefix(req.URL.Path, "/k8s/clusters/local")
		if url == "" {
			url = "/"
		}
		http.Redirect(rw, req, url, http.StatusFound)
	})

	publicHandler := publicui.NewPublicHandler()
	m.Path("/v1-public/ui").Handler(publicHandler)

	return m
}

func IsOffline() string {
	switch settings.UISource.Get() {
	case "auto":
		return "dynamic"
	case "external":
		return "false"
	}
	return "true"
}
