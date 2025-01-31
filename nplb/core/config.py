from pydantic_settings import BaseSettings
from functools import lru_cache

class Settings(BaseSettings):
    github_token: str
    base_url: str = "https://nplb.wastelandsystems.io"
    output_dir: str = "build"
    default_codename: str = "stable"
    
    # R2 Configuration
    r2_account_id: str
    r2_access_key_id: str
    r2_secret_access_key: str
    r2_bucket_name: str
    r2_bucket_region: str = "auto"
    r2_public_url: str = None  # Will default to {bucket_name}.r2.dev if not set
    
    @property
    def storage_url(self) -> str:
        if self.r2_public_url:
            return self.r2_public_url
        return f"https://{self.r2_bucket_name}.r2.dev"
    
    class Config:
        env_file = ".env"

@lru_cache()
def get_settings():
    return Settings() 