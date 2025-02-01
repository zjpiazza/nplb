from nplb.services.github import GitHubService
from nplb.services.repository import RepositoryService
from nplb.core.config import get_settings, Settings
from loguru import logger
from .exceptions import BuildRepositoryError

def build_repository_task(owner: str, repo: str, limit: int = 1, github_service: GitHubService = None, repo_service: RepositoryService = None):
    """Build a Debian repository from GitHub releases."""
    try:
        
        # Get repository releases
        releases = github_service.get_releases(owner, repo, limit)
        if not releases:
            raise ValueError(f"No releases found for {owner}/{repo}")
        
        try:
            # Create repository structure
            repo_service.create_repository()
            
            # Download release artifacts
            repo_service.download_artifacts(releases)
            
            # Generate metadata
            repo_service.generate_metadata()
            
        finally:
            # Clean up temporary files
            pass
            repo_service.cleanup()
        
    except Exception as e:
        logger.error(f"Failed to build repository: {str(e)}")
        raise BuildRepositoryError(f"Failed to build repository: {str(e)}")