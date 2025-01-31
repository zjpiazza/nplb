#!/usr/bin/env python3

import os
import sys
import tempfile
import requests
import json
import shutil
import subprocess
from typing import Dict, List
from github import Github
from debian import debfile
from pathlib import Path
from pydpkg import Dpkg

def get_github_releases(owner: str, repo: str) -> List[Dict]:
    """
    Fetch all releases for a given GitHub repository
    """
    gh = Github(os.getenv("GITHUB_TOKEN"))
    repo = gh.get_repo(f"{owner}/{repo}")
    releases = []
    
    for release in repo.get_releases():
        assets = []
        for asset in release.get_assets():
            if asset.name.endswith('.deb'):
                assets.append({
                    'name': asset.name,
                    'download_url': asset.browser_download_url,
                    'size': asset.size
                })
        
        if assets:  # Only include releases with .deb files
            releases.append({
                'tag_name': release.tag_name,
                'name': release.title or release.tag_name,
                'published_at': release.published_at.isoformat() if release.published_at else None,
                'assets': assets
            })
    
    return releases

def extract_deb_info(deb_path: str) -> Dict:
    """
    Extract package information from a .deb file
    """
    deb = debfile.DebFile(deb_path)
    control = deb.control.debcontrol()
    
    return {
        'package': control.get('Package', ''),
        'version': control.get('Version', ''),
        'architecture': control.get('Architecture', ''),
        'depends': control.get('Depends', ''),
        'description': control.get('Description', '')
    }

def download_file(url: str, target_path: str):
    """
    Download a file from URL to target path
    """
    response = requests.get(url, stream=True)
    response.raise_for_status()
    
    with open(target_path, 'wb') as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)

class AptRepoGenerator:
    def __init__(self, output_dir: str, repo_name: str, base_url: str, codename: str = "stable", architectures: List[str] = None):
        self.output_dir = Path(output_dir)
        self.repo_name = repo_name
        self.base_url = base_url
        self.codename = codename
        self.architectures = architectures or ["amd64", "arm64"]
        self.pool_dir = self.output_dir / "pool"
        self.dists_dir = self.output_dir / "dists" / self.codename
        self.gpg_key = None

    def init_repository(self):
        """Initialize repository directory structure"""
        # Create basic directory structure
        for arch in self.architectures:
            (self.dists_dir / "main" / "binary-{}".format(arch)).mkdir(parents=True, exist_ok=True)
        self.pool_dir.mkdir(parents=True, exist_ok=True)

        # Create or load GPG key
        self._setup_gpg()

    def _setup_gpg(self):
        """Setup GPG signing key"""
        # Keys stored outside of build directory
        keys_dir = Path("keys")
        private_key = keys_dir / "private-key.gpg"
        public_key = keys_dir / "public-key.gpg"
        
        # Create keys directory if it doesn't exist
        keys_dir.mkdir(exist_ok=True)
        
        # Generate keypair if it doesn't exist
        if not private_key.exists():
            # Create a static key configuration
            key_config = """Key-Type: RSA
Key-Length: 4096
Name-Real: APT Repository
Name-Email: repo@example.com
Expire-Date: 0
%no-protection
%commit
"""
            # Write config to temporary file
            with tempfile.NamedTemporaryFile(mode='w', delete=False) as temp_config:
                temp_config.write(key_config)
                temp_config.flush()
                
                try:
                    # Generate new GPG key using the config file
                    subprocess.run([
                        "gpg", "--batch", "--gen-key",
                        "--passphrase", '',
                        temp_config.name
                    ], check=True)
                    
                    # Get the key fingerprint
                    key_info = subprocess.check_output([
                        "gpg", "--list-keys", "--with-colons", "repo@example.com"
                    ]).decode()
                    key_fingerprint = next(line.split(':')[4] for line in key_info.splitlines() if line.startswith('pub:'))
                    
                    # Export keys to keys directory
                    subprocess.run([
                        "gpg", "--export", "--output", str(public_key),
                        key_fingerprint
                    ], check=True)
                    
                    subprocess.run([
                        "gpg", "--export-secret-key", "--output", str(private_key),
                        key_fingerprint
                    ], check=True)
                    
                    # Clean up the key from the keyring using full fingerprint
                    subprocess.run([
                        "gpg", "--batch", "--yes", "--delete-secret-key",
                        key_fingerprint
                    ], check=True)
                    
                    subprocess.run([
                        "gpg", "--batch", "--yes", "--delete-key",
                        key_fingerprint
                    ], check=True)
                finally:
                    os.unlink(temp_config.name)
        
        # Import the private key for signing
        subprocess.run([
            "gpg", "--import", str(private_key)
        ], check=True)
        
        # Copy public key to build directory
        shutil.copy2(public_key, self.output_dir / "key.gpg")

    def add_package(self, deb_path: str):
        """Add a package to the repository"""
        # Copy package to pool
        deb_name = os.path.basename(deb_path)
        shutil.copy2(deb_path, self.pool_dir / deb_name)

    def _generate_release_file(self):
        """Generate the Release file with required fields"""
        release_path = self.dists_dir / "Release"
        
        # Get current time in exact format APT expects
        date = subprocess.check_output(
            ['date', '-u', '+%a, %d %b %Y %H:%M:%S UTC'],
            encoding='utf-8'
        ).strip()
        
        release_content = f"""Origin: {self.repo_name}
Label: {self.repo_name}
Suite: {self.codename}
Codename: {self.codename}
Date: {date}
Architectures: {' '.join(self.architectures)}
Components: main
Description: GitHub Release Repository for {self.repo_name}"""
        
        # Collect all hash entries first
        sections = []
        hash_commands = {
            'MD5Sum': 'md5sum',
            'SHA1': 'sha1sum',
            'SHA256': 'sha256sum'
        }
        
        for hash_name in ['MD5Sum', 'SHA1', 'SHA256']:
            entries = []
            for component in ['main']:
                for arch in sorted(self.architectures):
                    for filename in [
                        f"{component}/binary-{arch}/Packages",
                        f"{component}/binary-{arch}/Packages.gz",
                        f"{component}/binary-{arch}/Packages.xz"
                    ]:
                        filepath = self.dists_dir / filename
                        if filepath.exists():
                            size = filepath.stat().st_size
                            checksum = subprocess.check_output(
                                [hash_commands[hash_name], str(filepath)],
                                encoding='utf-8'
                            ).split()[0]
                            entries.append(f" {checksum} {size:12d} {filename}")
            
            if entries:
                sections.append(f"{hash_name}:\n" + "\n".join(entries))
        
        # Join all sections with single newlines
        if sections:
            release_content += "\n" + "\n".join(sections)
        
        release_content += "\nAcquire-By-Hash: yes\n"
        
        # Write Release file
        with open(release_path, 'w', encoding='utf-8') as f:
            f.write(release_content)
        
        # Generate InRelease (signed Release)
        subprocess.run([
            'gpg', '--default-key', 'repo@example.com',
            '--clearsign', '--armor', '--yes',
            '--output', str(self.dists_dir / 'InRelease'),
            str(release_path)
        ], check=True)
        
        # Generate Release.gpg (detached signature)
        subprocess.run([
            'gpg', '--default-key', 'repo@example.com',
            '--detach-sign', '--armor', '--yes',
            '--output', str(release_path) + '.gpg',
            str(release_path)
        ], check=True)

    def generate_metadata(self):
        """Generate repository metadata files"""
        for arch in self.architectures:
            packages_dir = self.dists_dir / "main" / f"binary-{arch}"
            packages_path = packages_dir / "Packages"
            
            print(f"Processing architecture: {arch}")
            # Generate Packages file using pydpkg
            with open(packages_path, 'w', encoding='utf-8') as f:
                package_count = 0
                # Scan pool directory for .deb files
                for deb_file in self.pool_dir.glob('*.deb'):
                    pkg = Dpkg(str(deb_file))
                    print(f"Found package: {pkg.package} ({pkg.architecture})")
                    
                    # Skip if architecture doesn't match (unless it's 'all')
                    if pkg.architecture != arch and pkg.architecture != 'all':
                        print(f"Skipping {deb_file.name} - wrong architecture")
                        continue
                    
                    package_count += 1
                    print(f"Adding package: {pkg.package} version {pkg.version}")
                    # Write package info in debian control file format
                    f.write(f"Package: {pkg.package}\n")
                    f.write(f"Version: {pkg.version}\n")
                    f.write(f"Architecture: {pkg.architecture}\n")
                    f.write(f"Maintainer: {pkg.maintainer}\n")
                    if pkg.depends:
                        f.write(f"Depends: {pkg.depends}\n")
                    f.write(f"Filename: pool/{deb_file.name}\n")
                    f.write(f"Size: {deb_file.stat().st_size}\n")
                    f.write(f"MD5sum: {pkg.md5}\n")
                    f.write(f"SHA1: {pkg.sha1}\n")
                    f.write(f"SHA256: {pkg.sha256}\n")
                    if pkg.section:
                        f.write(f"Section: {pkg.section}\n")
                    if pkg.description:
                        f.write(f"Description: {pkg.description}\n")
                    f.write("\n")
                
                print(f"Added {package_count} packages for {arch}")

            # Generate compressed versions
            # Gzip
            subprocess.run([
                "gzip", "-k", "-f", "-9",
                str(packages_path)
            ], check=True)
            
            # XZ
            subprocess.run([
                "xz", "-k", "-f", "-9",
                str(packages_path)
            ], check=True)

        # Generate Release files
        self._generate_release_file()

def process_repository(owner: str, repo: str, output_dir: str):
    """Process a GitHub repository and create APT repository"""
    releases = get_github_releases(owner, repo)
    repo_gen = AptRepoGenerator(
        output_dir=output_dir,
        repo_name=f"{owner}/{repo}",
        base_url="https://nplb.wastelandsystems.io"
    )
    repo_gen.init_repository()
    
    with tempfile.TemporaryDirectory() as temp_dir:
        for release in releases:
            for asset in release['assets']:
                temp_deb_path = os.path.join(temp_dir, asset['name'])
                try:
                    download_file(asset['download_url'], temp_deb_path)
                    package_info = extract_deb_info(temp_deb_path)
                    repo_gen.add_package(temp_deb_path)
                except Exception as e:
                    print(f"Error processing {asset['name']}: {str(e)}", file=sys.stderr)
                finally:
                    if os.path.exists(temp_deb_path):
                        os.remove(temp_deb_path)
    
    repo_gen.generate_metadata()

def main():
    if len(sys.argv) != 4:
        print("Usage: python main.py <owner> <repo> <output_dir>")
        sys.exit(1)
    
    owner, repo, output_dir = sys.argv[1], sys.argv[2], sys.argv[3]
    try:
        process_repository(owner, repo, output_dir)
    except Exception as e:
        print(f"Error: {str(e)}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
