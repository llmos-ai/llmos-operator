package clusterinfo

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
	"github.com/llmos-ai/llmos-operator/pkg/version"
)

type ClusterInfo struct {
	mgmt *config.Management
}

func NewClusterInfo(mgmt *config.Management) ClusterInfo {
	return ClusterInfo{
		mgmt: mgmt,
	}
}

func (c ClusterInfo) tokenAuthMiddleware(req *http.Request) error {
	secrets := c.mgmt.CoreFactory.Core().V1().Secret()
	secret, err := secrets.Get(constant.SystemNamespaceName, "local-k8s-state", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret: %v", err)
	}

	if secret.Data["agentToken"] == nil {
		return fmt.Errorf("local k8s node token is not set")
	}

	token := getRequestToken(req)
	if token == "" {
		return fmt.Errorf("token is not set")
	}

	if string(secret.Data["agentToken"]) != token {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func getRequestToken(req *http.Request) string {
	token := req.URL.Query().Get("token")
	if token != "" {
		return token
	}

	tokenStr := req.Header.Get("Authorization")
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		return tokenStr
	}

	return ""
}

func (c ClusterInfo) ReadyzHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		data := []byte("ok")
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "text/plain")
		rw.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = rw.Write(data)
	})
}

func (c ClusterInfo) ClusterInfo() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := c.tokenAuthMiddleware(req); err != nil {
			utils.ResponseError(rw, http.StatusUnauthorized, err)
			return
		}

		info, err := c.mgmt.ClientSet.ServerVersion()
		if err != nil {
			utils.ResponseError(rw, http.StatusInternalServerError, err)
			return
		}
		resp := map[string]string{
			"k8sVersion":           info.String(),
			"llmosOperatorVersion": version.FriendlyVersion(),
		}

		utils.ResponseOKWithBody(rw, resp)
	})
}
