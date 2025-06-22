package v1

import (
	"github.com/llmos-ai/llmos-operator/pkg/utils/condition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FileInfo struct {
	UID          string      `json:"uid"`
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	Size         int64       `json:"size"`
	LastModified metav1.Time `json:"lastModified"`
	ContentType  string      `json:"contentType"`
	ETag         string      `json:"etag"`
}

var Ready condition.Cond = "ready"
