package weaviate

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"

	vd "github.com/llmos-ai/llmos-operator/pkg/vectordatabase"
	"github.com/llmos-ai/llmos-operator/pkg/vectordatabase/vectorizer"
)

// Client represents a Weaviate client with custom embedding functionality
type Client struct {
	Host           string
	Scheme         string
	Vectorizer     *vectorizer.CustomVectorizer
	weaviateClient *weaviate.Client
}

// NewClient creates a new Client instance
func NewClient(host, scheme string, vectorizer *vectorizer.CustomVectorizer) (vd.Client, error) {
	cfg := weaviate.Config{
		Host:   host,
		Scheme: scheme,
	}

	weaviateClient, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Weaviate client: %v", err)
	}

	return &Client{
		Host:           host,
		Scheme:         scheme,
		Vectorizer:     vectorizer,
		weaviateClient: weaviateClient,
	}, nil
}

// CreateCollection creates a new collection with the given name
func (c *Client) CreateCollection(ctx context.Context, collectionName string) error {
	// Define the collection class with custom embedding model
	classObj := &models.Class{
		Class:       collectionName,
		Description: "A collection for documents with custom embeddings",
		Vectorizer:  "none",
		Properties: []*models.Property{
			{
				Name:        "uid",
				DataType:    []string{"text"},
				Description: "Document UID",
			},
			{
				Name:        "document",
				DataType:    []string{"text"},
				Description: "Document name",
			},
			{
				Name:        "index",
				DataType:    []string{"int"},
				Description: "Chunk index",
			},
			{
				Name:        "keywords",
				DataType:    []string{"text"},
				Description: "Keywords",
			},
			{
				Name:        "content",
				DataType:    []string{"text"},
				Description: "Chunk content",
			},
			{
				Name:        "timestamp",
				DataType:    []string{"text"},
				Description: "Creation time",
			},
		},
	}

	// Check if class already exists
	exists, err := c.weaviateClient.Schema().ClassExistenceChecker().WithClassName(classObj.Class).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check class existence: %w", err)
	}

	if exists {
		logrus.Infof("Collection '%s' already exists", classObj.Class)
		return nil
	}

	// Create the class
	err = c.weaviateClient.Schema().ClassCreator().WithClass(classObj).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create class: %w", err)
	}

	return nil
}

// CollectionExists checks if a collection with the given name exists
func (c *Client) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	exists, err := c.weaviateClient.Schema().ClassExistenceChecker().WithClassName(collectionName).Do(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check class existence: %w", err)
	}
	return exists, nil
}

// DeleteCollection deletes a collection with the given name
func (c *Client) DeleteCollection(ctx context.Context, collectionName string) error {
	// Check if collection exists first
	exists, err := c.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		return nil
	}

	// Delete the collection
	err = c.weaviateClient.Schema().ClassDeleter().WithClassName(collectionName).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete collection '%s': %w", collectionName, err)
	}

	return nil
}

// ListObjects lists objects in the specified collection with pagination support
func (c *Client) ListObjects(ctx context.Context, collectionName, uid string,
	offset, limit int) (*vd.ObjectList, error) {
	// Get all objects from collection including vectors
	fields := []graphql.Field{
		{Name: "uid"},
		{Name: "document"},
		{Name: "index"},
		{Name: "keywords"},
		{Name: "content"},
		{Name: "timestamp"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "vector"},
		}},
	}

	// Create uid filter if provided (reused for both aggregate and get queries)
	var whereFilter *filters.WhereBuilder
	if uid != "" {
		whereFilter = filters.Where().
			WithPath([]string{"uid"}).
			WithOperator(filters.Equal).
			WithValueText(uid)
	}

	// First, get total count using Aggregate query for accurate count
	aggregateQuery := c.weaviateClient.GraphQL().Aggregate().
		WithClassName(collectionName).
		WithFields(graphql.Field{Name: "meta", Fields: []graphql.Field{{Name: "count"}}})
	if whereFilter != nil {
		aggregateQuery.WithWhere(whereFilter)
	}
	aggregateResult, err := aggregateQuery.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %v", err)
	}

	// Calculate total count from aggregate result
	totalCount := 0
	if aggregateResult.Data != nil {
		if aggregate, ok := aggregateResult.Data["Aggregate"].(map[string]interface{}); ok {
			if collection, ok := aggregate[collectionName].([]interface{}); ok {
				if len(collection) > 0 {
					if meta, ok := collection[0].(map[string]interface{}); ok {
						if metaData, ok := meta["meta"].(map[string]interface{}); ok {
							if count, ok := metaData["count"].(float64); ok {
								totalCount = int(count)
							}
						}
					}
				}
			}
		}
	}

	// Get paginated results
	query := c.weaviateClient.GraphQL().Get().
		WithClassName(collectionName).
		WithFields(fields...).
		WithOffset(offset).
		WithLimit(limit)
	// Add uid filter if provided
	if whereFilter != nil {
		query = query.WithWhere(whereFilter)
	}

	result, err := query.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get objects: %v", err)
	}

	// Parse results
	objectList := &vd.ObjectList{
		Objects: []vd.ObjectInfo{},
		Total:   totalCount,
		Offset:  offset,
		Limit:   limit,
	}

	if result.Data != nil {
		if get, ok := result.Data["Get"].(map[string]interface{}); ok {
			if collection, ok := get[collectionName].([]interface{}); ok {
				for _, item := range collection {
					if obj, ok := item.(map[string]interface{}); ok {
						objectInfo := vd.ObjectInfo{}

						if uid, ok := obj["uid"].(string); ok {
							objectInfo.UID = uid
						}
						if document, ok := obj["document"].(string); ok {
							objectInfo.Document = document
						}
						if index, ok := obj["index"].(float64); ok {
							objectInfo.Index = int(index)
						}
						if keywords, ok := obj["keywords"].(string); ok {
							objectInfo.Keywords = keywords
						}
						if content, ok := obj["content"].(string); ok {
							objectInfo.Content = content
						}
						if timestamp, ok := obj["timestamp"].(string); ok {
							objectInfo.Timestamp = timestamp
						}

						if additional, ok := obj["_additional"].(map[string]interface{}); ok {
							if id, ok := additional["id"].(string); ok {
								objectInfo.ID = id
							}
							if vector, ok := additional["vector"].([]interface{}); ok {
								vectorFloat32 := make([]float32, len(vector))
								for i, v := range vector {
									if val, ok := v.(float64); ok {
										vectorFloat32[i] = float32(val)
									}
								}
								objectInfo.Vector = vectorFloat32
							}
						}

						objectList.Objects = append(objectList.Objects, objectInfo)
					}
				}
			}
		}
	}

	return objectList, nil
}

// InsertObjects inserts multiple objects into the specified collection
func (c *Client) InsertObjects(ctx context.Context, collectionName string, documents []vd.Document) error {
	for i, doc := range documents {
		// Create a document object
		propertySchema := map[string]interface{}{
			"uid":       doc.UID,
			"document":  doc.Document,
			"index":     doc.Index,
			"keywords":  doc.Keywords,
			"content":   doc.Content,
			"timestamp": doc.Timestamp,
		}

		vector, err := c.Vectorizer.GetVector(doc.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for document %d: %v", i+1, err)
		}

		// Insert the object into the collection
		if _, err := c.weaviateClient.Data().Creator().
			WithClassName(collectionName).
			WithProperties(propertySchema).
			WithVector(vector).
			Do(ctx); err != nil {
			return fmt.Errorf("failed to insert document %d: %v", i+1, err)
		}
	}
	return nil
}

// DeleteObjects deletes all objects with the specified document name from the collection
func (c *Client) DeleteObjects(ctx context.Context, collectionName, uid string) (*vd.DeleteResult, error) {
	// First, find all objects with the specified document name
	fields := []graphql.Field{
		{Name: "uid"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
		}},
	}

	// Use GraphQL where filter to find objects with matching uid
	whereFilter := filters.Where().
		WithPath([]string{"uid"}).
		WithOperator(filters.Equal).
		WithValueText(uid)

	result, err := c.weaviateClient.GraphQL().Get().
		WithClassName(collectionName).
		WithFields(fields...).
		WithWhere(whereFilter).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to find objects with uid '%s': %v", uid, err)
	}

	deleteResult := &vd.DeleteResult{
		DeletedIDs: []string{},
		Total:      0,
	}

	// Parse the results to get object IDs
	var objectIDs []string
	if result.Data != nil {
		if get, ok := result.Data["Get"].(map[string]interface{}); ok {
			if collection, ok := get[collectionName].([]interface{}); ok {
				for _, item := range collection {
					if obj, ok := item.(map[string]interface{}); ok {
						if additional, ok := obj["_additional"].(map[string]interface{}); ok {
							if id, ok := additional["id"].(string); ok {
								objectIDs = append(objectIDs, id)
							}
						}
					}
				}
			}
		}
	}

	if len(objectIDs) == 0 {
		logrus.Infof("No objects found with uid '%s'\n", uid)
		return deleteResult, nil
	}

	logrus.Debugf("Found %d object(s) with uid '%s', deleting...\n", len(objectIDs), uid)

	// Delete each object by ID
	for i, objectID := range objectIDs {
		err := c.weaviateClient.Data().Deleter().
			WithClassName(collectionName).
			WithID(objectID).
			Do(ctx)

		if err != nil {
			return deleteResult, fmt.Errorf("failed to delete object %d with ID %s: %w", i+1, objectID, err)
		}

		deleteResult.DeletedIDs = append(deleteResult.DeletedIDs, objectID)
		logrus.Debugf("Deleted object %d with ID: %s\n", i+1, objectID)
	}

	deleteResult.Total = len(deleteResult.DeletedIDs)
	logrus.Debugf("Successfully deleted %d object(s) with uid '%s'\n", deleteResult.Total, uid)

	return deleteResult, nil
}

// Query performs a vector search on the specified collection with the given
// query string, similarity threshold and limit
func (c *Client) Search(ctx context.Context, collectionName, queryString string,
	threshold float64, limit int) (*vd.SearchResults, error) {
	// Generate embedding for the query string
	queryVector, err := c.Vectorizer.GetVector(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for query: %v", err)
	}

	// Perform nearVector search
	fields := []graphql.Field{
		{Name: "uid"},
		{Name: "document"},
		{Name: "index"},
		{Name: "keywords"},
		{Name: "content"},
		{Name: "timestamp"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "distance"},
			{Name: "vector"},
		}},
	}

	nearVectorBuilder := &graphql.NearVectorArgumentBuilder{}
	nearVectorBuilder.WithVector(queryVector)

	if limit == 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}

	if threshold > 0 {
		maxDistance := 1.0 - threshold // Convert similarity to distance
		nearVectorBuilder.WithDistance(float32(maxDistance))
	}

	nearVector := c.weaviateClient.GraphQL().Get().
		WithClassName(collectionName).
		WithFields(fields...).
		WithNearVector(nearVectorBuilder).
		WithLimit(limit)

	result, err := nearVector.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %v", err)
	}

	// Parse results
	queryResults := &vd.SearchResults{
		Results: []vd.SearchResult{},
		Total:   0,
	}

	if result.Data != nil {
		if get, ok := result.Data["Get"].(map[string]interface{}); ok {
			if collection, ok := get[collectionName].([]interface{}); ok {
				queryResults.Total = len(collection)

				for _, item := range collection {
					if obj, ok := item.(map[string]interface{}); ok {
						queryResult := vd.SearchResult{}
						if uid, ok := obj["uid"].(string); ok {
							queryResult.UID = uid
						}
						if document, ok := obj["document"].(string); ok {
							queryResult.Document = document
						}
						if index, ok := obj["index"].(float64); ok {
							queryResult.Index = int(index)
						}
						if keywords, ok := obj["keywords"].(string); ok {
							queryResult.Keywords = keywords
						}
						if content, ok := obj["content"].(string); ok {
							queryResult.Content = content
						}
						if timestamp, ok := obj["timestamp"].(string); ok {
							queryResult.Timestamp = timestamp
						}

						if additional, ok := obj["_additional"].(map[string]interface{}); ok {
							if id, ok := additional["id"].(string); ok {
								queryResult.ID = id
							}
							if distance, ok := additional["distance"].(float64); ok {
								queryResult.Similarity = 1 - distance
							}
							if vector, ok := additional["vector"].([]interface{}); ok {
								vectorFloat32 := make([]float32, len(vector))
								for i, v := range vector {
									if val, ok := v.(float64); ok {
										vectorFloat32[i] = float32(val)
									}
								}
								queryResult.Vector = vectorFloat32
							}
						}

						queryResults.Results = append(queryResults.Results, queryResult)
					}
				}
			}
		}
	}

	return queryResults, nil
}
