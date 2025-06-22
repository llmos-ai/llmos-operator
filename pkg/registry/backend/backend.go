package backend

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	UID          string
	Name         string
	Path         string
	Size         int64
	IsDir        bool
	LastModified time.Time
	ContentType  string
	ETag         string
}

// Backend defines the interface for backend storage
type Backend interface {
	Uploader
	Downloader

	Deleter
	Lister

	Copy(ctx context.Context, src, dst string) error
	CreateDirectory(ctx context.Context, path string) error
	DeleteDirectory(ctx context.Context, path string) error
	GetObjectURL(objectName string) string
	GetSize(ctx context.Context, path string) (int64, error)
}

// Uploader defines the interface for uploading data
type Uploader interface {
	// Upload uploads a file from local filesystem to the backend storage
	Upload(ctx context.Context, src, dst string) error
	// UploadFromReader uploads data from an io.Reader to the backend storage
	// This is useful for uploading data directly from HTTP requests without saving to local filesystem
	// The size parameter is required for some backends to properly upload the file
	// The contentType parameter is optional and will be detected if empty
	UploadFromReader(ctx context.Context, reader io.Reader, dst string, size int64, contentType string) error
}

// Downloader defines the interface for downloading data
type Downloader interface {
	Download(ctx context.Context, src string, rw io.Writer) error
	IncrementalDownload(ctx context.Context, targetDir, outputDir string, concurrency int) error
}

// Deleter defines the interface for deleting data
type Deleter interface {
	Delete(ctx context.Context, objectName string) error
}

// Lister defines the interface for listing files in a directory
type Lister interface {
	List(ctx context.Context, prefix string, recursive, skipItself bool) ([]FileInfo, error)
}
