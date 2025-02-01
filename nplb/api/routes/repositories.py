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

# def get_queue():
#     return Queue(connection=Redis())

def test_task():
    logger.info("Testing task")
    return "Hello World"


router = APIRouter()
@router.get("/test")
def test():
    redis = Redis()
    redis.set("test", "Hello World")
    return "Hello World"

@router.post("/build")
def build_repository(
    owner: str,
    repo: str,
    limit: int = 1,
    settings: Settings = Depends(get_settings),
):
    logger.info("Getting queue")
    q = Queue(connection=Redis(), default_timeout=5)
    try:
        logger.info("Creating GitHub service")
        github_service = GitHubService(settings.github_token)
        logger.info("Creating Repository service")
        repo_service = RepositoryService(
            repo_name=f"{owner}/{repo}",
            base_url=settings.storage_url,
        )
        logger.info("Enqueuing job")
        job = q.enqueue(test_task)
        # job = q.enqueue(build_repository_task, owner, repo, limit, github_service, repo_service)
        logger.info("Building repository")
        build_repository_task(owner, repo, limit, github_service, repo_service)
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