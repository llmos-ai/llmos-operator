package webhook

import (
	"context"
	"fmt"

	ws "github.com/oneblock-ai/webhook/pkg/server"
	"github.com/oneblock-ai/webhook/pkg/server/admission"
	"k8s.io/client-go/rest"

	wconfig "github.com/llmos-ai/llmos-operator/pkg/webhook/config"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/dataset"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/datasetversion"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/helmchart"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/localmodel"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/localmodelversion"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/managedaddon"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/model"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/modelservice"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/namespace"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/notebook"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/raycluster"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/upgrade"
	"github.com/llmos-ai/llmos-operator/pkg/webhook/resources/user"
)

func register(mgmt *wconfig.Management) (validators []admission.Validator, mutators []admission.Mutator) {
	validators = []admission.Validator{
		user.NewValidator(mgmt),
		raycluster.NewValidator(mgmt),
		notebook.NewValidator(mgmt),
		upgrade.NewValidator(mgmt),
		helmchart.NewValidator(),
		managedaddon.NewValidator(),
		namespace.NewValidator(),
		model.NewValidator(mgmt),
		dataset.NewValidator(mgmt),
		datasetversion.NewValidator(mgmt),
		localmodelversion.NewValidator(mgmt),
		localmodel.NewValidator(mgmt),
	}

	mutators = []admission.Mutator{
		user.NewMutator(),
		raycluster.NewMutator(mgmt),
		notebook.NewMutator(),
		modelservice.NewMutator(),
		datasetversion.NewMutator(mgmt),
		localmodelversion.NewMutator(mgmt),
	}

	return
}

func Register(ctx context.Context, restConfig *rest.Config, ws *ws.WebhookServer,
	releaseName string, threadiness int) error {
	// Separated factories are needed for the webhook register.
	// Controllers are running in active/standby mode. If the webhook register and controllers are use the same factories,
	// when the standby pod is upgraded to be active, it will be unable to add handlers and indexers to the controllers
	// because the factories are already started.
	mgmt, err := wconfig.SetupManagement(ctx, restConfig, releaseName)
	if err != nil {
		return fmt.Errorf("setup management failed: %w", err)
	}

	validators, mutators := register(mgmt)

	if err := ws.RegisterValidators(validators...); err != nil {
		return fmt.Errorf("register validators failed: %w", err)
	}

	if err := ws.RegisterMutators(mutators...); err != nil {
		return fmt.Errorf("register mutators failed: %w", err)
	}

	if err := mgmt.Start(threadiness); err != nil {
		return fmt.Errorf("start management failed: %w", err)
	}

	return nil
}
