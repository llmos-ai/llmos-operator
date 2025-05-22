package utils

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/sirupsen/logrus"
)

const errorType = "error"

// ErrorResponse describe the error response happened during request.
type ErrorResponse struct {
	Code      string `json:"code,omitempty"`
	Status    int    `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
	Type      string `json:"type,omitempty"`
	FieldName string `json:"fieldName,omitempty"`
	Error     string `json:"error,omitempty"`
}

func ResponseBody(obj interface{}) []byte {
	respBody, err := json.Marshal(obj)
	if err != nil {
		return []byte(`{\"error\":\"Failed to parse response body\"}`)
	}
	return respBody
}

func ResponseOKWithBody(rw http.ResponseWriter, obj interface{}) {
	rw.Header().Set("Content-type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, err := rw.Write(ResponseBody(obj))
	if err != nil {
		logrus.Errorf("failed to write response body: %v", err)
	}
}

func ResponseOKWithNoContent(rw http.ResponseWriter) {
	rw.Header().Set("Content-type", "application/json")
	rw.WriteHeader(http.StatusNoContent)
}

func ResponseAPIError(rw http.ResponseWriter, statusCode int, err *apierror.APIError) {
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(ResponseBody(ErrorResponse{
		Code:      err.Code.Code,
		Status:    err.Code.Status,
		Message:   err.Message,
		FieldName: err.FieldName,
		Type:      errorType,
	}))
}

func ResponseError(rw http.ResponseWriter, statusCode int, err error) {
	ResponseErrorMsg(rw, statusCode, err.Error())
}

func ResponseErrorMsg(rw http.ResponseWriter, statusCode int, errMsg string) {
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(ResponseBody(ErrorResponse{
		Code:    http.StatusText(statusCode),
		Status:  statusCode,
		Message: errMsg,
		Type:    errorType,
	}))
}

func EncodeVars(vars map[string]string) map[string]string {
	escapedVars := make(map[string]string)
	for k, v := range vars {
		escapedVars[k] = removeNewLineInString(v)
	}
	return escapedVars
}

func removeNewLineInString(v string) string {
	escaped := strings.ReplaceAll(v, "\n", "")
	escaped = strings.ReplaceAll(escaped, "\r", "")
	return escaped
}
