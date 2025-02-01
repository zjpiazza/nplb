from pydantic_settings import BaseSettings
from functools import lru_cache

class Settings(BaseSettings):
    github_token: str
    base_url: str = "https://nplb.wastelandsystems.io"
    output_dir: str = "build"
    default_codename: str = "stable"
    
    # S3 Configuration
    aws_access_key_id: str
    aws_secret_access_key: str
    aws_bucket_name: str
    aws_region: str = "us-east-1"
    aws_public_url: str | None = None
    
    # GPG Configuration
    gpg_home: str = "keys"  # Default location for GPG keys
    gpg_key_email: str | None = None  # Email associated with signing key
    
    @property
    def storage_url(self) -> str:
        if self.aws_public_url:
            return self.aws_public_url
        return f"https://{self.aws_bucket_name}.s3.{self.aws_region}.amazonaws.com"
    
    class Config:
        env_file = ".env"

@lru_cache()
def get_settings():
    return Settings() 