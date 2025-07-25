package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/settings"
)

// ModelsHandler proxies requests to the url
type ModelsHandler struct {
	Scheme string
	Host   string
}

const (
	huggingFaceEndpoint = "https://huggingface.co"
)

var (
	allowedSites = []string{
		huggingFaceEndpoint,
		"https://hf-mirror.com",
		"https://modelscope.cn",
		"https://www.modelscope.cn",
	}

	headerSkipped = map[string]bool{
		"host":               true,
		"port":               true,
		"proto":              true,
		"referer":            true,
		"server":             true,
		"content-length":     true,
		"transfer-encoding":  true,
		"cookie":             true,
		"x-forwarded-host":   true,
		"x-forwarded-port":   true,
		"x-forwarded-proto":  true,
		"x-forwarded-server": true,
	}
)

func NewModelsHandler() *ModelsHandler {
	return &ModelsHandler{}
}

func (h *ModelsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	modelsProxyHandler(rw, req)
}

func modelsProxyHandler(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	hfToken := r.URL.Query().Get("hf_token")
	if urlStr == "" {
		http.Error(w, "url parameter is missing", http.StatusBadRequest)
		return
	}

	if err := validateURL(urlStr); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	urlStr = replaceHFEndpoint(urlStr)
	forwardedHeaders := processHeaders(r, hfToken)

	client := &http.Client{Timeout: 60 * time.Second}
	request, err := http.NewRequest(r.Method, urlStr, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	request.Header = forwardedHeaders

	resp, err := client.Do(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.Errorf("error closing proxy response body: %s", err.Error())
		}
	}()

	// Handle 307 Temporary Redirect by following the redirect
	if resp.StatusCode == http.StatusTemporaryRedirect {
		location := resp.Header.Get("Location")
		if location == "" {
			http.Error(w, "Redirect location header is missing", http.StatusInternalServerError)
			return
		}

		// Validate the redirect URL
		redirectUrl := fmt.Sprintf("%s://%s%s", request.URL.Scheme, request.URL.Host, location)
		if err := validateURL(redirectUrl); err != nil {
			http.Error(w, fmt.Sprintf("Redirect URL not allowed: %s", err.Error()), http.StatusForbidden)
			return
		}

		// Make a new request to the redirect URL
		redirectReq, err := http.NewRequest(r.Method, redirectUrl, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		redirectReq.Header = forwardedHeaders

		resp, err = client.Do(redirectReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logrus.Errorf("error closing redirect response body: %s", err.Error())
			}
		}()
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusOK {
		if _, err := io.Copy(w, resp.Body); err != nil {
			http.Error(w, "Error writing response body", http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := w.Write([]byte(resp.Status)); err != nil {
			log.Println("Error writing response status:", err)
		}
	}
}

func validateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid url query parameter")
	}

	for _, site := range allowedSites {
		if site == parsedURL.Scheme+"://"+parsedURL.Host {
			return nil
		}
	}
	return fmt.Errorf("this site is not allowed")
}

func replaceHFEndpoint(rawURL string) string {
	hfEndpoint := settings.HuggingFaceEndpoint.Get()
	if hfEndpoint != "" && strings.HasPrefix(rawURL, huggingFaceEndpoint) {
		hfEndpoint = strings.TrimRight(hfEndpoint, "/")
		return strings.Replace(rawURL, huggingFaceEndpoint, hfEndpoint, 1)
	}
	return rawURL
}

func processHeaders(req *http.Request, hfToken string) http.Header {
	newHeaders := http.Header{}
	for key, values := range req.Header {
		keyLower := strings.ToLower(key)
		if headerSkipped[keyLower] {
			continue
		} else {
			newHeaders[key] = values
		}
	}

	if hfToken != "" {
		newHeaders.Set("Authorization", fmt.Sprintf("Bearer %s", hfToken))
		return newHeaders
	}
	return newHeaders
}
