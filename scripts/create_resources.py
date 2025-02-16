import boto3
from botocore.exceptions import ClientError
from loguru import logger
def create_tables():
    """Create required DynamoDB tables for local development."""
    # Configure DynamoDB client for local development
    dynamodb = boto3.resource(
        'dynamodb',
        endpoint_url='http://localhost:4566',
        region_name='us-east-1',
        aws_access_key_id='dummy',
        aws_secret_access_key='dummy'
    )
    
    try:
        # Create repositories table
        table = dynamodb.create_table(
            TableName='nplb-repositories',
            KeySchema=[
                {
                    'AttributeName': 'id',
                    'KeyType': 'HASH'
                }
            ],
            AttributeDefinitions=[
                {
                    'AttributeName': 'id',
                    'AttributeType': 'S'
                }
            ],
            ProvisionedThroughput={
                'ReadCapacityUnits': 5,
                'WriteCapacityUnits': 5
            }
        )
        
        # Wait for table to be created
        table.meta.client.get_waiter('table_exists').wait(
            TableName='nplb-repositories'
        )
        logger.info("Repositories table created successfully!")
        
    except ClientError as e:
        if e.response['Error']['Code'] == 'ResourceInUseException':
            logger.warning("Table already exists")
        else:
            logger.error(f"Error creating table: {str(e)}")
            raise

def create_queue():
    """Create required SQS queue for local development."""
    sqs = boto3.client(
        'sqs',
        endpoint_url='http://localhost:4566',
        region_name='us-east-1',
        aws_access_key_id='dummy',
        aws_secret_access_key='dummy'
    )
    
    try:
        sqs.create_queue(QueueName='builds')
        logger.info("Builds queue created successfully!")
    except ClientError as e:
        if e.response['Error']['Code'] == 'QueueAlreadyExists':
            logger.warning("Queue already exists")
        else:
            logger.error(f"Error creating queue: {str(e)}")
            raise

def create_s3_bucket():
    """Create required S3 bucket for local development."""
    s3 = boto3.client(
        's3',
        endpoint_url='http://localhost:4566',
        region_name='us-east-1',
        aws_access_key_id='dummy',
        aws_secret_access_key='dummy'
    )
    
    try:
        s3.create_bucket(Bucket='nplb-artifacts')
        logger.info("Artifacts bucket created successfully!")
    except ClientError as e:
        if e.response['Error']['Code'] == 'BucketAlreadyExists':
            logger.warning("Bucket already exists")
        else:
            logger.error(f"Error creating bucket: {str(e)}")
            raise

def create_ssm_parameter():
    try:
        ssm = boto3.client(
            'ssm',
            endpoint_url='http://localhost:4566',
            region_name='us-east-1',
        )
        with open('.env', 'r') as f:
            for line in f:
                key, value = line.strip().split('=')
                ssm.put_parameter(
                    Name=key,
                    Value=value,
                    Type='String'
                )
        logger.info("SSM parameters created successfully!")
    except ClientError as e:
        logger.error(f"Error creating SSM parameter: {str(e)}")
        raise


if __name__ == '__main__':
    create_tables()
    create_queue()
    create_s3_bucket()
    create_ssm_parameter()