package utils

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	// Errors happened during request.
	Errors []string `json:"errors,omitempty"`
}

func ResponseBody(obj interface{}) []byte {
	respBody, err := json.Marshal(obj)
	if err != nil {
		return []byte(`{\"errors\":[\"Failed to parse response body\"]}`)
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

func ResponseError(rw http.ResponseWriter, statusCode int, err error) {
	ResponseErrorMsg(rw, statusCode, err.Error())
}

func ResponseErrorMsg(rw http.ResponseWriter, statusCode int, errMsg string) {
	rw.WriteHeader(statusCode)
	_, _ = rw.Write(ResponseBody(ErrorResponse{Errors: []string{errMsg}}))
}

func EncodeVars(vars map[string]string) map[string]string {
	escapedVars := make(map[string]string)
	for k, v := range vars {
		escapedVars[k] = removeNewLineInString(v)
	}
	return escapedVars
}

func removeNewLineInString(v string) string {
	escaped := strings.Replace(v, "\n", "", -1)
	escaped = strings.Replace(escaped, "\r", "", -1)
	return escaped
}
