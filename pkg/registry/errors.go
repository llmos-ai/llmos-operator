package registry

var (
	ErrCreateBackendClient = "create backend client failed: %w"
	ErrCreateDirectory     = "create directory %s failed: %w"
	ErrUploadFile          = "upload file %s failed: %w"
	ErrDownloadFile        = "download file %s failed: %w"
	ErrListFiles           = "list files with prefix %s failed: %w"
	ErrDeleteFile          = "delete file %s failed: %w"
)
