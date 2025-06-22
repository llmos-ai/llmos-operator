package helper

import (
	"fmt"
	"strings"

	vd "github.com/llmos-ai/llmos-operator/pkg/vectordatabase"
	"github.com/llmos-ai/llmos-operator/pkg/vectordatabase/vectorizer"
	"github.com/llmos-ai/llmos-operator/pkg/vectordatabase/weaviate"
)

const (
	weaviateHost       = "weaviate.llmos-agents.svc.cluster.local:80"
	httpScheme         = "http"
	modelServicePrefix = "modelservice-"
)

// webhook will prove the embedding model is valid
func newVectorizer(embeddingModel string) *vectorizer.CustomVectorizer {
	if embeddingModel == "" {
		return nil
	}
	tmp := strings.Split(embeddingModel, "/")
	if len(tmp) != 2 || tmp[0] == "" || tmp[1] == "" {
		return nil
	}
	namespace, name := tmp[0], tmp[1]
	serviceName := modelServicePrefix + name
	host := serviceName + "." + namespace + ".svc.cluster.local:8000"
	return vectorizer.NewCustomVectorizer(host, httpScheme)
}

func NewVectorDatabaseClient(embeddingModel string) (vd.Client, error) {
	vectorizer := newVectorizer(embeddingModel)
	if vectorizer == nil {
		return nil, fmt.Errorf("invalid embedding model format: %s", embeddingModel)
	}

	return weaviate.NewClient(weaviateHost, httpScheme, vectorizer)
}
