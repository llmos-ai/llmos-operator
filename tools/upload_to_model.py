#!/usr/bin/env python3
"""Pod Directory/File to Model Upload Script

This script uploads files from a Pod directory or a single file to a specified model using presigned URLs.
It supports recursive directory upload and single file upload with progress tracking and comprehensive error handling.

Usage:
    # Upload directory
    python upload_to_model.py --source-dir /path/to/source --namespace default --model-name my-model --bearer-token your-token
    
    # Upload single file
    python upload_to_model.py --source-file /path/to/file.bin --namespace default --model-name my-model --bearer-token your-token
"""

import os
import sys
import argparse
import requests
import json
import logging
import time
import threading
import traceback
from pathlib import Path
from typing import List, Tuple
from concurrent.futures import ThreadPoolExecutor, as_completed
import urllib3

# Disable SSL warnings when skipping certificate verification
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

try:
    from tqdm import tqdm
except ImportError:
    tqdm = None


class ModelUploader:
    """Handles uploading files to model storage using presigned URLs."""

    class _ProgressFileWrapper:
        """Wrapper for file objects to update a tqdm progress bar."""
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

    def __init__(self, namespace: str, model_name: str,
                 timeout: int = 300, max_workers: int = 4,
                 bearer_token: str = None, skip_ssl_verify: bool = True,
                 debug: bool = False, api_server: str = None):
        self.api_server = api_server or 'https://llmos-operator.llmos-system.svc.cluster.local:8443'
        self.namespace = namespace
        self.model_name = model_name
        self.timeout = timeout
        self.max_workers = max_workers
        self.debug = debug
        
        self.logger = logging.getLogger(__name__)
        if debug:
            logging.basicConfig(level=logging.DEBUG, format='%(asctime)s - %(levelname)s - %(message)s')
        
        self.session = requests.Session()
        self.session.verify = not skip_ssl_verify
        if bearer_token:
            self.session.headers.update({'Authorization': f'Bearer {bearer_token}'})

        if debug:
            self.logger.debug("ModelUploader initialized with API Server: %s", self.api_server)

    def _get_presigned_url(self, object_name: str, content_type: str) -> str:
        """Get a presigned upload URL for the specified object."""
        url = f"{self.api_server}/v1/ml.llmos.ai.models/{self.namespace}/{self.model_name}?action=generatePresignedURL"
        payload = {"objectName": object_name, "operation": "upload", "contentType": content_type}
        
        self.logger.debug("Requesting presigned URL for %s", object_name)
        
        try:
            response = self.session.post(url, json=payload, timeout=30)
            response.raise_for_status()
            data = response.json()
            self.logger.debug("Successfully got presigned URL.")
            return data['presignedURL']
        except requests.exceptions.RequestException as e:
            error_msg = f"Failed to get presigned URL for {object_name}: {e}"
            self.logger.error(error_msg, exc_info=self.debug)
            raise Exception(error_msg) from e
        except (KeyError, json.JSONDecodeError) as e:
            error_msg = f"Invalid response from server for {object_name}: {e}"
            self.logger.error(error_msg, exc_info=self.debug)
            raise Exception(error_msg) from e

    def _upload_file_with_presigned_url(self, file_path: str, presigned_url: str, progress_bar=None) -> bool:
        """Upload a single file using a presigned URL with optional progress tracking."""
        self.logger.debug("Uploading file: %s", file_path)
        try:
            with open(file_path, 'rb') as f:
                content_type = self._get_content_type(file_path)
                headers = {'Content-Type': content_type}
                
                data_to_upload = self._ProgressFileWrapper(f, progress_bar) if progress_bar else f
                
                response = requests.put(presigned_url, data=data_to_upload, headers=headers, timeout=(10, self.timeout))
                response.raise_for_status()
                self.logger.debug("Upload successful for %s", file_path)
                return True
        except (FileNotFoundError, PermissionError) as e:
            print(f"Error accessing file {file_path}: {e}")
            self.logger.error("File access error", exc_info=self.debug)
            return False
        except requests.exceptions.RequestException as e:
            print(f"Error uploading {file_path}: {e}")
            self.logger.error("Upload failed for %s", file_path, exc_info=self.debug)
            return False

    @staticmethod
    def _get_content_type(file_path: str) -> str:
        """Determine content type based on file extension."""
        ext = Path(file_path).suffix.lower()
        return {
            '.txt': 'text/plain',
            '.json': 'application/json',
            '.yaml': 'application/x-yaml',
            '.yml': 'application/x-yaml',
        }.get(ext, 'application/octet-stream')

    def _collect_files(self, source_dir: str) -> List[Tuple[str, str]]:
        """Collect all files from a source directory recursively."""
        source_path = Path(source_dir)
        if not source_path.is_dir():
            raise NotADirectoryError(f"Source path is not a directory: {source_dir}")
        
        files = []
        for file_path in source_path.rglob('*'):
            if file_path.is_file():
                relative_path = file_path.relative_to(source_path)
                files.append((str(file_path), str(relative_path)))
        return files

    def upload_file(self, file_path: str, object_name: str = None) -> bool:
        """Upload a single file to model storage."""
        file_path_obj = Path(file_path)
        if not file_path_obj.is_file():
            raise ValueError(f"Source path is not a file: {file_path}")
        
        object_name = object_name or file_path_obj.name
        print(f"Uploading file: {file_path} -> {object_name}")

        try:
            presigned_url = self._get_presigned_url(object_name, self._get_content_type(file_path))
            
            progress_bar = None
            if tqdm:
                file_size = file_path_obj.stat().st_size
                progress_bar = tqdm(total=file_size, unit='B', unit_scale=True, desc=f"Uploading {file_path_obj.name}")

            with progress_bar or open(os.devnull, 'w'):
                success = self._upload_file_with_presigned_url(file_path, presigned_url, progress_bar)
            
            print(f"✓ {object_name}" if success else f"✗ {object_name}")
            return success
        except Exception as e:
            print(f"✗ {object_name}: {e}")
            return False

    def upload_directory(self, source_dir: str) -> Tuple[int, int]:
        """Upload an entire directory to model storage with parallel workers."""
        print(f"Collecting files from {source_dir}...")
        try:
            files = self._collect_files(source_dir)
        except (FileNotFoundError, NotADirectoryError) as e:
            print(f"Error: {e}")
            return 0, 0

        if not files:
            print("No files found to upload.")
            return 0, 0

        total_size = sum(Path(local_path).stat().st_size for local_path, _ in files)
        print(f"Found {len(files)} files to upload ({total_size:,} bytes total).")

        successful_uploads = 0
        failed_uploads = []
        
        if tqdm:
            # Use rich progress bars if tqdm is installed
            main_progress = tqdm(total=len(files), desc="Overall Progress", unit="file", position=0)
            progress_bars = {}
            progress_lock = threading.Lock()

            def upload_task(file_info, position):
                local_path, relative_path = file_info
                file_size = Path(local_path).stat().st_size
                
                with progress_lock:
                    file_progress = tqdm(total=file_size, desc=relative_path[:30], unit="B", unit_scale=True, position=position, leave=False)
                    progress_bars[relative_path] = file_progress

                try:
                    presigned_url = self._get_presigned_url(relative_path, self._get_content_type(local_path))
                    success = self._upload_file_with_presigned_url(local_path, presigned_url, file_progress)
                    
                    status_icon = "✓" if success else "✗"
                    file_progress.set_description(f"{status_icon} {relative_path[:28]}")
                    return success, relative_path
                except Exception as e:
                    file_progress.set_description(f"✗ {relative_path[:28]}")
                    self.logger.error("Upload task failed for %s: %s", relative_path, e, exc_info=self.debug)
                    return False, relative_path
                finally:
                    file_progress.close()
                    main_progress.update(1)

            with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
                future_to_file = {
                    executor.submit(upload_task, file_info, i + 1): file_info
                    for i, file_info in enumerate(files)
                }
                for future in as_completed(future_to_file):
                    success, relative_path = future.result()
                    if success:
                        successful_uploads += 1
                    else:
                        failed_uploads.append(relative_path)
            main_progress.close()
        else:
            # Fallback to simple print statements
            def upload_task_simple(file_info):
                local_path, relative_path = file_info
                try:
                    presigned_url = self._get_presigned_url(relative_path, self._get_content_type(local_path))
                    success = self._upload_file_with_presigned_url(local_path, presigned_url)
                    print(f"{'✓' if success else '✗'} {relative_path}")
                    return success, relative_path
                except Exception as e:
                    print(f"✗ {relative_path}: {e}")
                    return False, relative_path

            with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
                future_to_file = {executor.submit(upload_task_simple, file_info): file_info for file_info in files}
                for future in as_completed(future_to_file):
                    success, relative_path = future.result()
                    if success:
                        successful_uploads += 1
                    else:
                        failed_uploads.append(relative_path)

        print(f"\nUpload completed: {successful_uploads}/{len(files)} files successful.")
        if failed_uploads:
            print(f"Failed uploads ({len(failed_uploads)}):")
            for failed_file in failed_uploads[:10]:
                print(f"  - {failed_file}")
            if len(failed_uploads) > 10:
                print(f"  ... and {len(failed_uploads) - 10} more.")
        
        return successful_uploads, len(files)


def handle_dry_run(args, uploader):
    print("DRY RUN: No files will be uploaded.")
    if args.source_file:
        file_path = Path(args.source_file)
        if not file_path.is_file():
            sys.exit(f"Error: Source is not a valid file: {args.source_file}")
        object_name = args.object_name or file_path.name
        print(f"Would upload 1 file: {file_path.name} -> {object_name}")
    else:
        try:
            files = uploader._collect_files(args.source_dir)
            total_size = sum(Path(local_path).stat().st_size for local_path, _ in files)
            print(f"Would upload {len(files)} files ({total_size:,} bytes total):")
            for _, relative_path in files:
                print(f"  - {relative_path}")
        except (FileNotFoundError, NotADirectoryError) as e:
            sys.exit(f"Error: {e}")

def main():
    parser = argparse.ArgumentParser(description="Upload files to model storage.", formatter_class=argparse.RawDescriptionHelpFormatter)
    source_group = parser.add_mutually_exclusive_group(required=True)
    source_group.add_argument('--source-dir', help='Source directory to upload.')
    source_group.add_argument('--source-file', help='Source file to upload.')
    parser.add_argument('--object-name', help='Object name in storage (for single file upload).')
    parser.add_argument('--namespace', required=True, help='Kubernetes namespace.')
    parser.add_argument('--model-name', required=True, help='Model name.')
    parser.add_argument('--bearer-token', required=True, help='Authentication token.')
    parser.add_argument('--api-server', help='API server URL.')
    parser.add_argument('--skip-ssl-verify', action='store_true', default=True, help='Skip SSL verification.')
    parser.add_argument('--verify-ssl', action='store_false', dest='skip_ssl_verify', help='Enable SSL verification.')
    parser.add_argument('--timeout', type=int, default=300, help='Upload timeout in seconds.')
    parser.add_argument('--max-workers', type=int, default=4, help='Max concurrent uploads.')
    parser.add_argument('--dry-run', action='store_true', help='Simulate upload.')
    parser.add_argument('--debug', action='store_true', help='Enable debug logging.')
    args = parser.parse_args()

    if not tqdm:
        print("Warning: tqdm not found. Progress bars disabled. `pip install tqdm`")

    uploader = ModelUploader(
        namespace=args.namespace, model_name=args.model_name, timeout=args.timeout,
        max_workers=args.max_workers, bearer_token=args.bearer_token,
        skip_ssl_verify=args.skip_ssl_verify, debug=args.debug, api_server=args.api_server
    )

    if args.dry_run:
        handle_dry_run(args, uploader)
        return

    try:
        start_time = time.time()
        if args.source_file:
            success = uploader.upload_file(args.source_file, args.object_name)
            successful, total = (1, 1) if success else (0, 1)
        else:
            successful, total = uploader.upload_directory(args.source_dir)
        
        print(f"\nCompleted in {time.time() - start_time:.2f}s.")
        if successful == total and total > 0:
            print("✅ All files uploaded successfully!")
            sys.exit(0)
        elif total > 0:
            sys.exit(f"⚠️ {total - successful} files failed to upload.")
        else:
            sys.exit(0)
            
    except (KeyboardInterrupt, Exception) as e:
        if isinstance(e, KeyboardInterrupt):
            print("\n❌ Upload cancelled by user.")
            sys.exit(130)
        print(f"❌ Error: {e}")
        if args.debug:
            traceback.print_exc()
        sys.exit(1)

if __name__ == '__main__':
    main()
