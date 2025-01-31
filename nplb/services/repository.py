from pathlib import Path
from typing import List
import gnupg
import os
from ..core.models import DebInfo
from .debian import DebianService
from .storage import R2StorageService
from pydpkg import Dpkg
import shutil
import subprocess
import hashlib

class RepositoryService:
    def __init__(
        self,
        output_dir: str,
        repo_name: str,
        base_url: str,
        storage: R2StorageService,
        codename: str = "stable"
    ):
        self.output_dir = Path(output_dir)
        self.repo_name = repo_name
        self.base_url = base_url
        self.codename = codename
        self.architectures = ["amd64", "arm64"]
        self.pool_dir = self.output_dir / "pool"
        self.dists_dir = self.output_dir / "dists" / self.codename
        self.debian_service = DebianService()
        self.storage = storage
        
        # Initialize GPG
        self.gpg_home = Path("keys")
        self.gpg = gnupg.GPG(gnupghome=str(self.gpg_home))
        self.gpg.encoding = 'utf-8'

    def init_repository(self):
        """Initialize repository directory structure"""
        for arch in self.architectures:
            (self.dists_dir / "main" / f"binary-{arch}").mkdir(parents=True, exist_ok=True)
        self.pool_dir.mkdir(parents=True, exist_ok=True)
        self._setup_gpg()

    def add_package(self, deb_path: str):
        """Add a package to the repository"""
        # Copy package to pool
        deb_name = os.path.basename(deb_path)
        shutil.copy2(deb_path, self.pool_dir / deb_name)

    def generate_metadata(self):
        """Generate repository metadata files"""
        # Keep track of architectures that actually have packages
        active_architectures = set()
        
        for arch in self.architectures:
            packages_dir = self.dists_dir / "main" / f"binary-{arch}"
            packages_path = packages_dir / "Packages"
            
            print(f"Processing architecture: {arch}")
            
            # Count packages before writing file
            package_count = 0
            for deb_file in self.pool_dir.glob('*.deb'):
                pkg = Dpkg(str(deb_file))
                if pkg.architecture == arch or pkg.architecture == 'all':
                    package_count += 1
            
            # Skip this architecture if no packages found
            if package_count == 0:
                print(f"No packages found for architecture {arch}, skipping")
                continue
            
            active_architectures.add(arch)
            
            with open(packages_path, 'w', encoding='utf-8') as f:
                # Scan pool directory for .deb files
                for deb_file in self.pool_dir.glob('*.deb'):
                    pkg = Dpkg(str(deb_file))
                    print(f"Found package: {pkg.package} ({pkg.architecture})")
                    
                    # Skip if architecture doesn't match (unless it's 'all')
                    if pkg.architecture != arch and pkg.architecture != 'all':
                        print(f"Skipping {deb_file.name} - wrong architecture")
                        continue
                    
                    print(f"Adding package: {pkg.package} version {pkg.version}")
                    
                    # Calculate the R2 path for the package
                    r2_path = f"repos/{self.repo_name}/pool/{deb_file.name}"
                    
                    # Write package info in debian control file format
                    f.write(f"Package: {pkg.package}\n")
                    f.write(f"Version: {pkg.version}\n")
                    f.write(f"Architecture: {pkg.architecture}\n")
                    f.write(f"Maintainer: {pkg.maintainer}\n")
                    if pkg.depends:
                        f.write(f"Depends: {pkg.depends}\n")
                    f.write(f"Filename: {r2_path}\n")  # Use R2 path
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

            # Generate compressed versions only if we wrote a Packages file
            subprocess.run(["gzip", "-k", "-f", "-9", str(packages_path)], check=True)
            subprocess.run(["xz", "-k", "-f", "-9", str(packages_path)], check=True)

        # Update self.architectures to only include architectures with packages
        self.architectures = sorted(active_architectures)
        
        # Generate Release files with updated architecture list
        self._generate_release_file()

    def _setup_gpg(self):
        """Setup GPG signing key"""
        self.gpg_home.mkdir(exist_ok=True)
        
        # Check if we already have a key
        existing_keys = self.gpg.list_keys(secret=True)
        if not existing_keys:
            # Generate a new key
            key_input = self.gpg.gen_key_input(
                key_type="RSA",
                key_length=4096,
                name_real="APT Repository",
                name_email="repo@example.com",
                expire_date=0,
                no_protection=True
            )
            key = self.gpg.gen_key(key_input)
            
            # Export public key to repository
            public_key = self.gpg.export_keys(key.fingerprint)
            with open(self.output_dir / "key.gpg", 'w') as f:
                f.write(public_key)
        else:
            # Export existing public key to repository
            public_key = self.gpg.export_keys(existing_keys[0]['fingerprint'])
            with open(self.output_dir / "key.gpg", 'w') as f:
                f.write(public_key)

    def _generate_release_file(self):
        """Generate the Release file with required fields"""
        release_path = self.dists_dir / "Release"
        
        # Get current time in exact format APT expects
        from datetime import datetime, timezone
        date = datetime.now(timezone.utc).strftime("%a, %d %b %Y %H:%M:%S UTC")
        
        release_content = f"""Origin: {self.repo_name}
Label: {self.repo_name}
Suite: {self.codename}
Codename: {self.codename}
Date: {date}
Architectures: {' '.join(self.architectures)}
Components: main
Description: GitHub Release Repository for {self.repo_name}
Acquire-By-Hash: yes"""
        
        # Collect all hash entries
        sections = []
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
                            with open(filepath, 'rb') as f:
                                content = f.read()
                                if hash_name == 'MD5Sum':
                                    checksum = hashlib.md5(content).hexdigest()
                                elif hash_name == 'SHA1':
                                    checksum = hashlib.sha1(content).hexdigest()
                                else:  # SHA256
                                    checksum = hashlib.sha256(content).hexdigest()
                            entries.append(f" {checksum} {size:12d} {filename}")
            
            if entries:
                sections.append(f"\n{hash_name}:\n" + "\n".join(entries))
        
        release_content += "".join(sections)
        
        # Write Release file
        with open(release_path, 'w', encoding='utf-8') as f:
            f.write(release_content)
        
        # Generate InRelease (clearsigned Release)
        with open(release_path, 'rb') as f:
            signed_data = self.gpg.sign(
                f.read(),
                keyid='repo@example.com',
                clearsign=True,
                detach=False
            )
            with open(self.dists_dir / 'InRelease', 'w') as sf:
                sf.write(str(signed_data))
        
        # Generate Release.gpg (detached signature)
        with open(release_path, 'rb') as f:
            detached_sig = self.gpg.sign(
                f.read(),
                keyid='repo@example.com',
                detach=True
            )
            with open(str(release_path) + '.gpg', 'w') as sf:
                sf.write(str(detached_sig))

    def publish(self):
        """Upload the repository to R2"""
        # First, clean up any existing content
        self.storage.delete_prefix(f"repos/{self.repo_name}")
        
        # Upload the entire repository
        self.storage.upload_directory(
            self.output_dir,
            prefix=f"repos/{self.repo_name}"
        ) 