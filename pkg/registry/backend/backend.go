package backend

import (
	"io"
	"time"
)

// FileInfo represents metadata about a file
type FileInfo struct {
	Name         string
	Path         string
	Size         int64
	LastModified time.Time
	ContentType  string
}

// Backend defines the interface for backend storage
type Backend interface {
	Uploader
	Downloader

	Deleter
	Lister

	Copy(src, dst string) error
	CreateDirectory(path string) error
	DeleteDirectory(path string) error
	GetObjectURL(objectName string) string
}

// Uploader defines the interface for uploading data
type Uploader interface {
	Upload(objectName string, reader io.Reader, objectSize int64, contentType string) error
}

// Downloader defines the interface for downloading data
type Downloader interface {
	Download(objectName string, writer io.Writer) error
}

// Deleter defines the interface for deleting data
type Deleter interface {
	Delete(objectName string) error
}

// Lister defines the interface for listing files in a directory
type Lister interface {
	List(prefix string, recursive, skipItself bool) ([]FileInfo, error)
}
