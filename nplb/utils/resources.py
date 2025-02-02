from typing import Optional
import boto3
import os
from chalice import Chalice

class Resources:
    def __init__(self):
        self.sqs = boto3.client('sqs')
        self.dynamodb = boto3.resource('dynamodb')
        self.s3 = boto3.client('s3')
        self.github_token = os.getenv('GITHUB_TOKEN')
        self.storage_url = os.getenv('STORAGE_URL')
        
    @property
    def repository_table(self):
        return self.dynamodb.Table(os.getenv('REPOSITORY_TABLE', 'repositories'))
    
    @property
    def build_queue(self):
        return os.getenv('BUILD_QUEUE_NAME', 'repository-builds')

class ResourceProvider:
    _instance = None

    @classmethod
    def get_resources(cls) -> Resources:
        if cls._instance is None:
            cls._instance = Resources()
        return cls._instance 