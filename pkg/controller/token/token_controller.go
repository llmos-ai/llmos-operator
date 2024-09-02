package token

import (
	"context"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/auth/tokens"
	ctlmgmtv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/management.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	tokenOnChangeName = "token.onChange"
	syncInterval      = 15 * time.Minute
)

type handler struct {
	tokens     ctlmgmtv1.TokenClient
	tokenCache ctlmgmtv1.TokenCache
}

func Register(ctx context.Context, management *config.Management) error {
	tokens := management.MgmtFactory.Management().V1().Token()

	h := &handler{
		tokens:     tokens,
		tokenCache: tokens.Cache(),
	}

	tokens.OnChange(ctx, tokenOnChangeName, h.OnChanged)

	go h.onCleanUpSync(ctx)
	return nil
}

func (h *handler) OnChanged(_ string, token *mgmtv1.Token) (*mgmtv1.Token, error) {
	if token == nil || token.DeletionTimestamp != nil {
		return token, nil
	}

	return h.reconcileStatus(token)
}

func (h *handler) reconcileStatus(token *mgmtv1.Token) (*mgmtv1.Token, error) {
	toUpdate := token.DeepCopy()

	tokens.SetTokenExpiresAt(toUpdate)
	toUpdate.Status.IsExpired = tokens.IsExpired(toUpdate)
	if !reflect.DeepEqual(token.Status, toUpdate.Status) {
		return h.tokens.UpdateStatus(toUpdate)
	}

	return token, nil
}

// onCleanUpSync periodically checks expired tokens and deletes them
func (h *handler) onCleanUpSync(ctx context.Context) {
	ticker := time.NewTicker(syncInterval)
	for {
		select {
		case <-ticker.C:
			tokenList, err := h.tokenCache.List(labels.Everything())
			if err != nil {
				logrus.Errorf("failed to list tokens: %v", err)
				continue
			}

			for _, token := range tokenList {
				if token.DeletionTimestamp != nil {
					continue
				}

				if token.Status.IsExpired || tokens.IsExpired(token) {
					if err := h.tokens.Delete(token.Name, &metav1.DeleteOptions{}); err != nil {
						logrus.Errorf("failed to delete token %s: %v", token.Name, err)
					}
				}
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
