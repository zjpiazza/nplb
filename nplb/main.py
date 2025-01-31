from fastapi import FastAPI
from .api.routes import repositories
import uvicorn

app = FastAPI(
    title="NPLB - APT Repository Generator",
    description="API for generating APT repositories from GitHub releases",
    version="1.0.0"
)

app.include_router(repositories.router, prefix="/api/v1", tags=["repositories"])

if __name__ == "__main__":
    uvicorn.run("nplb.main:app", host="0.0.0.0", port=8000, reload=True)
