package s3

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
)

// MinioClient represents a MinIO client
type MinioClient struct {
	client *minio.Client
	bucket string
}

var _ backend.Backend = (*MinioClient)(nil)

// NewMinioClient initializes a new MinIO client
func NewMinioClient(endpoint, accessKeyID, accessKeySecret, bucket string, useSSL bool) (backend.Backend, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, accessKeySecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client to access %s: %w", endpoint, err)
	}

	// Check if the bucket exists.
	found, err := minioClient.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket %s exists: %w", bucket, err)
	}

	if !found {
		return nil, fmt.Errorf("bucket %s does not exist", bucket)
	}

	return &MinioClient{
		client: minioClient,
		bucket: bucket,
	}, nil
}

// Upload uploads an object to the specified bucket
func (mc *MinioClient) Upload(objectName string, reader io.Reader, objectSize int64, contentType string) error {
	_, err := mc.client.PutObject(context.Background(), mc.bucket, objectName, reader, objectSize, minio.PutObjectOptions{ContentType: contentType})
	return err
}

// Download downloads an object from the specified bucket
func (mc *MinioClient) Download(objectName string, writer io.Writer) error {
	object, err := mc.client.GetObject(context.Background(), mc.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer object.Close()

	_, err = io.Copy(writer, object)
	return err
}

// Delete deletes an object from the specified bucket
func (mc *MinioClient) Delete(objectName string) error {
	files, err := mc.List(objectName, true, false)
	if err != nil {
		return fmt.Errorf("list file %s failed: %w", objectName, err)
	}

	ctx := context.Background()
	for _, file := range files {
		if err := mc.client.RemoveObject(ctx, mc.bucket, file.Path, minio.RemoveObjectOptions{}); err != nil {
			if !strings.Contains(err.Error(), "NoSuchKey") {
				return fmt.Errorf("remove file %s failed: %w", file.Path, err)
			}
		}
	}

	return nil
}

// Get retrieves metadata about an object. The object can not be a directory.
func (mc *MinioClient) get(objectName string) (backend.FileInfo, error) {
	objInfo, err := mc.client.StatObject(context.Background(), mc.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return backend.FileInfo{}, fmt.Errorf("get object info failed: %w", err)
	}

	return backend.FileInfo{
		Name:         path.Base(objInfo.Key),
		Path:         objInfo.Key,
		Size:         objInfo.Size,
		LastModified: objInfo.LastModified,
		ContentType:  objInfo.ContentType,
	}, nil
}

// List lists objects in the specified directory (prefix)
func (mc *MinioClient) List(prefix string, recursive, skipItself bool) ([]backend.FileInfo, error) {
	if file, err := mc.get(prefix); err == nil {
		return []backend.FileInfo{file}, nil
	}

	// Ensure the prefix ends with a slash to make it a directory
	// The minio client will list all objects with the prefix if it is a directory
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	// Create a channel to receive objects
	objectCh := mc.client.ListObjects(context.Background(), mc.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	})

	var fileInfos []backend.FileInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		// Skip the directory itself
		if skipItself && object.Key == prefix {
			continue
		}

		fileInfos = append(fileInfos, backend.FileInfo{
			Name:         path.Base(object.Key),
			Path:         object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ContentType:  object.ContentType,
		})
	}

	return fileInfos, nil
}

func (mc *MinioClient) CreateDirectory(objectName string) error {
	// Ensure the object name ends with a slash
	if !strings.HasSuffix(objectName, "/") {
		objectName = objectName + "/"
	}

	if exist, err := mc.directoryExists(objectName); err != nil {
		return fmt.Errorf("check directory existence failed: %w", err)
	} else if exist {
		return nil
	}

	// Create an empty object to represent the directory
	_, err := mc.client.PutObject(context.Background(), mc.bucket, objectName, strings.NewReader(""), 0, minio.PutObjectOptions{})
	return err
}

func (mc *MinioClient) DeleteDirectory(objectName string) error {
	// Ensure the object name ends with a slash
	if !strings.HasSuffix(objectName, "/") {
		objectName = objectName + "/"
	}

	if err := mc.client.RemoveObject(context.Background(), mc.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		// ignore NoSuchKey error, because it may be not directory but file
		if !strings.Contains(err.Error(), "NoSuchKey") {
			return fmt.Errorf("remove directory %s failed: %w", objectName, err)
		}
	}

	return nil
}

func (mc *MinioClient) GetObjectURL(objectKey string) string {
	endpoint := mc.client.EndpointURL()

	// virtualHostStyle: <scheme>://<bucket>.<endpoint>/<objectKey>
	return fmt.Sprintf("%s://%s.%s/%s", endpoint.Scheme, mc.bucket, endpoint.Host, objectKey)
}

// Copy copies a file or directory from source to destination
func (mc *MinioClient) Copy(sourcePath, destPath string) error {
	// List all files in source directory
	files, err := mc.List(sourcePath, true, true)
	if err != nil {
		return fmt.Errorf("list source directory failed: %w", err)
	}

	// Copy each file
	for _, file := range files {
		// Calculate target path
		relPath := strings.TrimPrefix(file.Path, sourcePath)
		targetPath := path.Join(destPath, relPath)

		// Copy file
		dst := minio.CopyDestOptions{
			Bucket: mc.bucket,
			Object: targetPath,
		}
		src := minio.CopySrcOptions{
			Bucket: mc.bucket,
			Object: file.Path,
		}
		_, err = mc.client.CopyObject(context.Background(), dst, src)
		if err != nil {
			return fmt.Errorf("copy file %s to %s failed: %w", file.Path, targetPath, err)
		}
	}

	return nil
}

// Exists checks if a file or directory exists in the bucket
func (mc *MinioClient) directoryExists(path string) (bool, error) {
	// 列出目录下的对象
	objectCh := mc.client.ListObjects(context.Background(), mc.bucket, minio.ListObjectsOptions{
		Prefix:    path,
		Recursive: false,
		MaxKeys:   1,
	})

	// 检查是否有对象
	for obj := range objectCh {
		if obj.Err != nil {
			return false, obj.Err
		}
		// Only consider it a match if it's the directory itself or a file within it
		if strings.HasPrefix(obj.Key, path) {
			return true, nil
		}
	}

	return false, nil
}
