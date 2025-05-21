package registry

var (
	ErrCreateBackendClient = "failed to create backend client: %w"
	ErrCreateDirectory     = "failed to create directory %s: %w"
	ErrUploadFile          = "failed to upload file %s: %w"
	ErrDownloadFile        = "failed to download file %s: %w"
	ErrListFiles           = "failed to list files with prefix %s: %w"
	ErrDeleteFile          = "failed to delete file %s: %w"
	ErrGetSize             = "failed to get size of path %s: %w"
)
