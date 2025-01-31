from debian import debfile
from ..core.models import DebInfo
import requests

class DebianService:
    def extract_deb_info(self, deb_path: str) -> DebInfo:
        deb = debfile.DebFile(deb_path)
        control = deb.control.debcontrol()
        
        return DebInfo(
            package=control.get('Package', ''),
            version=control.get('Version', ''),
            architecture=control.get('Architecture', ''),
            depends=control.get('Depends', ''),
            description=control.get('Description', '')
        )

    def download_file(self, url: str, target_path: str):
        response = requests.get(url, stream=True)
        response.raise_for_status()
        
        with open(target_path, 'wb') as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk) 