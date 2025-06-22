package datacollection

import (
	"context"
	"fmt"
	"path"
	"reflect"

	ctlbatchv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/batch/v1"

	agentv1 "github.com/llmos-ai/llmos-operator/pkg/apis/agent.llmos.ai/v1"
	ctlagentv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/agent.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/registry"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

type handler struct {
	ctx context.Context

	dataCollectionClient ctlagentv1.DataCollectionClient
	jobClient            ctlbatchv1.JobClient

	rm *registry.Manager
}

func Register(_ context.Context, mgmt *config.Management, _ config.Options) error {
	registries := mgmt.LLMFactory.Ml().V1().Registry()
	secrets := mgmt.CoreFactory.Core().V1().Secret()
	datacollections := mgmt.AgentFactory.Agent().V1().DataCollection()
	jobs := mgmt.BatchFactory.Batch().V1().Job()

	h := handler{
		ctx: mgmt.Ctx,

		dataCollectionClient: datacollections,
		jobClient:            jobs,
	}
	h.rm = registry.NewManager(secrets.Cache().Get, registries.Cache().Get)

	datacollections.OnChange(mgmt.Ctx, "datacollection.OnChange", h.OnChange)
	datacollections.OnRemove(mgmt.Ctx, "datacollection.OnRemove", h.OnRemove)

	return nil
}

// OnChange is called when an DataCollection object is created or updated.
func (h *handler) OnChange(_ string, ad *agentv1.DataCollection) (*agentv1.DataCollection, error) {
	if ad == nil || ad.DeletionTimestamp != nil {
		return ad, nil
	}

	adCopy := ad.DeepCopy()
	err := h.ensureRootPath(adCopy)
	return h.updateDataCollectionStatus(adCopy, ad, err)
}

func (h *handler) OnRemove(_ string, dc *agentv1.DataCollection) (*agentv1.DataCollection, error) {
	if dc == nil || dc.DeletionTimestamp == nil {
		return dc, nil
	}

	b, err := h.rm.NewBackendFromRegistry(h.ctx, dc.Spec.Registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend from registry %s: %w", dc.Spec.Registry, err)
	}
	if err := b.Delete(h.ctx, getRootPath(dc.Namespace, dc.Name)); err != nil {
		return nil, fmt.Errorf("failed to delete directory %s in registry %s: %w",
			getRootPath(dc.Namespace, dc.Name), dc.Spec.Registry, err)
	}

	return dc, nil
}

func (h *handler) ensureRootPath(ad *agentv1.DataCollection) error {
	// the rootPath recorded in the status is the sourceFiles directory
	rootPath := getRootPath(ad.Namespace, ad.Name)
	if agentv1.Ready.IsTrue(ad) && ad.Status.RootPath == rootPath {
		return nil
	}
	b, err := h.rm.NewBackendFromRegistry(h.ctx, ad.Spec.Registry)
	if err != nil {
		return fmt.Errorf("failed to create backend from registry %s: %w", ad.Spec.Registry, err)
	}
	if err := b.CreateDirectory(h.ctx, rootPath); err != nil {
		return fmt.Errorf("failed to create directory %s in registry %s: %w", rootPath, ad.Spec.Registry, err)
	}
	ad.Status.RootPath = rootPath
	agentv1.Ready.True(ad)
	return nil
}

func (h *handler) updateDataCollectionStatus(adCopy, ad *agentv1.DataCollection,
	err error) (*agentv1.DataCollection, error) {
	if err == nil {
		agentv1.Ready.True(adCopy)
		agentv1.Ready.Message(adCopy, "")
	} else {
		agentv1.Ready.False(adCopy)
		agentv1.Ready.Message(adCopy, err.Error())
	}

	// don't update when no change happens
	if reflect.DeepEqual(adCopy.Status, ad.Status) {
		return adCopy, err
	}

	updatedDc, updateErr := h.dataCollectionClient.UpdateStatus(adCopy)
	if updateErr != nil {
		if err != nil {
			return nil, fmt.Errorf("update application data status failed: %w (original error: %v)", updateErr, err)
		}
		return nil, fmt.Errorf("update application data status failed: %w", updateErr)
	}
	return updatedDc, err
}

func getRootPath(namespace, name string) string {
	return path.Join(agentv1.DataCollectionResourceName, namespace, name)
}
