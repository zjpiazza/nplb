from fastapi import FastAPI
from .api.routes import repositories
import uvicorn
from requests_cache import DO_NOT_CACHE, get_cache, install_cache

app = FastAPI(
    title="NPLB - APT Repository Generator",
    description="API for generating APT repositories from GitHub releases",
    version="1.0.0"
)

app.include_router(repositories.router, prefix="/api/v1", tags=["repositories"])

install_cache(
    cache_control=True,
    urls_expire_after={
        '*.github.com': 360,  # Placeholder expiration; should be overridden by Cache-Control
        '*': DO_NOT_CACHE,  # Don't cache anything other than GitHub requests
    },
)

if __name__ == "__main__":
    uvicorn.run("nplb.main:app", host="0.0.0.0", port=8000, reload=True)
