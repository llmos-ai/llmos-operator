package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

// Handler proxies requests to the rancher service
type Handler struct {
	Scheme string
	Host   string
}

const (
	ForwardedAPIHostHeader = "X-API-Host"
	ForwardedProtoHeader   = "X-Forwarded-Proto"
	ForwardedHostHeader    = "X-Forwarded-Host"
	PrefixHeader           = "X-API-URL-Prefix"
	LocalLLMApiPrefix      = "/local_llm_api/v1"
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	localLLMUrl := settings.LocalLLMServerURL.Get()
	url, err := url.Parse(localLLMUrl)
	if err != nil {
		logrus.Errorf("error parsing local LLM url: %v", err)
		_, _ = rw.Write([]byte(fmt.Sprintf("error parsing local LLM url: %v", err)))
		return
	}
	director := func(r *http.Request) {
		r.URL.Scheme = url.Scheme
		r.URL.Host = url.Host
		r.URL.Path = trimProxyPrefix(req.URL.Path)
		// set forwarded header
		r.Header.Set(ForwardedAPIHostHeader, GetLastExistValue(req.Host, req.Header.Get(ForwardedAPIHostHeader)))
		r.Header.Set(ForwardedHostHeader, GetLastExistValue(req.Host, req.Header.Get(ForwardedHostHeader)))
		r.Header.Set(ForwardedProtoHeader, GetLastExistValue(req.URL.Scheme, req.Header.Get(ForwardedProtoHeader)))
		r.Header.Set(PrefixHeader, GetLastExistValue(req.Header.Get(PrefixHeader)))
	}
	httpProxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	httpProxy.ServeHTTP(rw, req)
}

func GetLastExistValue(values ...string) string {
	var result string
	for _, value := range values {
		if value != "" {
			result = value
		}
	}
	return result
}

func trimProxyPrefix(path string) string {
	return strings.TrimPrefix(path, LocalLLMApiPrefix)
}
