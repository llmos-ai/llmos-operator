package s3

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"

	"github.com/llmos-ai/llmos-operator/pkg/registry/backend"
)

// MinioClient represents a MinIO client
type MinioClient struct {
	client   *minio.Client
	endpoint string
	bucket   string
}

var _ backend.Backend = (*MinioClient)(nil)

// NewMinioClient initializes a new MinIO client
func NewMinioClient(ctx context.Context, endpoint, accessKeyID, accessKeySecret,
	bucket string, useSSL bool) (backend.Backend, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}
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
		client:   minioClient,
		endpoint: endpoint,
		bucket:   bucket,
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

// UploadFromReader uploads data from an io.Reader to the backend storage
// This is useful for uploading data directly from HTTP requests without saving to local filesystem
func (mc *MinioClient) UploadFromReader(ctx context.Context, reader io.Reader, dst string,
	size int64, contentType string) error {
	// If content type is not provided, try to detect it
	if contentType == "" {
		// Create a buffer to read the first 512 bytes for content type detection
		bufReader := bufio.NewReader(reader)
		header, err := bufReader.Peek(512)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read header for content type detection: %w", err)
		}

		// Detect content type
		detectedType := mimetype.Detect(header)
		contentType = detectedType.String()

		// Use the buffered reader for the upload
		reader = bufReader
	}

	// Upload the file to MinIO
	options := minio.PutObjectOptions{ContentType: contentType}

	// If size is not known (size <= 0), use streaming upload
	if size <= 0 {
		// For unknown size, we need to use PutObject with a reader that doesn't need size
		// This might be less efficient for large files
		info, err := mc.client.PutObject(ctx, mc.bucket, dst, reader, -1, options)
		if err != nil {
			return fmt.Errorf("upload from reader failed: %w", err)
		}
		logrus.Debugf("Uploaded object %s of size %d with etag %s", info.Key, info.Size, info.ETag)
	} else {
		// For known size, use the more efficient method
		info, err := mc.client.PutObject(ctx, mc.bucket, dst, reader, size, options)
		if err != nil {
			return fmt.Errorf("upload from reader failed: %w", err)
		}
		logrus.Debugf("Uploaded object %s of size %d with etag %s", info.Key, info.Size, info.ETag)
	}

	return nil
}

func (mc *MinioClient) uploadFile(ctx context.Context, src, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open file %s failed: %v", src, err)
	}
	defer file.Close() //nolint:errcheck

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

func (mc *MinioClient) Download(ctx context.Context, src string, rw io.Writer) error {
	files, err := mc.List(ctx, src, true, true)
	if err != nil {
		return fmt.Errorf("list file %s failed: %v", src, err)
	}

	if len(files) == 0 {
		return fmt.Errorf("src %s not found or it's directory containing no objects", src)
	}

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
	defer object.Close() //nolint:errcheck

	_, err = io.Copy(writer, object)
	return err
}

func (mc *MinioClient) downloadFile(ctx context.Context, file backend.FileInfo, rw io.Writer) error {
	buf := bufio.NewWriter(rw)
	defer buf.Flush() //nolint:errcheck

	return mc.download(ctx, file.Path, buf)
}

// downloadDirectory downloads a directory as a zip file
// TODO: Verify large directory download handling. compressing large directories on the fly might consume
// significant CPU/memory. Consider streaming a zip response or chunking downloads to handle big datasets gracefully
func (mc *MinioClient) downloadDirectory(ctx context.Context, srcDir string,
	files []backend.FileInfo, rw io.Writer) error {
	zw := zip.NewWriter(rw)
	defer zw.Close() //nolint:errcheck

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
		IsDir:        false,
		LastModified: objInfo.LastModified,
		ContentType:  objInfo.ContentType,
		ETag:         objInfo.ETag,
	}, nil
}

// List lists objects in the specified directory (prefix)
func (mc *MinioClient) List(ctx context.Context, prefix string, recursive,
	skipItself bool) ([]backend.FileInfo, error) {
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

	fileInfos := make([]backend.FileInfo, 0, len(objectCh))
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		// Skip the directory itself
		if skipItself && object.Key == prefix {
			continue
		}

		fileInfos = append(fileInfos, backend.FileInfo{
			UID:          fileUid(mc.endpoint, object.Key, object.ETag),
			Name:         path.Base(object.Key),
			Path:         object.Key,
			Size:         object.Size,
			IsDir:        object.Size == 0,
			LastModified: object.LastModified,
			ContentType:  object.ContentType,
			ETag:         object.ETag,
		})
	}

	return fileInfos, nil
}

func fileUid(endpoint, path, etag string) string {
	data := fmt.Appendf([]byte{}, "%s~%s~%s", endpoint, path, etag)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (mc *MinioClient) GetSize(ctx context.Context, prefix string) (int64, error) {
	if file, err := mc.get(ctx, prefix); err == nil {
		return file.Size, nil
	}

	var size int64
	prefix = ensureTrailingSlash(prefix)
	// Create a channel to receive objects
	objectCh := mc.client.ListObjects(ctx, mc.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return 0, object.Err
		}

		size += object.Size
	}

	return size, nil
}

// CreateDirectory TODO: Consider using h.mu.Lock() and defer h.mu.Unlock() ensures that the operations like create
// directory, copy and delete directory section are thread-safe and do not interfere with concurrent operations.
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

// IncrementalDownload downloads files from S3 incrementally to a local directory.
// It compares local metadata with remote files and only downloads changed or new files.
// Because the files in target directory may changed during the download, retry to compare
// metadata and download until the metadata is consistent with remote files.
// concurrency controls the number of concurrent downloads.
func (mc *MinioClient) IncrementalDownload(ctx context.Context, targetDir, outputDir string, concurrency int) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory %s failed: %w", outputDir, err)
	}

	metadataPath := filepath.Join(outputDir, ".metadata.json")
	maxRetries := 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Load local metadata if exists
		localMetadata := make(map[string]backend.FileInfo)
		if _, err := os.Stat(metadataPath); err == nil {
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				return fmt.Errorf("read metadata file failed: %w", err)
			}

			if err := json.Unmarshal(data, &localMetadata); err != nil {
				return fmt.Errorf("unmarshal metadata failed: %w", err)
			}
		}

		// Get remote files
		remoteFiles, err := mc.List(ctx, targetDir, true, false)
		if err != nil {
			return fmt.Errorf("list remote files in %s failed: %w", targetDir, err)
		}

		// Build remote file map for comparison
		remoteFileMap := make(map[string]backend.FileInfo)
		for _, file := range remoteFiles {
			if !file.IsDir {
				remoteFileMap[file.Path] = file
			}
		}

		// Determine files to download (new or changed)
		var filesToDownload []backend.FileInfo
		for _, file := range remoteFiles {
			// Skip directories
			if file.IsDir {
				continue
			}

			// Check if file exists in local metadata with same size and checksum
			if localFile, exists := localMetadata[file.Path]; exists {
				if localFile.Size == file.Size && localFile.ETag == file.ETag {
					continue // File unchanged, skip download
				}
			}

			filesToDownload = append(filesToDownload, file)
		}

		// Determine files to delete (exist locally but not in remote)
		var filesToDelete []string
		for path := range localMetadata {
			if _, exists := remoteFileMap[path]; !exists {
				filesToDelete = append(filesToDelete, path)
			}
		}

		logrus.Debugf("attempt: %d", attempt)
		logrus.Debugf("filesToDownload: %+v", filesToDownload)
		logrus.Debugf("filesToDelete: %+v", filesToDelete)

		// If no changes detected, we're done
		if len(filesToDownload) == 0 && len(filesToDelete) == 0 {
			// If this is not the first attempt, it means we've reached consistency
			if attempt > 0 {
				logrus.Debug("metadata is consistent with remote files, download finished")
				return nil
			}
		}

		// Download changed/new files
		if len(filesToDownload) > 0 {
			if err = mc.downloadFilesWithConcurrency(ctx, filesToDownload, targetDir, outputDir, concurrency); err != nil {
				return err
			}

			// Update local metadata for downloaded files
			for _, file := range filesToDownload {
				localMetadata[file.Path] = file
			}
		}

		// Delete files that no longer exist in remote
		for _, path := range filesToDelete {
			relPath := strings.TrimPrefix(path, targetDir)
			localPath := filepath.Join(outputDir, relPath)

			if err = os.Remove(localPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove deleted file %s failed: %w", localPath, err)
			}

			// Remove from metadata
			delete(localMetadata, path)
		}

		// Save updated metadata with pretty formatting
		updatedMetadata, err := json.MarshalIndent(localMetadata, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal metadata failed: %w", err)
		}

		logrus.Debugf("write metadata to %s", metadataPath)
		if err := os.WriteFile(metadataPath, updatedMetadata, 0644); err != nil {
			return fmt.Errorf("write metadata file failed: %w", err)
		}
	}

	// If we've reached here, we've hit the maximum number of retries without consistency
	return fmt.Errorf("failed to achieve consistent state after %d attempts", maxRetries)
}

// downloadFilesWithConcurrency downloads multiple files concurrently
func (mc *MinioClient) downloadFilesWithConcurrency(ctx context.Context, files []backend.FileInfo,
	targetDir, outputDir string, concurrency int) error {
	// Create worker pool for concurrent downloads
	var wg sync.WaitGroup
	errorCh := make(chan error, len(files))
	fileCh := make(chan backend.FileInfo, len(files))

	// Limit concurrency to the number of files if fewer
	if concurrency <= 1 {
		concurrency = 1
	}
	if concurrency > len(files) {
		concurrency = len(files)
	}

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				if err := mc.downloadSingleFile(ctx, file, targetDir, outputDir); err != nil {
					errorCh <- fmt.Errorf("download file %s failed: %w", file.Path, err)
					return
				}
			}
		}()
	}

	// Send files to workers
	for _, file := range files {
		fileCh <- file
	}
	close(fileCh)

	// Wait for all downloads to complete
	wg.Wait()
	close(errorCh)

	// Check for errors
	errs := make([]error, 0, len(errorCh))
	for err := range errorCh {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("failed to download, multiple errors: %v", errs)
}

// downloadSingleFile downloads a single file to the output directory
func (mc *MinioClient) downloadSingleFile(
	ctx context.Context,
	file backend.FileInfo,
	targetDir, outputDir string,
) error {
	// Create relative path and ensure parent directories exist
	relPath := strings.TrimPrefix(file.Path, targetDir)
	localPath := filepath.Join(outputDir, relPath)

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("create directory for %s failed: %w", localPath, err)
	}

	// Download to temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(localPath), "download-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tempPath := tempFile.Name()

	// Close the file handle as download will reopen it
	tempFile.Close() //nolint:errcheck

	// Clean up temp file when done
	defer os.Remove(tempPath) //nolint:errcheck

	// Download the file
	tempWriter, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create temp file for writing failed: %w", err)
	}

	if err = mc.downloadFile(ctx, file, tempWriter); err != nil {
		tempWriter.Close() //nolint:errcheck
		return err
	}
	tempWriter.Close() //nolint:errcheck

	// Verify downloaded file
	downloadedFile, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("open downloaded file failed: %w", err)
	}
	defer downloadedFile.Close() //nolint:errcheck

	// Verify file integrity using ETag
	// MinIO uses MD5 for ETag unless it's a multipart upload
	// For multipart uploads, ETag format is: "MD5-MULTIPART_COUNT"
	if file.ETag != "" {
		// Remove quotes if present in ETag
		etag := strings.Trim(file.ETag, "\"")

		// Check if it's a multipart upload (contains a hyphen)
		if !strings.Contains(etag, "-") {
			// Calculate MD5 hash of the downloaded file
			hash := md5.New()
			if _, err := io.Copy(hash, downloadedFile); err != nil {
				return fmt.Errorf("calculate MD5 hash failed: %w", err)
			}

			// Compare calculated MD5 with ETag
			calculatedMD5 := fmt.Sprintf("%x", hash.Sum(nil))
			if !strings.EqualFold(calculatedMD5, etag) {
				return fmt.Errorf("ETag verification failed: expected %s, got %s", etag, calculatedMD5)
			}

			// Reset file position for any subsequent operations
			if _, err := downloadedFile.Seek(0, 0); err != nil {
				return fmt.Errorf("reset file position failed: %w", err)
			}
		}
		// For multipart uploads, we skip verification as it requires knowledge of part sizes
		// which we don't have access to here
	}

	// Move temp file to final location
	if err := os.Rename(tempPath, localPath); err != nil {
		return fmt.Errorf("rename temp file failed: %w", err)
	}

	return nil
}
