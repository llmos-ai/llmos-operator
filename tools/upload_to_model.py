#!/usr/bin/env python3
"""
Pod Directory/File to Model Upload Script

This script uploads files from a Pod directory or a single file to a specified model using presigned URLs.
It supports recursive directory upload and single file upload with progress tracking and error handling.

Usage:
    # Upload directory
    python upload_to_model.py --source-dir /path/to/source --namespace default --model-name my-model --bearer-token your-token
    
    # Upload single file
    python upload_to_model.py --source-file /path/to/file.bin --namespace default --model-name my-model --bearer-token your-token

Requirements:
    - requests
    - tqdm (for progress bar)
"""

import os
import sys
import argparse
import requests
import json
from pathlib import Path
from typing import List, Tuple, Optional
from urllib.parse import urljoin
from concurrent.futures import ThreadPoolExecutor, as_completed
import time
import threading
import urllib3

# Disable SSL warnings when skipping certificate verification
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

try:
    from tqdm import tqdm
except ImportError:
    print("Warning: tqdm not installed. Progress bar will not be available.")
    print("Install with: pip install tqdm")
    tqdm = None


class ModelUploader:
    """Handles uploading files to model storage using presigned URLs."""
    
    def __init__(self, namespace: str, model_name: str, 
                 timeout: int = 300, max_workers: int = 4, 
                 bearer_token: str = None, skip_ssl_verify: bool = True):
        self.api_server = 'https://llmos-operator.llmos-system.svc.cluster.local:8443'
        self.namespace = namespace
        self.model_name = model_name
        self.timeout = timeout
        self.max_workers = max_workers
        self.bearer_token = bearer_token
        self.skip_ssl_verify = skip_ssl_verify
        self.session = requests.Session()
        
        # Set reasonable timeouts
        self.session.timeout = (10, timeout)
        
        # Configure SSL verification
        self.session.verify = not skip_ssl_verify
        
        # Set authorization header if token is provided
        if bearer_token:
            self.session.headers.update({
                'Authorization': f'Bearer {bearer_token}'
            })
        
    def _get_presigned_url(self, object_name: str, content_type: str = 'application/octet-stream', 
                          expiry_hours: int = 2) -> str:
        """Get presigned upload URL for the specified object."""
        url = f"{self.api_server}/v1/ml.llmos.ai.models/{self.namespace}/{self.model_name}?action=generatePresignedURL"
        
        payload = {
            "objectName": object_name,
            "operation": "upload",
            "contentType": content_type,
            "expiryHours": expiry_hours
        }
        
        try:
            response = self.session.post(url, json=payload, timeout=30)
            response.raise_for_status()
            
            data = response.json()
            return data['presignedURL']
            
        except requests.exceptions.RequestException as e:
            raise Exception(f"Failed to get presigned URL for {object_name}: {e}") from e
        except KeyError as e:
            raise Exception(f"Invalid response format: missing {e}") from e
    
    def _upload_file_with_presigned_url(self, file_path: str, presigned_url: str, show_progress: bool = False) -> bool:
        """Upload a single file using presigned URL.
        
        Args:
            file_path: Local file path to upload
            presigned_url: Presigned URL for upload
            show_progress: Whether to show progress bar for this file
        """
        try:
            file_size = Path(file_path).stat().st_size
            
            with open(file_path, 'rb') as f:
                # Detect content type based on file extension
                content_type = self._get_content_type(file_path)
                
                headers = {
                    'Content-Type': content_type
                }
                
                # Create progress bar if requested and tqdm is available
                if show_progress and tqdm and file_size > 0:
                    progress_bar = tqdm(
                        total=file_size,
                        unit='B',
                        unit_scale=True,
                        desc=f"Uploading {Path(file_path).name}"
                    )
                    
                    # Create a wrapper that updates progress
                    class ProgressFileWrapper:
                        def __init__(self, file_obj, progress_bar):
                            self.file_obj = file_obj
                            self.progress_bar = progress_bar
                            
                        def read(self, size=-1):
                            data = self.file_obj.read(size)
                            if data:
                                self.progress_bar.update(len(data))
                            return data
                            
                        def __getattr__(self, name):
                            return getattr(self.file_obj, name)
                    
                    wrapped_file = ProgressFileWrapper(f, progress_bar)
                    
                    try:
                        response = requests.put(
                            presigned_url, 
                            data=wrapped_file, 
                            headers=headers,
                            timeout=(10, self.timeout)
                        )
                        response.raise_for_status()
                        return True
                    finally:
                        progress_bar.close()
                else:
                    response = requests.put(
                        presigned_url, 
                        data=f, 
                        headers=headers,
                        timeout=(10, self.timeout)
                    )
                    response.raise_for_status()
                    return True
                
        except Exception as e:
            print(f"Error uploading {file_path}: {e}")
            return False
    
    def _upload_file_with_presigned_url_tracked(self, file_path: str, presigned_url: str, progress_bar) -> bool:
        """Upload a single file using presigned URL with external progress tracking.
        
        Args:
            file_path: Local file path to upload
            presigned_url: Presigned URL for upload
            progress_bar: External tqdm progress bar to update
        """
        try:
            with open(file_path, 'rb') as f:
                # Detect content type based on file extension
                content_type = self._get_content_type(file_path)
                
                headers = {
                    'Content-Type': content_type
                }
                
                # Create a wrapper that updates the external progress bar
                class ProgressFileWrapper:
                    def __init__(self, file_obj, progress_bar):
                        self.file_obj = file_obj
                        self.progress_bar = progress_bar
                        
                    def read(self, size=-1):
                        data = self.file_obj.read(size)
                        if data and self.progress_bar:
                            self.progress_bar.update(len(data))
                        return data
                        
                    def __getattr__(self, name):
                        return getattr(self.file_obj, name)
                
                wrapped_file = ProgressFileWrapper(f, progress_bar)
                
                response = requests.put(
                    presigned_url, 
                    data=wrapped_file, 
                    headers=headers,
                    timeout=(10, self.timeout)
                )
                response.raise_for_status()
                return True
                
        except Exception as e:
            print(f"Error uploading {file_path}: {e}")
            return False
    
    def _get_content_type(self, file_path: str) -> str:
        """Determine content type based on file extension."""
        ext = Path(file_path).suffix.lower()
        
        content_types = {
            '.txt': 'text/plain',
            '.json': 'application/json',
            '.yaml': 'application/x-yaml',
            '.yml': 'application/x-yaml',
            '.py': 'text/x-python',
            '.sh': 'application/x-sh',
            '.bin': 'application/octet-stream',
            '.safetensors': 'application/octet-stream',
            '.pt': 'application/octet-stream',
            '.pth': 'application/octet-stream',
            '.ckpt': 'application/octet-stream',
            '.pkl': 'application/octet-stream',
            '.pickle': 'application/octet-stream',
        }
        
        return content_types.get(ext, 'application/octet-stream')
    
    def _collect_files(self, source_dir: str) -> List[Tuple[str, str]]:
        """Collect all files from source directory recursively.
        
        Returns:
            List of tuples (local_file_path, relative_path)
        """
        files = []
        source_path = Path(source_dir)
        
        if not source_path.exists():
            raise FileNotFoundError(f"Source directory does not exist: {source_dir}")
        
        if not source_path.is_dir():
            raise NotADirectoryError(f"Source path is not a directory: {source_dir}")
        
        for file_path in source_path.rglob('*'):
            if file_path.is_file():
                relative_path = file_path.relative_to(source_path)
                files.append((str(file_path), str(relative_path)))
        
        return files
    
    def upload_file(self, file_path: str, object_name: str = None) -> bool:
        """Upload a single file to model storage.
        
        Args:
            file_path: Local file path to upload
            object_name: Object name in storage (defaults to filename)
            
        Returns:
            True if upload successful, False otherwise
        """
        file_path_obj = Path(file_path)
        
        if not file_path_obj.exists():
            raise FileNotFoundError(f"Source file does not exist: {file_path}")
        
        if not file_path_obj.is_file():
            raise ValueError(f"Source path is not a file: {file_path}")
        
        if object_name is None:
            object_name = file_path_obj.name
        
        print(f"Uploading file: {file_path} -> {object_name}")
        
        try:
            # Get presigned URL
            content_type = self._get_content_type(file_path)
            presigned_url = self._get_presigned_url(object_name, content_type)
            
            # Upload file with progress bar
            success = self._upload_file_with_presigned_url(file_path, presigned_url, show_progress=True)
            
            if success:
                print(f"✓ {object_name}")
            else:
                print(f"✗ {object_name}")
            
            return success
            
        except Exception as e:
            print(f"✗ {object_name}: {e}")
            return False
    
    def upload_directory(self, source_dir: str) -> Tuple[int, int]:
        """Upload entire directory to model storage.
        
        Args:
            source_dir: Local directory path to upload
            
        Returns:
            Tuple of (successful_uploads, total_files)
        """
        print(f"Collecting files from {source_dir}...")
        files = self._collect_files(source_dir)
        
        if not files:
            print("No files found to upload.")
            return 0, 0
        
        # Calculate total size of all files
        total_size = 0
        file_sizes = {}
        for local_path, relative_path in files:
            file_size = Path(local_path).stat().st_size
            file_sizes[relative_path] = file_size
            total_size += file_size
        
        print(f"Found {len(files)} files to upload ({total_size:,} bytes total).")
        
        successful_uploads = 0
        failed_uploads = []
        
        # Use individual progress bars for each file if tqdm is available
        if tqdm:
            # Create a main progress bar for overall progress
            main_progress = tqdm(total=len(files), desc="Overall Progress", unit="file", position=0)
            
            # Dictionary to store individual progress bars
            progress_bars = {}
            progress_lock = threading.Lock()
            
            def upload_single_file_with_progress(file_info):
                local_path, relative_path = file_info
                object_name = relative_path
                file_size = file_sizes[relative_path]
                
                # Create individual progress bar for this file
                with progress_lock:
                    position = len(progress_bars) + 1
                    file_progress = tqdm(
                        total=file_size,
                        desc=f"{relative_path[:30]}.." if len(relative_path) > 30 else relative_path,
                        unit="B",
                        unit_scale=True,
                        position=position,
                        leave=False
                    )
                    progress_bars[relative_path] = file_progress
                
                try:
                    # Get presigned URL
                    content_type = self._get_content_type(local_path)
                    presigned_url = self._get_presigned_url(object_name, content_type)
                    
                    # Upload file with individual progress tracking
                    success = self._upload_file_with_presigned_url_tracked(
                        local_path, presigned_url, file_progress
                    )
                    
                    # Update status
                    if success:
                        file_progress.set_description(f"✓ {relative_path[:25]}.." if len(relative_path) > 25 else f"✓ {relative_path}")
                    else:
                        file_progress.set_description(f"✗ {relative_path[:25]}.." if len(relative_path) > 25 else f"✗ {relative_path}")
                    
                    # Close individual progress bar
                    file_progress.close()
                    
                    # Update main progress
                    main_progress.update(1)
                    
                    return success, relative_path, file_size
                    
                except Exception as e:
                    file_progress.set_description(f"✗ {relative_path[:20]}.. - {str(e)[:10]}")
                    file_progress.close()
                    main_progress.update(1)
                    return False, relative_path, file_size
            
            # Upload files with thread pool
            with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
                future_to_file = {executor.submit(upload_single_file_with_progress, file_info): file_info 
                                for file_info in files}
                
                for future in as_completed(future_to_file):
                    success, relative_path, file_size = future.result()
                    if success:
                        successful_uploads += 1
                    else:
                        failed_uploads.append(relative_path)
            
            main_progress.close()
            
        else:
            # Fallback without progress bars
            def upload_single_file(file_info):
                local_path, relative_path = file_info
                object_name = relative_path
                file_size = file_sizes[relative_path]
                
                try:
                    # Get presigned URL
                    content_type = self._get_content_type(local_path)
                    presigned_url = self._get_presigned_url(object_name, content_type)
                    
                    # Upload file
                    success = self._upload_file_with_presigned_url(local_path, presigned_url)
                    
                    status = "✓" if success else "✗"
                    print(f"{status} {relative_path} ({file_size:,} bytes)")
                    
                    return success, relative_path, file_size
                    
                except Exception as e:
                    print(f"✗ {relative_path}: {e}")
                    return False, relative_path, file_size
            
            # Upload files with thread pool
            with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
                future_to_file = {executor.submit(upload_single_file, file_info): file_info 
                                for file_info in files}
                
                for future in as_completed(future_to_file):
                    success, relative_path, file_size = future.result()
                    if success:
                        successful_uploads += 1
                    else:
                        failed_uploads.append(relative_path)
        
        print(f"\nUpload completed: {successful_uploads}/{len(files)} files successful")
        
        if failed_uploads:
            print(f"Failed uploads ({len(failed_uploads)}):")
            for failed_file in failed_uploads[:10]:  # Show first 10 failures
                print(f"  - {failed_file}")
            if len(failed_uploads) > 10:
                print(f"  ... and {len(failed_uploads) - 10} more")
        
        return successful_uploads, len(files)


def main():
    parser = argparse.ArgumentParser(
        description="Upload Pod directory or single file to model storage using presigned URLs",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Upload current directory to model
  python upload_to_model.py --source-dir . --namespace default --model-name my-model --bearer-token your-token
  
  # Upload single file to model
  python upload_to_model.py --source-file /path/to/model.bin --namespace default --model-name my-model --bearer-token your-token
  
  # Upload single file with custom object name
  python upload_to_model.py --source-file /path/to/model.bin --object-name custom-model.bin --namespace default --model-name my-model --bearer-token your-token
  
  # Upload with custom namespace and authentication
  python upload_to_model.py --source-dir /data/model --namespace production --model-name llama-7b --bearer-token your-token
  
  # Upload with custom settings and SSL verification
  python upload_to_model.py --source-dir /data --namespace default --model-name training-data --bearer-token your-token --max-workers 8 --timeout 600 --verify-ssl
        """
    )
    
    # Create mutually exclusive group for source input
    source_group = parser.add_mutually_exclusive_group(required=True)
    source_group.add_argument(
        '--source-dir', 
        help='Source directory path to upload'
    )
    source_group.add_argument(
        '--source-file', 
        help='Source file path to upload'
    )
    
    parser.add_argument(
        '--object-name', 
        help='Object name in storage (only used with --source-file, defaults to filename)'
    )
    
    parser.add_argument(
        '--namespace', 
        required=True,
        help='Kubernetes namespace'
    )
    
    parser.add_argument(
        '--model-name', 
        required=True,
        help='Model name in the API path'
    )
    
    parser.add_argument(
        '--bearer-token', 
        required=True,
        help='Bearer token for authentication'
    )
    
    parser.add_argument(
        '--skip-ssl-verify', 
        action='store_true',
        default=True,
        help='Skip SSL certificate verification (default: True)'
    )
    
    parser.add_argument(
        '--verify-ssl', 
        action='store_false',
        dest='skip_ssl_verify',
        help='Enable SSL certificate verification'
    )
    
    parser.add_argument(
        '--timeout', 
        type=int,
        default=300,
        help='Upload timeout in seconds (default: 300)'
    )
    
    parser.add_argument(
        '--max-workers', 
        type=int,
        default=4,
        help='Maximum concurrent uploads (default: 4)'
    )
    
    parser.add_argument(
        '--dry-run', 
        action='store_true',
        help='Show files that would be uploaded without actually uploading'
    )
    
    args = parser.parse_args()
    
    # Validate arguments
    if args.source_file and args.object_name and args.max_workers != 4:
        print("Warning: --max-workers is ignored when uploading a single file")
    
    try:
        uploader = ModelUploader(
            namespace=args.namespace,
            model_name=args.model_name,
            timeout=args.timeout,
            max_workers=args.max_workers,
            bearer_token=args.bearer_token,
            skip_ssl_verify=args.skip_ssl_verify
        )
        
        if args.dry_run:
            if args.source_file:
                print("DRY RUN: Single file upload...")
                file_path = Path(args.source_file)
                if not file_path.exists():
                    print(f"Error: File does not exist: {args.source_file}")
                    sys.exit(1)
                if not file_path.is_file():
                    print(f"Error: Path is not a file: {args.source_file}")
                    sys.exit(1)
                
                object_name = args.object_name or file_path.name
                file_size = file_path.stat().st_size
                print(f"Would upload 1 file ({file_size:,} bytes total):")
                print(f"  {file_path.name} -> {object_name} ({file_size:,} bytes)")
            else:
                print("DRY RUN: Collecting files...")
                files = uploader._collect_files(args.source_dir)
                
                # Calculate total size
                total_size = 0
                for local_path, relative_path in files:
                    total_size += os.path.getsize(local_path)
                
                print(f"Would upload {len(files)} files ({total_size:,} bytes total):")
                for local_path, relative_path in files:
                    object_name = relative_path
                    file_size = os.path.getsize(local_path)
                    print(f"  {relative_path} -> {object_name} ({file_size:,} bytes)")
            return
        
        start_time = time.time()
        
        if args.source_file:
            # Upload single file
            success = uploader.upload_file(args.source_file, args.object_name)
            successful = 1 if success else 0
            total = 1
        else:
            # Upload directory
            successful, total = uploader.upload_directory(args.source_dir)
        
        end_time = time.time()
        
        duration = end_time - start_time
        print(f"\nUpload completed in {duration:.2f} seconds")
        
        if successful == total:
            print("✅ All files uploaded successfully!")
            sys.exit(0)
        else:
            print(f"⚠️  {total - successful} files failed to upload")
            sys.exit(1)
            
    except KeyboardInterrupt:
        print("\n❌ Upload cancelled by user")
        sys.exit(130)
    except Exception as e:
        print(f"❌ Error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()