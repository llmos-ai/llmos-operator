package s3

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
)

// MinioClient represents a MinIO client
type MinioClient struct {
	client *minio.Client
	bucket string
}

var _ backend.Backend = (*MinioClient)(nil)

// NewMinioClient initializes a new MinIO client
func NewMinioClient(ctx context.Context, endpoint, accessKeyID, accessKeySecret, bucket string, useSSL bool) (backend.Backend, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, accessKeySecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client to access %s: %w", endpoint, err)
	}

	// Check if the bucket exists.
	found, err := minioClient.BucketExists(ctx, bucket)
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

// Upload support both file and directory upload
func (mc *MinioClient) Upload(ctx context.Context, src, dst string) error {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat file %s failed: %v", src, err)
	}

	if fileInfo.IsDir() {
		return mc.uploadDirectory(ctx, src, dst)
	}
	return mc.uploadFile(ctx, src, path.Join(dst, fileInfo.Name()))
}

func (mc *MinioClient) uploadFile(ctx context.Context, src, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open file %s failed: %v", src, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file %s failed: %v", src, err)
	}

	contentType, err := mimetype.DetectReader(file)
	if err != nil {
		return fmt.Errorf("detect file %s mimetype failed: %v", src, err)
	}
	// reset file position because mimetype.DetectReader read the file
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("reset file position failed: %v", err)
	}

	if _, err := mc.client.PutObject(ctx, mc.bucket, dst, file,
		fileInfo.Size(), minio.PutObjectOptions{ContentType: contentType.String()}); err != nil {
		return fmt.Errorf("upload file %s failed: %v", src, err)
	}

	return nil
}

func (mc *MinioClient) uploadDirectory(ctx context.Context, src, dst string) error {
	baseDir := filepath.Base(src)
	return filepath.Walk(src, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(src, filePath)
		if err != nil {
			return fmt.Errorf("get relative path failed: %v", err)
		}

		objectName := path.Join(dst, baseDir, relPath)

		return mc.uploadFile(ctx, filePath, objectName)
	})
}

func (mc *MinioClient) Download(ctx context.Context, src string, rw http.ResponseWriter) error {
	files, err := mc.List(ctx, src, true, true)
	if err != nil {
		return fmt.Errorf("list file %s failed: %v", src, err)
	}

	logrus.Infof("%+v", files)

	if len(files) == 1 && files[0].Path == src {
		return mc.downloadFile(ctx, files[0], rw)
	}

	return mc.downloadDirectory(ctx, src, files, rw)
}

func (mc *MinioClient) download(ctx context.Context, objectName string, writer io.Writer) error {
	object, err := mc.client.GetObject(ctx, mc.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer object.Close()

	_, err = io.Copy(writer, object)
	return err
}

func (mc *MinioClient) downloadFile(ctx context.Context, file backend.FileInfo, rw http.ResponseWriter) error {
	fileName := path.Base(file.Path)
	rw.Header().Set("Content-Type", file.ContentType)
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		url.QueryEscape(fileName), url.QueryEscape(fileName)))

	buf := bufio.NewWriter(rw)
	defer buf.Flush()

	return mc.download(ctx, file.Path, buf)
}

// downloadDirectory downloads a directory as a zip file
// TODO: Verify large directory download handling. Compressing large directories on the fly might consume significant CPU/memory.
// Consider streaming a zip response or chunking downloads to handle big datasets gracefully.
func (mc *MinioClient) downloadDirectory(ctx context.Context, srcDir string, files []backend.FileInfo, rw http.ResponseWriter) error {
	zipFileName := path.Base(srcDir) + ".zip"
	rw.Header().Set("Content-Type", "application/zip")
	rw.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		url.QueryEscape(zipFileName), url.QueryEscape(zipFileName)))

	zw := zip.NewWriter(rw)
	defer zw.Close()

	for _, file := range files {
		relPath := strings.TrimPrefix(file.Path, srcDir)
		fw, err := zw.Create(relPath)
		if err != nil {
			return fmt.Errorf("create zip entry failed: %v", err)
		}

		if err := mc.download(ctx, file.Path, fw); err != nil {
			return fmt.Errorf("download file %s failed: %v", file.Name, err)
		}
	}

	return nil
}

// Delete deletes an object from the specified bucket
func (mc *MinioClient) Delete(ctx context.Context, objectName string) error {
	files, err := mc.List(ctx, objectName, true, false)
	if err != nil {
		return fmt.Errorf("list file %s failed: %w", objectName, err)
	}

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
func (mc *MinioClient) get(ctx context.Context, objectName string) (backend.FileInfo, error) {
	objInfo, err := mc.client.StatObject(ctx, mc.bucket, objectName, minio.StatObjectOptions{})
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
func (mc *MinioClient) List(ctx context.Context, prefix string, recursive, skipItself bool) ([]backend.FileInfo, error) {
	if file, err := mc.get(ctx, prefix); err == nil {
		return []backend.FileInfo{file}, nil
	}

	// Ensure the prefix ends with a slash to make it a directory
	// The minio client will list all objects with the prefix if it is a directory
	prefix = ensureTrailingSlash(prefix)
	// Create a channel to receive objects
	objectCh := mc.client.ListObjects(ctx, mc.bucket, minio.ListObjectsOptions{
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

// TODO: Consider using h.mu.Lock() and defer h.mu.Unlock() ensures that the operations like create directory,
// copy and delete directory section are thread-safe and do not interfere with concurrent operations.
func (mc *MinioClient) CreateDirectory(ctx context.Context, objectName string) error {
	// Ensure the object name ends with a slash
	objectName = ensureTrailingSlash(objectName)

	if exist, err := mc.directoryExists(ctx, objectName); err != nil {
		return fmt.Errorf("check directory existence failed: %w", err)
	} else if exist {
		return nil
	}

	// Create an empty object to represent the directory
	_, err := mc.client.PutObject(ctx, mc.bucket, objectName, strings.NewReader(""), 0, minio.PutObjectOptions{})
	return err
}

func (mc *MinioClient) DeleteDirectory(ctx context.Context, objectName string) error {
	// Ensure the object name ends with a slash
	objectName = ensureTrailingSlash(objectName)

	if err := mc.client.RemoveObject(ctx, mc.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
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
func (mc *MinioClient) Copy(ctx context.Context, sourcePath, destPath string) error {
	// List all files in source directory
	files, err := mc.List(ctx, sourcePath, true, true)
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
		_, err = mc.client.CopyObject(ctx, dst, src)
		if err != nil {
			return fmt.Errorf("copy file %s to %s failed: %w", file.Path, targetPath, err)
		}
	}

	return nil
}

// Exists checks if a file or directory exists in the bucket
func (mc *MinioClient) directoryExists(ctx context.Context, path string) (bool, error) {
	objectCh := mc.client.ListObjects(ctx, mc.bucket, minio.ListObjectsOptions{
		Prefix:    path,
		Recursive: false,
		MaxKeys:   1,
	})

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

func ensureTrailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}

	return path
}
