from chalice import Blueprint, Response
from ...services.github import GitHubService
from ...services.repository import RepositoryService
from ...tasks.exceptions import BuildRepositoryError
from ...utils.resources import ResourceProvider
from loguru import logger
import json

# Create blueprint
blueprint = Blueprint(__name__)
resources = ResourceProvider.get_resources()

@blueprint.route('/list', methods=['GET'])
def list_repositories():
    try:
        # Get repositories from DynamoDB
        response = resources.repository_table.scan()
        return {'repositories': response.get('Items', [])}
    except resources.repository_table.meta.client.exceptions.ResourceNotFoundException:
        # Return empty list if table doesn't exist yet
        logger.warning("Repository table does not exist yet")
        return {'repositories': []}
    except Exception as e:
        logger.error(f"Error scanning repository table: {str(e)}")
        return Response(
            body={'error': 'Internal server error'},
            status_code=500
        )

@blueprint.route('/{repo_id}', methods=['GET'])
def get_repository(repo_id):
    response = resources.repository_table.get_item(
        Key={'id': repo_id}
    )
    if 'Item' not in response:
        return Response(
            body={'error': 'Repository not found'},
            status_code=404
        )
    return response['Item']

@blueprint.route('/build', methods=['POST'])
def build_repository():
    request = blueprint.current_request
    payload = request.json_body
    owner = payload.get('owner')
    repo = payload.get('repo')
    limit = payload.get('limit', 1)

    # Validate input
    if not owner or not repo:
        return Response(
            body={'error': 'owner and repo are required'},
            status_code=400
        )

    try:
        # Queue the build job
        message = {
            'owner': owner,
            'repo': repo,
            'limit': limit
        }
        
        response = resources.sqs.send_message(
            QueueUrl=resources.build_queue,
            MessageBody=json.dumps(message)
        )

        return {
            'status': 'success',
            'message': f'Build queued for {owner}/{repo}',
            'job_id': response['MessageId']
        }
    except BuildRepositoryError as e:
        return Response(
            body={'error': str(e)},
            status_code=500
        )