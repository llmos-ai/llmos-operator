package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

// ProxyHandler proxies requests to backend services based on request path
type ProxyHandler struct{}

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
	proxyType, err := getProxyType(req)
	if proxyType == "" || err != nil {
		logrus.Errorf("invalid proxy type: %v", proxyType)
		http.Error(rw, fmt.Sprintf("invalid proxy type: %v", err), http.StatusBadRequest)
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
		http.Error(rw, fmt.Sprintf("unsupported proxy type: %v", proxyType), http.StatusBadRequest)
		return
	}

	upstreamBase, err := url.Parse(proxyServerUrl)
	if err != nil {
		logrus.Errorf("error parsing proxy server upstreamBase: %v", err)
		http.Error(rw, "error parsing proxy server upstreamBase", http.StatusBadRequest)
		return
	}
	logrus.Debugf("forward to proxy: %s%s, type: %v", proxyServerUrl,
		trimProxyPrefix(req.URL.Path, proxyPathPrefix), proxyType)

	director := func(r *http.Request) {
		r.URL.Scheme = upstreamBase.Scheme
		r.URL.Host = upstreamBase.Host
		r.URL.Path = trimProxyPrefix(req.URL.Path, proxyPathPrefix)
		// set forwarded header
		r.Header.Set(ForwardedAPIHostHeader, GetLastExistValue(req.Host, req.Header.Get(ForwardedAPIHostHeader)))
		r.Header.Set(ForwardedHostHeader, GetLastExistValue(req.Host, req.Header.Get(ForwardedHostHeader)))
		r.Header.Set(ForwardedProtoHeader, GetLastExistValue(req.URL.Scheme, req.Header.Get(ForwardedProtoHeader)))
		r.Header.Set(PrefixHeader, GetLastExistValue(req.Header.Get(PrefixHeader)))
	}

	baseTrans := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		},
	}

	httpProxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: baseTrans,
		ModifyResponse: func(resp *http.Response) error {
			// Only care about HTTP 307 (Temporary Redirect) for now
			if resp.StatusCode != http.StatusTemporaryRedirect {
				return nil
			}

			locRaw := resp.Header.Get("Location")
			if locRaw == "" {
				return fmt.Errorf("received 307 but no Location header")
			}

			// Close the original 307 response body so we don't leak.
			if err := resp.Body.Close(); err != nil {
				logrus.Errorf("error closing proxy response body: %s", err.Error())
			}

			locURL, err := upstreamBase.Parse(locRaw)
			if err != nil {
				return fmt.Errorf("invalid redirect URL: %s", err.Error())
			}
			// resolve it against the upstreamBase so we never end up pointing back to the api server(or localhost)
			locURL.Host = upstreamBase.Host
			logrus.Debugf("redirecting url to %s", locURL)

			newReq := &http.Request{
				Method: resp.Request.Method,
				URL:    locURL,
				Header: make(http.Header),
			}

			// If the original request had a body (e.g. a POST/PUT), we need to buffer
			// that body ahead of time. Here, we assume GET/HEAD will have empty body.
			if (req.Method != "GET" && req.Method != "HEAD") && resp.Request.Body != nil {
				bodyBytes, err := io.ReadAll(resp.Request.Body)
				if err != nil {
					return fmt.Errorf("failed to read request body: %s", err.Error())
				}
				resp.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				newReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			// Copy all relevant headers from the original request (cookies, auth, etc.):
			for k, vv := range resp.Request.Header {
				for _, v := range vv {
					newReq.Header.Add(k, v)
				}
			}

			newResp, err := baseTrans.RoundTrip(newReq)
			if err != nil {
				return fmt.Errorf("following redirect: %v", err)
			}

			// Overwrite everything in resp with newResp:
			*resp = *newResp
			return nil
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
