package modelfile

import (
	"context"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/llmos-ai/llmos-operator/pkg/constant"
	ctlmlv1 "github.com/llmos-ai/llmos-operator/pkg/generated/controllers/ml.llmos.ai/v1"

	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/server/config"
)

const (
	modelFilesOnChange = "modelFile.onChange"
	modelFilesOnDelete = "modelFile.onDelete"
)

type handler struct {
	modelFiles     ctlmlv1.ModelFileClient
	ModelFileCache ctlmlv1.ModelFileCache
	client         *retryablehttp.Client
}

func Register(ctx context.Context, mgmt *config.Management, _ config.Options) error {
	modelFiles := mgmt.LLMFactory.Ml().V1().ModelFile()
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	h := &handler{
		modelFiles:     modelFiles,
		ModelFileCache: modelFiles.Cache(),
		client:         retryClient,
	}

	modelFiles.OnChange(ctx, modelFilesOnChange, h.OnChange)
	modelFiles.OnRemove(ctx, modelFilesOnDelete, h.OnDelete)

	syncer := NewModelSyncer(ctx, h)
	go syncer.start()

	// sync local models once on start
	return h.syncLocalModels()
}

func (h *handler) OnChange(_ string, mf *mlv1.ModelFile) (*mlv1.ModelFile, error) {
	if mf == nil || mf.DeletionTimestamp != nil {
		return mf, nil
	}

	if mlv1.ModelFileCreated.IsTrue(mf) && mlv1.ModelFileCompleted.IsTrue(mf) &&
		mf.Status.Model != "" && mf.Status.ModelID != "" {
		return mf, nil
	}

	if mlv1.ModelFileCreated.GetStatus(mf) == "" || mlv1.ModelFileCreated.IsFalse(mf) {
		// skip creating new modelfile if it's synced from local models by syncer
		if mf.Annotations == nil || mf.Annotations[constant.ModelOriginModelAnnotation] == "" {
			toUpdate := mf.DeepCopy()
			// init modelApi file status
			_, err := h.createModelFile(toUpdate)
			if err != nil {
				mlv1.ModelFileCreated.SetError(toUpdate, "", err)
				return h.modelFiles.UpdateStatus(toUpdate)
			}
		}
	}

	if mf.Status.Model == "" || mf.Status.ModelID == "" {
		toUpdate := mf.DeepCopy()
		// get modelApi file status
		if err := h.getModelFileStatus(toUpdate); err != nil {
			mlv1.ModelFileCreated.SetError(toUpdate, "", err)
			return h.modelFiles.UpdateStatus(toUpdate)
		}
	}

	return mf, nil
}

func (h *handler) OnDelete(_ string, mf *mlv1.ModelFile) (*mlv1.ModelFile, error) {
	if mf == nil || mf.DeletionTimestamp == nil {
		return nil, nil
	}

	if mf.Annotations != nil && mf.Annotations[constant.ModelFileSkipDeleteAnnotation] == "true" {
		return nil, nil
	}

	if mlv1.ModelFileCreated.IsTrue(mf) {
		if err := h.deleteModelFile(mf); err != nil {
			return nil, err
		}
	}
	return mf, nil
}
