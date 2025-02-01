from fastapi import APIRouter, Depends, HTTPException
from ...core.config import get_settings, Settings
from ...services.github import GitHubService
from ...services.repository import RepositoryService
from ...core.models import RepositoryResponse
from loguru import logger
from ...tasks.build import build_repository_task
from ...tasks.exceptions import BuildRepositoryError
from rq import Queue
from rq.job import Job
from redis import Redis

router = APIRouter()

@router.post("/build")
def build_repository(
    owner: str,
    repo: str,
    limit: int = 1,
    settings: Settings = Depends(get_settings),
):
    # TODO: Use proper dependency injection
    q = Queue(connection=Redis(settings.redis_host, settings.redis_port, settings.redis_password, settings.redis_db))
    logger.info("Getting queue")
    try:
        logger.info("Creating GitHub service")
        github_service = GitHubService(settings.github_token)
        logger.info("Creating Repository service")
        repo_service = RepositoryService(
            repo_name=f"{owner}/{repo}",
            base_url=settings.storage_url,
        )
        logger.info("Enqueuing job")
        job = q.enqueue(build_repository_task, owner, repo, limit, github_service, repo_service)
        logger.info("Building repository")
        logger.info("Repository built")
        return RepositoryResponse(
            status="success",
            message=f"Job {owner}/{repo} queued",
            job_id=job.id
        )
    except BuildRepositoryError as e:
        raise HTTPException(status_code=500, detail=str(e))
    

def get_job(job_id: str):
    job = Job.fetch(job_id, connection=Redis())
    return job.result