from chalice import Chalice, AuthResponse, Response
import os
import boto3
import json
from nplb.api.routes.repositories import blueprint as repositories

app = Chalice(app_name='nplb')

# Configure API to not require authentication by default
app.api.default_authorizer = None

def setup_local_resources():
    # Common configs for local AWS services
    local_config = {
        'endpoint_url': 'http://localhost:4566',  # LocalStack default endpoint
        'region_name': 'local',
        'aws_access_key_id': 'dummy',
        'aws_secret_access_key': 'dummy'
    }

    # Setup DynamoDB
    dynamodb = boto3.resource('dynamodb', **local_config)
    table_name = 'repository-table'
    existing_tables = dynamodb.meta.client.list_tables()['TableNames']
    
    if table_name not in existing_tables:
        table = dynamodb.create_table(
            TableName=table_name,
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
        table.wait_until_exists()
        print(f"Created DynamoDB table: {table_name}")

    # Setup SQS
    sqs = boto3.client('sqs', **local_config)
    queue_name = 'build-queue'
    try:
        queue = sqs.create_queue(
            QueueName=queue_name,
            Attributes={
                'VisibilityTimeout': '60',  # 60 seconds
                'MessageRetentionPeriod': '86400'  # 1 day
            }
        )
        print(f"Created SQS queue: {queue_name}")
    except sqs.exceptions.QueueNameExists:
        print(f"SQS queue already exists: {queue_name}")

# Setup local resources if running locally
if os.environ.get('AWS_CHALICE_CLI_MODE') == 'LOCAL':
    setup_local_resources()

# Add a test route to verify our setup
@app.route('/')
def index():
    return Response(
        body={'message': 'API is working'},
        status_code=200,
        headers={'Content-Type': 'application/json'}
    )

# Register blueprints
app.register_blueprint(repositories, url_prefix='/repositories')

# Add CORS configuration
app.cors = True

# Disable authentication requirements for local development
# if os.environ.get('CHALICE_ENV') == 'local':

# Add a catch-all authorizer that always allows requests
@app.authorizer()
def always_allow(auth_request):
    return AuthResponse(routes=['*'], principal_id='user')