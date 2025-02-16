from typing import Optional
import boto3
from botocore.config import Config
import os

class ResourceProvider:
    _instance: Optional['ResourceProvider'] = None
    
    def __init__(self):
        # Configure boto3 client
        config = Config(
            retries=dict(
                max_attempts=3
            )
        )
        
        # Decide if we're local or not
        endpoint_url = None
        if os.getenv("IS_OFFLINE", "false").lower() == "true":
            endpoint_url = "http://localhost:4566"

        # Initialize AWS resources with local endpoints for development
        self.dynamodb = boto3.resource(
            'dynamodb',
            endpoint_url=endpoint_url,  # now it's conditional
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy'
        )
        
        self.sqs = boto3.client(
            'sqs',
            endpoint_url=endpoint_url,  # also conditional
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy',
            config=config
        )
        
        # Get table and queue references
        self.repository_table = self.dynamodb.Table('nplb-repositories')
        self.build_queue = 'builds'

        self.ssm = boto3.client(
            'ssm',
            endpoint_url=endpoint_url,  # also conditional
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy'
        )
        
    @classmethod
    def get_resources(cls) -> 'ResourceProvider':
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance 

class LambdaResourceProvider:
    _instance: Optional['LambdaResourceProvider'] = None
    
    def __init__(self):
        # Configure boto3 client
        config = Config(
            retries=dict(
                max_attempts=3
            )
        )
        
        # LocalStack endpoint for Lambda environment
        endpoint_url = "http://localhost.localstack.cloud:4566"

        # Initialize AWS resources with LocalStack endpoints
        self.dynamodb = boto3.resource(
            'dynamodb',
            endpoint_url=endpoint_url,
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy'
        )
        
        self.sqs = boto3.client(
            'sqs',
            endpoint_url=endpoint_url,
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy',
            config=config
        )
        
        # Get table and queue references
        self.repository_table = self.dynamodb.Table('nplb-repositories')
        self.build_queue = 'builds'

        self.ssm = boto3.client(
            'ssm',
            endpoint_url=endpoint_url,
            region_name='us-east-1',
            aws_access_key_id='dummy',
            aws_secret_access_key='dummy'
        )
        
    @classmethod
    def get_resources(cls) -> 'LambdaResourceProvider':
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance 
