# Presigned URL Upload Example

This document demonstrates how to use presigned URLs to implement direct upload of large files, avoiding file transfer through the server.

## Workflow

1. **Client requests presigned URL**: Request a presigned upload URL from the server
2. **Server generates presigned URL**: Server calls S3 backend to generate presigned URL and returns it to the client
3. **Client uploads directly**: Client uses the presigned URL to upload files directly to S3 storage

## API Usage Examples

### 1. Get Presigned Upload URL

```bash
curl -X POST "http://localhost:8080/v1/namespaces/default/registries/my-registry/generatePresignedURL" \
  -H "Content-Type: application/json" \
  -d '{
    "objectName": "models/large-model.bin",
    "operation": "upload",
    "contentType": "application/octet-stream",
    "expiryHours": 2
  }'
```

**Response Example:**
```json
{
  "presignedURL": "https://minio.example.com/bucket/models/large-model.bin?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...",
  "expiresAt": "2024-01-15T10:30:00Z",
  "operation": "upload"
}
```

### 2. Upload File Using Presigned URL

```bash
curl -X PUT "${presignedURL}" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @large-model.bin
```

### 3. Get Presigned Download URL

```bash
curl -X POST "http://localhost:8080/v1/namespaces/default/registries/my-registry/generatePresignedURL" \
  -H "Content-Type: application/json" \
  -d '{
    "objectName": "models/large-model.bin",
    "operation": "download",
    "expiryHours": 1
  }'
```

## Frontend JavaScript Examples

### Get Presigned URL and Upload File

```javascript
async function uploadFileWithPresignedURL(file, objectName) {
  try {
    // 1. Get presigned URL
    const response = await fetch('/v1/namespaces/default/registries/my-registry/generatePresignedURL', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        objectName: objectName,
        operation: 'upload',
        contentType: file.type || 'application/octet-stream',
        expiryHours: 2
      })
    });
    
    const { presignedURL } = await response.json();
    
    // 2. t to S3
    const uploadResponse = await fetch(presignedURL, {
      method: 'PUT',
      headers: {
        'Content-Type': file.type || 'application/octet-stream',
      },
      body: file
    });
    
    if (uploadResponse.ok) {
      console.log('File uploaded successfully!');
    } else {
      throw new Error('Upload failed');
    }
    
  } catch (error) {
    console.error('Upload error:', error);
  }
}

// Usage example
const fileInput = document.getElementById('fileInput');
fileInput.addEventListener('change', (event) => {
  const file = event.target.files[0];
  if (file) {
    uploadFileWithPresignedURL(file, `uploads/${file.name}`);
  }
});
```

### Upload with Progress Bar

```javascript
async function uploadWithProgress(file, objectName, onProgress) {
  try {
    // Get presigned URL
    const response = await fetch('/v1/namespaces/default/registries/my-registry/generatePresignedURL', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        objectName: objectName,
        operation: 'upload',
        contentType: file.type || 'application/octet-stream',
        expiryHours: 2
      })
    });
    
    const { presignedURL } = await response.json();
    
    // Use XMLHttpRequest to support progress monitoring
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable) {
          const percentComplete = (event.loaded / event.total) * 100;
          onProgress(percentComplete);
        }
      });
      
      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          resolve();
        } else {
          reject(new Error(`Upload failed: ${xhr.status}`));
        }
      });
      
      xhr.addEventListener('error', () => {
        reject(new Error('Network error'));
      });
      
      xhr.open('PUT', presignedURL);
      xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream');
      xhr.send(file);
    });
    
  } catch (error) {
    console.error('Upload error:', error);
    throw error;
  }
}
```

## Advantages

1. **Performance Optimization**: Files are uploaded directly to S3 without going through the application server, reducing server load
2. **Memory Friendly**: Browser doesn't need to load the entire file into memory, supports streaming upload
3. **Scalability**: Server only needs to generate URLs without handling file transfer, can support more concurrent users
4. **Security**: Presigned URLs have time limits and can only be used for specified operations
5. **User Experience**: Supports upload progress monitoring, more friendly for large file uploads

## Considerations

1. **URL Expiry**: Presigned URLs have time limits, default 1 hour, recommended maximum not exceeding 24 hours
2. **CORS Configuration**: Ensure S3 bucket is configured with correct CORS policy
3. **File Size Limits**: Although server memory limits are avoided, S3's single object size limits still need to be considered
4. **Error Handling**: Need to handle network interruptions, URL expiry and other exceptional situations
5. **Security Considerations**: objectName should be validated to avoid path traversal attacks
