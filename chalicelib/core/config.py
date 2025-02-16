from chalice import Chalice
import os
from typing import Optional

class Settings:
    def __init__(self):
        self.github_token: str = os.environ.get('GITHUB_TOKEN', '')
        self.storage_url: str = os.environ.get('STORAGE_URL', '')
        self.gpg_home: str = os.environ.get('GPG_HOME', '/tmp/.gnupg')
        self.gpg_key_email: str = os.environ.get('GPG_KEY_EMAIL', '')
        self.aws_access_key_id: str = os.environ.get('AWS_ACCESS_KEY_ID', '')
        self.aws_secret_access_key: str = os.environ.get('AWS_SECRET_ACCESS_KEY', '')
        self.aws_bucket_name: str = os.environ.get('AWS_BUCKET_NAME', '')
        self.aws_region: str = os.environ.get('AWS_REGION', 'us-east-1')
        self.output_dir: str = os.environ.get('OUTPUT_DIR', '/tmp/nplb')

_settings: Optional[Settings] = None

def get_settings() -> Settings:
    """Get application settings."""
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings 