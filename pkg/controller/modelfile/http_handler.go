package modelfile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	ollaApi "github.com/ollama/ollama/api"
	"github.com/ollama/ollama/format"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mgmtv1 "github.com/llmos-ai/llmos-operator/pkg/apis/common"
	mlv1 "github.com/llmos-ai/llmos-operator/pkg/apis/ml.llmos.ai/v1"
	"github.com/llmos-ai/llmos-operator/pkg/constant"
	"github.com/llmos-ai/llmos-operator/pkg/utils"
)

type modelApi string

const (
	createModel   modelApi = "api/create"
	listModel     modelApi = "api/tags"
	showModelInfo modelApi = "api/show"
	copyModel     modelApi = "api/copy"
	deleteModel   modelApi = "api/delete"
	pullModel     modelApi = "api/pull"
	pushModel     modelApi = "api/push"

	jsonType = "application/json"
)

func (h *handler) listModels() (*ollaApi.ListResponse, error) {
	url, err := getRequestURL(listModel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse list model api: %s", err.Error())
	}

	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Fatalf("failed to close response body: %s", err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	models := &ollaApi.ListResponse{}
	if err = json.Unmarshal(body, models); err != nil {
		return nil, err
	}

	return models, nil
}

func (h *handler) createModelFile(modelFile *mlv1.ModelFile) (*mlv1.ModelFile, error) {
	url, err := getRequestURL(createModel)
	if err != nil {
		return nil, fmt.Errorf("failed to parse create model api: %s", err.Error())
	}

	req := ollaApi.CreateRequest{
		Name:      modelFile.Name,
		Modelfile: modelFile.Spec.FileSpec,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Post(url, jsonType, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create modelfile: %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Fatalf("failed to close response body: %s", err.Error())
		}
	}(resp.Body)

	// Set status since model file is created
	mlv1.ModelFileCreated.SetStatusBool(modelFile, true)
	mlv1.ModelFileCreated.SetError(modelFile, "", nil)

	// Read response body
	scanner := bufio.NewScanner(resp.Body)
	var byteData []byte
	for scanner.Scan() {
		logrus.Debugf("create model file, status: %s", scanner.Text())
		byteData = scanner.Bytes()
		if scanner.Err() != nil {
			mlv1.ModelFileCreated.SetError(modelFile, "", scanner.Err())
			return h.modelFiles.UpdateStatus(modelFile)
		}
	}

	status := make(map[string]interface{})
	if err = json.Unmarshal(byteData, &status); err != nil {
		logrus.Debugf("failed to unmarshal model file status: %s", err.Error())
		return nil, err
	}

	// update complete status by response
	if status["error"] != nil {
		mlv1.ModelFileCreated.SetError(modelFile, "Error", fmt.Errorf("%s", status["error"]))
		mlv1.ModelFileCompleted.CreateUnknownIfNotExists(modelFile)
		return h.modelFiles.UpdateStatus(modelFile)
	} else if status["status"] != nil && status["status"] == "success" {
		mlv1.ModelFileCompleted.SetStatusBool(modelFile, true)
		return h.modelFiles.UpdateStatus(modelFile)
	}

	mlv1.ModelFileCompleted.SetStatusBool(modelFile, false)
	mlv1.ModelFileCompleted.SetStatus(modelFile, string(byteData))
	return h.modelFiles.UpdateStatus(modelFile)
}

func (h *handler) deleteModelFile(modelFile *mlv1.ModelFile) error {
	url, err := getRequestURL(deleteModel)
	if err != nil {
		return fmt.Errorf("failed to parse delete model api: %s", err.Error())
	}
	// make http delete request
	deleteReq := &ollaApi.DeleteRequest{Model: getModelFileName(modelFile)}

	jsonDel, err := json.Marshal(deleteReq)
	if err != nil {
		return err
	}

	req, err := retryablehttp.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonDel))
	if err != nil {
		return err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// Read Response Body
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			logrus.Warnf("model file not found: %s, skip deleting", modelFile.Name)
			return nil
		}
		return fmt.Errorf("failed to delete modelfile: %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Fatalf("failed to close response body: %s", err.Error())
		}
	}(resp.Body)

	logrus.Infof("delete model file %s successfully", modelFile.Name)
	return nil
}

// nolint:golint,unused
func (h *handler) showModelInfo(modelName string) (*ollaApi.ShowResponse, error) {
	url, err := getRequestURL(showModelInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse show model info api: %s", err.Error())
	}

	req := ollaApi.ShowRequest{
		Model: modelName,
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Post(url, jsonType, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to show model info: %s", err.Error())
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			logrus.Fatalf("failed to close response body: %s", err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model not found: %s", modelName)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get model info: %s", string(body))
	}

	showResp := &ollaApi.ShowResponse{}
	err = json.Unmarshal(body, showResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal model info: %s", err.Error())
	}
	logrus.Debugf("show model info: %v", showResp)
	return showResp, nil
}

func (h *handler) getModelFileStatus(mf *mlv1.ModelFile) error {
	models, err := h.listModels()
	if err != nil {
		return fmt.Errorf("failed to list models, error: %v", err)
	}

	for _, m := range models.Models {
		if checkModelNameExist(mf, &m) {
			setModelStatus(mf, &m)
			if _, err = h.modelFiles.UpdateStatus(mf); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func checkModelNameExist(mf *mlv1.ModelFile, model *ollaApi.ModelResponse) bool {
	mfName := getModelFileName(mf)
	if model.Model == mfName {
		return true
	}
	if utils.ReplaceAndLower(model.Model) == mfName {
		return true
	}
	if utils.ReplaceAndLower(strings.TrimSuffix(model.Model, ":latest")) == mfName {
		return true
	}
	return false
}

func (h *handler) updateModelFileStatus(modelFile *mlv1.ModelFile,
	model *ollaApi.ModelResponse) error {
	toUpdate := modelFile.DeepCopy()
	setModelStatus(toUpdate, model)
	if !reflect.DeepEqual(modelFile.Status, toUpdate.Status) {
		if _, err := h.modelFiles.UpdateStatus(toUpdate); err != nil {
			return err
		}
	}
	return nil
}

func setModelStatus(modelFile *mlv1.ModelFile, model *ollaApi.ModelResponse) {
	modelFile.Status = mlv1.ModelFileStatus{
		Model:    model.Model,
		ByteSize: format.HumanBytes(model.Size),
		Size:     model.Size,
		Digest:   model.Digest,
		ModelID:  model.Digest[:12],
		Details: mlv1.ModelDetails{
			ParentModel:       model.Details.ParentModel,
			Format:            model.Details.Format,
			Family:            model.Details.Family,
			Families:          model.Details.Families,
			ParameterSize:     model.Details.ParameterSize,
			QuantizationLevel: model.Details.QuantizationLevel,
		},
		Conditions: []mgmtv1.Condition{
			{
				Type:           mlv1.ModelFileCreated,
				Status:         metav1.ConditionTrue,
				LastUpdateTime: model.ModifiedAt.Format(constant.TimeLayout),
			},
			{
				Type:           mlv1.ModelFileCompleted,
				Status:         metav1.ConditionTrue,
				LastUpdateTime: model.ModifiedAt.Format(constant.TimeLayout),
			},
		},
		ModifiedAt: model.ModifiedAt.Format(constant.TimeLayout),
		ExpiresAt:  model.ExpiresAt.Format(constant.TimeLayout),
		IsPublic:   modelFile.Spec.IsPublic,
	}
}

func getRequestURL(api modelApi) (string, error) {
	url, err := utils.GetLocalLLMUrl()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s/%s", url.Scheme, url.Host, api), nil
}

func getModelFileName(mf *mlv1.ModelFile) string {
	if mf.Annotations != nil && mf.Annotations[constant.ModelOriginModelAnnotation] != "" {
		return mf.Annotations[constant.ModelOriginModelAnnotation]
	}
	if mf.Status.Model != "" {
		return mf.Status.Model
	}
	return mf.Name
}
