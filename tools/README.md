# Pod Directory/File to Model Upload Script

This Python script enables uploading files from a Pod directory or single files to model storage using presigned URLs. It provides efficient, memory-friendly file transfers with progress tracking and error handling.

## Features

- **Recursive Directory Upload**: Automatically uploads all files in a directory and its subdirectories
- **Single File Upload**: Upload individual files with optional custom object names
- **Presigned URL Support**: Uses presigned URLs for direct uploads to storage backend (S3/MinIO)
- **Progress Tracking**: Visual progress bar for both single file and directory uploads with detailed status updates
- **Concurrent Uploads**: Configurable parallel uploads for improved performance (directory uploads)
- **Error Handling**: Robust error handling with detailed failure reporting
- **Content Type Detection**: Automatic content type detection based on file extensions
- **Dry Run Mode**: Preview files that would be uploaded without actually uploading
- **Flexible Configuration**: Support for custom namespaces, registries, and API servers

## Installation

### Prerequisites

- Python 3.6 or higher
- Access to the llmos-operator API server
- Appropriate permissions for the target namespace and model
- Bearer token for authentication (if required)
- SSL certificate trust or ability to skip SSL verification

### Install Dependencies

```bash
# Install required packages
pip install -r requirements.txt

# Or install manually
pip install requests urllib3 tqdm
```

## Usage

### Basic Usage

```bash
# Upload current directory to a model
python upload_to_model.py \
  --source-dir . \
  --model-name my-model \
  --namespace default \
  --bearer-token your-auth-token

# Upload specific directory
python upload_to_model.py \
  --source-dir /path/to/model/files \
  --model-name llama-7b \
  --namespace default \
  --bearer-token your-auth-token

# Upload single file
python upload_to_model.py \
  --source-file /path/to/model.bin \
  --model-name my-model \
  --namespace default \
  --bearer-token your-auth-token

# Upload single file with custom object name
python upload_to_model.py \
  --source-file /path/to/local-model.bin \
  --object-name custom-model.bin \
  --model-name my-model \
  --namespace default \
  --bearer-token your-auth-token
```

### Advanced Usage

```bash
# Upload directory with performance tuning and SSL verification
python upload_to_model.py \
  --source-dir /data/large-model \
  --model-name large-model \
  --namespace default \
  --bearer-token your-auth-token \
  --max-workers 8 \
  --timeout 600 \
  --verify-ssl

# Upload single file with custom timeout and SSL verification
python upload_to_model.py \
  --source-file /data/large-model.bin \
  --model-name large-model \
  --namespace default \
  --bearer-token your-auth-token \
  --timeout 1800 \
  --verify-ssl

# Dry run to preview directory uploads
python upload_to_model.py \
  --source-dir /data/model \
  --model-name test-model \
  --namespace default \
  --bearer-token your-auth-token \
  --dry-run

# Dry run to preview single file upload
python upload_to_model.py \
  --source-file /data/model.bin \
  --object-name custom-name.bin \
  --model-name test-model \
  --namespace default \
  --bearer-token your-auth-token \
  --dry-run
```

### Command Line Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `--source-dir` | Yes* | - | Source directory path to upload |
| `--source-file` | Yes* | - | Source file path to upload |
| `--object-name` | No | filename | Object name in storage (only used with --source-file) |
| `--model-name` | Yes | - | Model name in the API path |
| `--namespace` | Yes | - | Kubernetes namespace |
| `--bearer-token` | Yes | - | Bearer token for authentication |
| `--skip-ssl-verify` | No | `true` | Skip SSL certificate verification |
| `--verify-ssl` | No | `false` | Enable SSL certificate verification |
| `--timeout` | No | `300` | Upload timeout in seconds |
| `--max-workers` | No | `4` | Maximum concurrent uploads (directory only) |
| `--dry-run` | No | `false` | Show files that would be uploaded without uploading |

*Either `--source-dir` or `--source-file` is required (mutually exclusive)

## Examples

### Example 1: Upload Model Files

```bash
# Directory structure:
# /data/my-model/
# ├── config.json
# ├── pytorch_model.bin
# ├── tokenizer.json
# └── vocab.txt

python upload_to_model.py \
  --source-dir /data/my-model \
  --model-name my-custom-model \
  --namespace default \
  --bearer-token your-auth-token

# Files will be uploaded as:
# config.json
# pytorch_model.bin
# tokenizer.json
# vocab.txt
```

### Example 2: Large Model Upload with Optimization

```bash
# For large models, increase workers and timeout
python upload_to_model.py \
  --source-dir /data/large-model \
  --model-name llama-70b \
  --namespace default \
  --bearer-token $AUTH_TOKEN \
  --max-workers 8 \
  --timeout 1800
```

### Example 3: Single File Upload

```bash
# Upload a single model file
python upload_to_model.py \
  --source-file /data/pytorch_model.bin \
  --model-name my-model \
  --namespace default \
  --bearer-token $AUTH_TOKEN

# Upload with custom object name
python upload_to_model.py \
  --source-file /data/local_model.bin \
  --object-name production_model.bin \
  --model-name my-model \
  --namespace default \
  --bearer-token $AUTH_TOKEN
```

## Output Examples

### Successful Directory Upload

```
Collecting files from /data/my-model...
Found 4 files to upload (2.5 GB total).
Overall Progress: 100%|████████████| 4/4 [00:15<00:00,  0.27file/s]
✓ config.json: 100%|████████████| 1.0kB/1.0kB [00:01<00:00, 1.0kB/s]
✓ pytorch_model.bin: 100%|████████████| 2.5GB/2.5GB [00:12<00:00, 213MB/s]
✓ tokenizer.json: 100%|████████████| 2.0kB/2.0kB [00:01<00:00, 2.0kB/s]
✓ vocab.txt: 100%|████████████| 512B/512B [00:01<00:00, 512B/s]

Upload completed: 4/4 files successful
✅ All files uploaded successfully!

Upload completed in 15.42 seconds
```

### Successful Single File Upload

```
Uploading file: /data/pytorch_model.bin -> pytorch_model.bin
Uploading pytorch_model.bin: 100%|████████████| 1.07G/1.07G [00:08<00:00, 134MB/s]
✓ pytorch_model.bin

Upload completed: 1/1 files successful
✅ All files uploaded successfully!

Upload completed in 8.23 seconds
```

### Upload with Failures

```
Collecting files from /data/my-model...
Found 10 files to upload.
Uploading: 100%|████████████| 10/10 [00:45<00:00,  4.50s/file]

Upload completed: 8/10 files successful
Failed uploads (2):
  - large_file.bin
  - corrupted_file.txt

⚠️  2 files failed to upload
```

### Dry Run Output

**Directory Upload:**
```
DRY RUN: Collecting files...
Would upload 4 files (1.07 GB total):
  config.json -> config.json (1024 bytes)
  pytorch_model.bin -> pytorch_model.bin (1073741824 bytes)
  tokenizer.json -> tokenizer.json (2048 bytes)
  vocab.txt -> vocab.txt (512 bytes)
```

**Single File Upload:**
```
DRY RUN: Single file upload...
Would upload 1 file:
  pytorch_model.bin -> custom-model.bin (1073741824 bytes)
```

## Supported File Types

The script automatically detects content types for common file extensions:

- **Text files**: `.txt`, `.py`, `.sh` → `text/plain`, `text/x-python`, `application/x-sh`
- **Configuration files**: `.json`, `.yaml`, `.yml` → `application/json`, `application/x-yaml`
- **Model files**: `.bin`, `.safetensors`, `.pt`, `.pth`, `.ckpt` → `application/octet-stream`
- **Serialized data**: `.pkl`, `.pickle` → `application/octet-stream`
- **Default**: All other files → `application/octet-stream`

## Error Handling

The script provides comprehensive error handling:

- **Network errors**: Automatic retry for transient failures
- **Authentication errors**: Clear error messages for permission issues
- **File system errors**: Validation of source directory and file accessibility
- **API errors**: Detailed error reporting for API communication failures

## Performance Considerations

### Concurrent Uploads

- Default: 4 concurrent uploads
- Recommended for large files: 2-4 workers
- Recommended for many small files: 6-8 workers
- Adjust based on network bandwidth and server capacity

### Timeout Settings

- Default: 300 seconds (5 minutes)
- For large files (>1GB): Increase to 1800+ seconds
- For fast networks: Can reduce to 120-180 seconds

### Memory Usage

- The script streams files directly to storage
- Memory usage is minimal regardless of file size
- Each worker uses approximately 8-16MB of memory

## Troubleshooting

### Common Issues

1. **Permission Denied**
   ```
   Error: Failed to get presigned URL: 403 Forbidden
   ```
   - Check namespace and registry permissions
   - Verify API server authentication

2. **Network Timeout**
   ```
   Error uploading file.bin: HTTPSConnectionPool timeout
   ```
   - Increase `--timeout` value
   - Check network connectivity
   - Reduce `--max-workers` for unstable connections

3. **File Not Found**
   ```
   FileNotFoundError: Source directory does not exist
   ```
   - Verify source directory path
   - Check file permissions

4. **API Server Unreachable**
   ```
   Error: Failed to get presigned URL: Connection refused
   ```
   - Verify `--api-server` URL
   - Check if API server is running
   - Verify network connectivity

### Debug Mode

For detailed debugging, you can modify the script to enable verbose logging:

```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

## Integration with Kubernetes

### Running in Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: model-uploader
spec:
  containers:
  - name: uploader
    image: python:3.9-slim
    command: ["/bin/bash"]
    args:
      - -c
      - |
        pip install requests tqdm
        python /scripts/upload_to_model.py \
          --source-dir /data/model \
          --namespace default \
          --model-name my-model \
          --api-server http://llmos-operator.llmos-system.svc.cluster.local:8080
    volumeMounts:
    - name: model-data
      mountPath: /data/model
    - name: scripts
      mountPath: /scripts
  volumes:
  - name: model-data
    persistentVolumeClaim:
      claimName: model-data-pvc
  - name: scripts
    configMap:
      name: upload-scripts
  restartPolicy: Never
```

### ConfigMap for Scripts

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: upload-scripts
data:
  upload_to_model.py: |
    # Script content here...
```

## Security Considerations

- **Authentication**: Bearer token authentication is supported for secure API access
- **SSL/TLS**: HTTPS is used by default with option to skip certificate verification
- **Presigned URLs**: Have limited lifetime (default: 2 hours) with upload-only permissions
- **Data Privacy**: No sensitive data is logged or exposed
- **Secure Transfer**: File content is streamed directly without intermediate storage
- **Token Management**: Bearer tokens should be stored securely (e.g., Kubernetes secrets)

### Authentication Setup

```bash
# Set bearer token as environment variable
export BEARER_TOKEN="your-auth-token-here"

# Use in command
python upload_to_model.py \
  --source-dir /data/model \
  --model-name my-model \
  --namespace default \
  --bearer-token "$BEARER_TOKEN"
```

### SSL Configuration

```bash
# Skip SSL verification (default)
python upload_to_model.py --skip-ssl-verify ...

# Enable SSL verification
python upload_to_model.py --verify-ssl ...
```

## License

This script is part of the llmos-operator project and follows the same license terms.