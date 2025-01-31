from pydantic import BaseModel
from typing import List, Optional
from datetime import datetime

class DebAsset(BaseModel):
    name: str
    download_url: str
    size: int

class Release(BaseModel):
    tag_name: str
    name: str
    published_at: Optional[datetime]
    assets: List[DebAsset]

class DebInfo(BaseModel):
    package: str
    version: str
    architecture: str
    depends: str
    description: str

class RepositoryResponse(BaseModel):
    status: str
    message: str 