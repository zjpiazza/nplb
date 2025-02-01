from pathlib import Path
from typing import List, Dict
import os
import tempfile
from loguru import logger
import requests
import hashlib
import gnupg
from debian import debfile  # For parsing .deb files

class RepositoryService:
    def __init__(
        self,
        repo_name: str,
        base_url: str,
        gpg_home: str = None,
        gpg_key_email: str = None
    ):
        """
        Initialize repository service.
        
        Args:
            repo_name: Name of repository (e.g. 'owner/repo')
            base_url: Base URL for the repository
            gpg_home: Path to GPG home directory
            gpg_key_email: Email of GPG key to use for signing
        """
        self.repo_name = repo_name
        self.base_url = base_url
        self.temp_dir = None
        self.pool_dir = None
        self.dists_dir = None
        self.gpg_home = gpg_home
        self.gpg_key_email = gpg_key_email
        
    def create_repository(self) -> str:
        """
        Create a new temporary repository structure.
        
        Returns:
            Path to the temporary directory containing the repository
        """
        # Create temp directory that will persist until cleanup is called
        self.temp_dir = tempfile.mkdtemp()
        logger.info(f"Created temporary directory at {self.temp_dir}")
        
        # Create repository structure
        self.pool_dir = os.path.join(self.temp_dir, "pool", "main")
        self.dists_dir = os.path.join(self.temp_dir, "dists", "stable")
        
        os.makedirs(self.pool_dir, exist_ok=True)
        os.makedirs(self.dists_dir, exist_ok=True)
        
        logger.info("Repository structure created")
        return self.temp_dir
        
    def download_artifacts(self, releases: List[dict]) -> None:
        """
        Download release artifacts into pool directory.
        
        Args:
            releases: List of GitHub release objects containing assets
        """
        if not self.pool_dir:
            raise ValueError("Repository not initialized. Call create_repository() first.")
            
        for release in releases:
            for asset in release.assets:
                if not asset.name.endswith('.deb'):
                    logger.debug(f"Skipping non-deb asset: {asset.name}")
                    continue
                    
                dest_path = os.path.join(self.pool_dir, asset.name)
                logger.info(f"Downloading {asset.name} to {dest_path}")
                
                self._download_file(asset.download_url, dest_path)
                
    def _download_file(self, url: str, dest_path: str) -> None:
        """
        Download a file from a URL to a destination path.
        
        Args:
            url: URL to download from
            dest_path: Path to save the file to
        """
        response = requests.get(url, stream=True)
        response.raise_for_status()
        
        with open(dest_path, 'wb') as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
                
    def cleanup(self) -> None:
        """Remove temporary directory and all contents."""
        if self.temp_dir and os.path.exists(self.temp_dir):
            import shutil
            shutil.rmtree(self.temp_dir)
            logger.info(f"Cleaned up temporary directory {self.temp_dir}")
            self.temp_dir = None
            self.pool_dir = None
            self.dists_dir = None 

    def generate_metadata(self) -> None:
        """Generate repository metadata files."""
        if not self.pool_dir or not self.dists_dir:
            raise ValueError("Repository not initialized. Call create_repository() first.")
            
        # Create binary package directories
        binary_dir = os.path.join(self.dists_dir, "main", "binary-amd64")
        os.makedirs(binary_dir, exist_ok=True)
        
        # Generate Packages file
        packages_path = os.path.join(binary_dir, "Packages")
        self._generate_packages_file(packages_path)
        
        # Generate compressed versions
        self._compress_file(packages_path)
        
        # Generate and sign Release file
        self._generate_release_file()
        
    def _generate_packages_file(self, packages_path: str) -> None:
        """Generate Packages file containing metadata for all .deb packages."""
        logger.info("Generating Packages file")
        
        with open(packages_path, 'w') as f:
            for deb_file in os.listdir(self.pool_dir):
                if not deb_file.endswith('.deb'):
                    continue
                    
                deb_path = os.path.join(self.pool_dir, deb_file)
                metadata = self._extract_deb_metadata(deb_path)
                
                # Write package metadata
                f.write(str(metadata))
                f.write('\n\n')
                
    def _generate_release_file(self) -> None:
        """Generate and sign Release file."""
        logger.info("Generating Release file")
        
        release_path = os.path.join(self.dists_dir, "Release")
        
        checksums: Dict[str, List[Dict]] = {
            'MD5Sum': [],
            'SHA1': [],
            'SHA256': []
        }
        
        # Calculate checksums for all files in dists/
        for root, _, files in os.walk(self.dists_dir):
            for filename in files:
                if filename in ['Release', 'Release.gpg']:
                    continue
                    
                filepath = os.path.join(root, filename)
                relpath = os.path.relpath(filepath, self.dists_dir)
                size = os.path.getsize(filepath)
                
                with open(filepath, 'rb') as f:
                    data = f.read()
                    checksums['MD5Sum'].append({
                        'hash': hashlib.md5(data).hexdigest(),
                        'size': size,
                        'path': relpath
                    })
                    checksums['SHA1'].append({
                        'hash': hashlib.sha1(data).hexdigest(),
                        'size': size,
                        'path': relpath
                    })
                    checksums['SHA256'].append({
                        'hash': hashlib.sha256(data).hexdigest(),
                        'size': size,
                        'path': relpath
                    })
        
        # Write Release file
        with open(release_path, 'w') as f:
            f.write(f"Origin: {self.repo_name}\n")
            f.write(f"Label: {self.repo_name} Repository\n")
            f.write("Suite: stable\n")
            f.write("Codename: stable\n")
            f.write("Components: main\n")
            f.write("Architectures: amd64\n")
            f.write(f"Date: {self._get_current_date()}\n")
            
            # Write checksums
            for algo in ['MD5Sum', 'SHA1', 'SHA256']:
                f.write(f"{algo}:\n")
                for entry in sorted(checksums[algo], key=lambda x: x['path']):
                    f.write(f" {entry['hash']} {entry['size']} {entry['path']}\n")
        
        logger.info("Release file generated")
        
        # Sign Release file if GPG is configured
        if self.gpg_home and self.gpg_key_email:
            self._sign_release()
            
    def _sign_release(self) -> None:
        """Sign the Release file with GPG."""
        logger.info("Signing Release file")
        
        gpg = gnupg.GPG(gnupghome=self.gpg_home)
        release_path = os.path.join(self.dists_dir, "Release")
        
        with open(release_path, 'rb') as f:
            signed_data = gpg.sign_file(
                f,
                keyid=self.gpg_key_email,
                detach=True,
                output=os.path.join(self.dists_dir, "Release.gpg")
            )
            
            if not signed_data:
                raise ValueError("Failed to sign Release file")
                
    def _compress_file(self, filepath: str) -> None:
        """Create compressed versions of a file (gz and xz)."""
        import gzip
        import lzma
        
        # Create gzip version
        gzip_path = filepath + '.gz'
        with open(filepath, 'rb') as f_in:
            with gzip.open(gzip_path, 'wb') as f_out:
                f_out.write(f_in.read())
        logger.debug(f"Created gzip compressed file: {gzip_path}")
        
        # Create xz version
        xz_path = filepath + '.xz'
        with open(filepath, 'rb') as f_in:
            with lzma.open(xz_path, 'wb') as f_out:
                f_out.write(f_in.read())
        logger.debug(f"Created xz compressed file: {xz_path}")
        
    @staticmethod
    def _get_current_date() -> str:
        """Get current date in Debian repository format."""
        from datetime import datetime, timezone
        return datetime.now(timezone.utc).strftime("%a, %d %b %Y %H:%M:%S %Z")

    def _extract_deb_metadata(self, deb_path: str) -> debfile.DebControl:
        """
        Extract metadata from a .deb file.
        
        Args:
            deb_path: Path to the .deb file
            
        Returns:
            DebControl object containing package metadata
        """
        # Parse package using python-debian
        deb = debfile.DebFile(deb_path)
        control_data = deb.control.debcontrol()
        
        # Add additional required fields
        filename = os.path.basename(deb_path)
        size = os.path.getsize(deb_path)
        
        # Calculate checksums
        with open(deb_path, 'rb') as f:
            data = f.read()
            md5sum = hashlib.md5(data).hexdigest()
            sha1 = hashlib.sha1(data).hexdigest()
            sha256 = hashlib.sha256(data).hexdigest()
        
        # Add fields required for the Packages file
        control_data['Filename'] = os.path.join('pool/main', filename)
        control_data['Size'] = str(size)
        control_data['MD5sum'] = md5sum
        control_data['SHA1'] = sha1
        control_data['SHA256'] = sha256
        
        return control_data 