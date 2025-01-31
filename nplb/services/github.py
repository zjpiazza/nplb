from typing import List
from github import Github
from ..core.models import Release, DebAsset

class GitHubService:
    def __init__(self, github_token: str):
        self.client = Github(github_token)
    
    def get_releases(self, owner: str, repo: str) -> List[Release]:
        repo = self.client.get_repo(f"{owner}/{repo}")
        releases = []
        
        for release in repo.get_releases():
            assets = []
            for asset in release.get_assets():
                if asset.name.endswith('.deb'):
                    assets.append(DebAsset(
                        name=asset.name,
                        download_url=asset.browser_download_url,
                        size=asset.size
                    ))
            
            if assets:
                releases.append(Release(
                    tag_name=release.tag_name,
                    name=release.title or release.tag_name,
                    published_at=release.published_at,
                    assets=assets
                ))
        
        return releases 