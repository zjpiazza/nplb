from chalice import Chalice
from nplb.utils.resources import ResourceProvider
from nplb.services.github import GitHubService
from nplb.services.repository import RepositoryService
import json

app = Chalice(app_name='nplb-worker')
resources = ResourceProvider.get_resources()

@app.on_sqs_message(queue=resources.build_queue)
def handle_build_job(event):
    for record in event:
        payload = json.loads(record.body)
        owner = payload['owner']
        repo = payload['repo']
        limit = payload['limit']

        github_service = GitHubService(resources.github_token)
        repo_service = RepositoryService(
            repo_name=f"{owner}/{repo}",
            base_url=resources.storage_url,
        )

        try:
            # Process repository build
            releases = github_service.get_releases(owner, repo, limit)
            repo_service.build_repository(releases)
            
            # Update build status in DynamoDB
            resources.repository_table.update_item(
                Key={'id': f"{owner}/{repo}"},
                UpdateExpression="SET build_status = :status",
                ExpressionAttributeValues={':status': 'completed'}
            )
        except Exception as e:
            # Update build status with error
            resources.repository_table.update_item(
                Key={'id': f"{owner}/{repo}"},
                UpdateExpression="SET build_status = :status, error = :error",
                ExpressionAttributeValues={
                    ':status': 'failed',
                    ':error': str(e)
                }
            )
            raise 