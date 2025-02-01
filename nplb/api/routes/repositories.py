from fastapi import APIRouter, Depends, HTTPException
from ...core.config import get_settings, Settings
from ...services.github import GitHubService
from ...services.repository import RepositoryService
from ...core.models import RepositoryResponse
from loguru import logger

router = APIRouter()

@router.post("/repositories/{owner}/{repo}/build", response_model=RepositoryResponse)
async def build_repository(
    owner: str,
    repo: str,
    limit: int = 1,
    settings: Settings = Depends(get_settings)
):
    """Build a Debian repository from GitHub releases."""
    try:
        # Initialize services
        github_service = GitHubService(settings.github_token)
        repo_service = RepositoryService(
            repo_name=f"{owner}/{repo}",
            base_url=settings.storage_url,
        )
        
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
            
            return RepositoryResponse(
                status="success",
                message=f"Repository built successfully for {owner}/{repo}"
            )
            
        finally:
            # Clean up temporary files
            pass
            repo_service.cleanup()
            
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to build repository: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to build repository: {str(e)}"
        )