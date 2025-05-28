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

// ProxyHandler proxies requests to apps api
type ProxyHandler struct {
	Scheme string
	Host   string
}

const (
	ForwardedAPIHostHeader = "X-API-Host"
	ForwardedProtoHeader   = "X-Forwarded-Proto"
	ForwardedHostHeader    = "X-Forwarded-Host"
	PrefixHeader           = "X-API-URL-Prefix"
	AppsProxyPrefix        = "/proxy/apps"
	VectorDBProxyPrefix    = "/proxy/vectorDB"
)

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{}
}

func (h *ProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	proxyType, error := getProxyType(req)
	if proxyType == "" || error != nil {
		logrus.Errorf("invalid proxy type: %v", proxyType)
		_, _ = rw.Write([]byte(fmt.Sprintf("invalid proxy type: %v", proxyType)))
		return
	}

	var proxyServerUrl string
	var proxyPathPrefix string

	switch proxyType {
	case "apps":
		proxyServerUrl = settings.ProxyAppsServerUrl.Get()
		proxyPathPrefix = AppsProxyPrefix
	case "vectorDB":
		proxyServerUrl = settings.ProxyVectorServerUrl.Get()
		proxyPathPrefix = VectorDBProxyPrefix
	default:
		logrus.Errorf("unsupported proxy type: %v", proxyType)
		_, _ = rw.Write([]byte(fmt.Sprintf("unsupported proxy type: %v", proxyType)))
		return
	}

	url, err := url.Parse(proxyServerUrl)
	if err != nil {
		logrus.Errorf("error parsing proxy server url: %v", err)
		_, _ = rw.Write([]byte(fmt.Sprintf("error parsing proxy server url: %v", err)))
		return
	}
	logrus.Debugf("proxy type: %v", proxyType)
	logrus.Debugf("apps request: %v", req.URL.Path)
	logrus.Debugf("proxy server url: %v", proxyServerUrl)
	logrus.Debugf("proxy path prefix: %v", trimProxyPrefix(req.URL.Path, proxyPathPrefix))

	director := func(r *http.Request) {
		r.URL.Scheme = url.Scheme
		r.URL.Host = url.Host
		r.URL.Path = trimProxyPrefix(req.URL.Path, proxyPathPrefix)
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

// getProxyType returns the proxy type from the request path
func getProxyType(r *http.Request) (string, error) {
	// Remove any leading/trailing slashes, then split on “/”
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// parts[0] == "proxy", parts[1] == "apps", ...
	if len(parts) < 2 {
		return "", fmt.Errorf("missing path segment")
	}
	return parts[1], nil
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

func trimProxyPrefix(path string, pathPrefix string) string {
	return strings.TrimPrefix(path, pathPrefix)
}
