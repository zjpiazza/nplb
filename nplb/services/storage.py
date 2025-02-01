import boto3
from pathlib import Path
from typing import List
import mimetypes
import os
from botocore.exceptions import ClientError
from time import sleep

class S3StorageService:
    def __init__(
        self,
        access_key_id: str,
        secret_access_key: str,
        bucket_name: str,
        region: str
    ):
        self.bucket_name = bucket_name
        self.client = boto3.client(
            's3',
            aws_access_key_id=access_key_id,
            aws_secret_access_key=secret_access_key,
            region_name=region
        )

    def _get_content_type(self, filename: str) -> str:
        content_type, _ = mimetypes.guess_type(filename)
        return content_type or 'application/octet-stream'

    def upload_file(self, file_path: str | Path, key: str, max_retries: int = 5) -> str:
        """Upload a single file to S3"""
        file_path = Path(file_path).resolve()
        print(f"Uploading {file_path} to {key}")
        extra_args = {
            'ContentType': self._get_content_type(str(file_path))
        }
        
        # If the file is a Release or Packages file, set cache control
        if file_path.name in ['Release', 'InRelease', 'Packages', 'Packages.gz', 'Packages.xz']:
            extra_args['CacheControl'] = 'no-cache, no-store, must-revalidate'
            extra_args['Expires'] = '0'
        
        retry_count = 0
        while retry_count < max_retries:
            try:
                print(f"Attempt {retry_count + 1}: Uploading {file_path} to {key} with extra args: {extra_args}")
                self.client.upload_file(
                    str(file_path),
                    self.bucket_name,
                    key,
                    ExtraArgs=extra_args
                )
                return key
            except ClientError as e:
                error_code = e.response.get('Error', {}).get('Code', '')
                error_message = e.response.get('Error', {}).get('Message', '')
                print(f"Error uploading file (attempt {retry_count + 1}): {error_code} - {error_message}")
                
                if error_code in ['500', 'InternalError']:
                    retry_count += 1
                    if retry_count < max_retries:
                        sleep_time = 2 ** retry_count
                        print(f"Retrying in {sleep_time} seconds...")
                        sleep(sleep_time)
                    continue
                else:
                    raise
            except Exception as e:
                print(f"Unexpected error uploading file: {str(e)}")
                raise

        raise Exception(f"Failed to upload {file_path} after {max_retries} attempts")

    def upload_directory(self, directory: str | Path, prefix: str = "") -> List[str]:
        """Upload an entire directory to S3"""
        directory = Path(directory)
        uploaded_files = []
        
        for root, _, files in os.walk(directory):
            for file in files:
                file_path = Path(root) / file
                relative_path = file_path.relative_to(directory)
                key = str(Path(prefix) / relative_path)
                
                self.upload_file(file_path, key)
                uploaded_files.append(key)
        
        return uploaded_files

    def delete_prefix(self, prefix: str):
        """Delete all objects with the given prefix"""
        paginator = self.client.get_paginator('list_objects_v2')
        
        for page in paginator.paginate(Bucket=self.bucket_name, Prefix=prefix):
            if 'Contents' in page:
                objects = [{'Key': obj['Key']} for obj in page['Contents']]
                self.client.delete_objects(
                    Bucket=self.bucket_name,
                    Delete={'Objects': objects}
                ) 