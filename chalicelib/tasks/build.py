import os
from loguru import logger
from chalicelib.services.github import GitHubService
from chalicelib.services.repository import RepositoryService
from chalicelib.services.storage import S3Service
from chalicelib.utils.resources import LambdaResourceProvider
import json
from chalice.app import SQSEvent
import traceback
import stat

resources = LambdaResourceProvider.get_resources()

def setup_binary():
    """Set up the unzstd binary for use."""
    # Add /var/task to PATH
    os.environ['PATH'] = f"/var/task:{os.environ['PATH']}"
    
    return '/var/task/unzstd'

def build(event: SQSEvent):
    """
    Generate an APT repository from GitHub releases in response to a queue message.
    
    Expects one or more messages with keys like:
      {
        "owner": "some_github_owner",
        "repo": "some_github_repo",
        "limit": 1
      }
    """

    try:
        # Set up binary before processing
        # setup_binary()

        # Iterate over each record in the SQSEvent
        for record in event:
            message = json.loads(record.body)

        owner = message["owner"]
        repo = message["repo"]
        limit = message.get("limit", 1)

        if not (owner and repo):
            raise ValueError("Message must contain 'owner' and 'repo' fields.")

        github_token_parameter = resources.ssm.get_parameter(Name="GITHUB_TOKEN")
        github_token = github_token_parameter["Parameter"]["Value"]

        # Fetch releases from GitHub
        git_service = GitHubService(github_token=github_token)
        releases = git_service.get_releases(owner, repo, limit=limit)

        # Create and populate repository
        repo_service = RepositoryService(
            repo_name=f"{owner}/{repo}",
            repo_owner=owner
        )
        repo_service.create_repository()
        repo_service.download_artifacts(releases=releases)
        repo_service.generate_metadata()
        # Upload artifacts to S3
        s3_service = S3Service()
        s3_service.upload_artifacts(repo_service.artifacts_dir)
        resources.repository_table.put_item(
            Item={
                'owner': owner,
                'repo': repo,
                'status': 'completed'
            }
        )

        logger.info(f"APT repository build complete for {owner}/{repo}.")
    except Exception as e:
        # Log the full stack trace
        logger.error(f"Exception: {str(e)}")
        logger.error(f"Stack trace: {traceback.format_exc()}")
        raise
