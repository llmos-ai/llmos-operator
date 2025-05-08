package backend

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a file
type FileInfo struct {
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
	Upload(ctx context.Context, src, dst string) error
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
