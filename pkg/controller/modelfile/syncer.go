package modelfile

import (
	"context"
	"fmt"
	"time"

	ollaApi "github.com/ollama/ollama/api"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1 "github.com/llmos-ai/llmos-controller/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-controller/pkg/constant"
	"github.com/llmos-ai/llmos-controller/pkg/settings"
	"github.com/llmos-ai/llmos-controller/pkg/utils"
)

const (
	syncInterval = 15 * time.Minute
)

type modelSyncer struct {
	ctx     context.Context
	handler *handler
}

func NewModelSyncer(ctx context.Context, h *handler) *modelSyncer {
	return &modelSyncer{
		ctx:     ctx,
		handler: h,
	}
}

func (s *modelSyncer) start() {
	logrus.Infoln("Starting local model syncer")

	ticker := time.NewTicker(syncInterval)
	for {
		select {
		case <-ticker.C:
			if err := s.handler.syncLocalModels(); err != nil {
				logrus.Warnf("failed syncing local model file: %v", err)
			}
		case <-s.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (h *handler) syncLocalModels() error {
	// skip if local LLM server url is not set
	if len(settings.LocalLLMServerURL.Get()) == 0 {
		return nil
	}

	models, err := h.listModels()
	if err != nil {
		return fmt.Errorf("failed to list models, error: %v", err)
	}

	modelFiles, err := h.modelFiles.List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, m := range models.Models {
		found := false
		for _, mf := range modelFiles.Items {
			if mf.Status.Model == m.Model {
				found = true
				logrus.Debugf("model %s found in modelFile CR: %s", m.Model, mf.Name)
				if err = h.updateModelFileStatus(&mf, &m); err != nil {
					return err
				}
				break
			}
		}
		if !found {
			logrus.Infof("model %s not found in modelFile CR, creating new modelFile CR", m.Model)
			newMF := newExistingModelFile(&m)
			newMF, err = h.modelFiles.Create(newMF)
			if err != nil {
				return err
			}

			if err = h.updateModelFileStatus(newMF, &m); err != nil {
				return err
			}
		}
	}
	return nil
}

func newExistingModelFile(model *ollaApi.ModelResponse) *mlv1.ModelFile {
	return &mlv1.ModelFile{
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.ReplaceAndLower(model.Model),
			Annotations: map[string]string{
				constant.ModelOriginModelAnnotation: model.Model,
			},
		},
		Spec: mlv1.ModelFileSpec{
			FileSpec: fmt.Sprintf("FROM %s", model.Model),
		},
	}
}
