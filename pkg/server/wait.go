package server

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	wconfig "github.com/llmos-ai/llmos-operator/pkg/webhook/config"
)

const (
	PollingInterval = 5 * time.Second
	PollingTimeout  = 2 * time.Minute
)

// WaitingWebhooks waits for the admission webhook server to registered successfully.
func WaitingWebhooks(ctx context.Context, clientSet *kubernetes.Clientset, releaseName string) error {
	webhookName := wconfig.GetWebhookName(releaseName)
	return wait.PollUntilContextTimeout(ctx, PollingInterval, PollingTimeout, true,
		func(ctx context.Context) (bool, error) {

			logrus.Infof("Waiting for ValidatingWebhookConfiguration %s...", webhookName)
			_, err := clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
				ctx, webhookName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			logrus.Infof("Waiting for MutatingWebhookConfiguration %s...", webhookName)
			_, err = clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
				ctx, webhookName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			logrus.Infof("Admission webhooks are ready")
			return true, nil
		})
}
