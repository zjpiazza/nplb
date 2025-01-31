from fastapi import APIRouter, Depends, HTTPException
from ...core.config import get_settings, Settings
from ...services.github import GitHubService
from ...services.repository import RepositoryService
from ...services.storage import R2StorageService
from ...core.models import RepositoryResponse
import tempfile
import os

router = APIRouter()

@router.post("/upload")
async def upload_file(
    settings: Settings = Depends(get_settings)
):
    storage = R2StorageService(settings.r2_account_id, settings.r2_access_key_id, settings.r2_secret_access_key, settings.r2_bucket_name)
    storage.upload_file(file_path="test/file.txt", key="test/file.txt")
    return {"message": "File uploaded successfully"}

@router.post("/repositories/{owner}/{repo}/build", response_model=RepositoryResponse)
async def build_repository(
    owner: str,
    repo: str,
    settings: Settings = Depends(get_settings)
):
    github_service = GitHubService(settings.github_token)
    releases = github_service.get_releases(owner, repo)
    
    # Initialize R2 storage
    storage = R2StorageService(
        account_id=settings.r2_account_id,
        access_key_id=settings.r2_access_key_id,
        secret_access_key=settings.r2_secret_access_key,
        bucket_name=settings.r2_bucket_name
    )
    
    repo_service = RepositoryService(
        output_dir=settings.output_dir,
        repo_name=f"{owner}/{repo}",
        base_url=settings.storage_url,
        storage=storage
    )
    
    repo_service.init_repository()
    
    with tempfile.TemporaryDirectory() as temp_dir:
        for release in releases:
            for asset in release.assets:
                temp_deb_path = os.path.join(temp_dir, asset.name)
                try:
                    repo_service.debian_service.download_file(asset.download_url, temp_deb_path)
                    repo_service.add_package(temp_deb_path)
                finally:
                    if os.path.exists(temp_deb_path):
                        os.remove(temp_deb_path)
    
    repo_service.generate_metadata()
    repo_service.publish()  # Upload to R2
    
    return RepositoryResponse(
        status="success",
        message=f"Repository built and published successfully for {owner}/{repo}"
    )