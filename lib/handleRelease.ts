// pages/api/dists/[...path].ts
import { NextApiRequest, NextApiResponse } from 'next'
import { createHash } from 'crypto'
import { gzip } from 'zlib'
import { promisify } from 'util'
import { PrismaClient } from '@prisma/client'
import { findDebReleases, getDebFileInfo } from './github'

const gzipAsync = promisify(gzip)
const prisma = new PrismaClient()

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const { path } = req.query
  const pathStr = Array.isArray(path) ? path.join('/') : path

  console.log('Handling path:', pathStr)

  if (pathStr === 'stable/Release') {
    await handleRelease(res)
  } else {
    res.status(404).end()
  }
}

async function handleRelease(res: NextApiResponse) {
  try {
    // Get all packages using Prisma
    const packages = await prisma.package.findMany()

    if (packages.length === 0) {
      console.log('No packages found')
      // Still generate a valid Release file even with no packages
    }

    // Generate Packages file content
    const packagesContent = packages.map(pkg => `Package: ${pkg.name}
Version: ${pkg.version}
Architecture: ${pkg.architecture}
Maintainer: ${pkg.maintainer || 'Unknown'}
Installed-Size: ${Math.ceil(Number(pkg.size) / 1024)}
Depends: ${pkg.depends || ''}
Filename: ${pkg.path}
Size: ${pkg.size}
MD5sum: ${pkg.md5}
SHA1: ${pkg.sha1}
SHA256: ${pkg.sha256}
Description: ${pkg.description || 'No description available'}
`).join('\n')

    // Compress it as we'll need the hashes of the compressed file
    const packagesGz = await gzipAsync(Buffer.from(packagesContent))

    // Calculate hashes of the Packages.gz file
    const hashes = {
      md5: createHash('md5').update(packagesGz).digest('hex'),
      sha1: createHash('sha1').update(packagesGz).digest('hex'),
      sha256: createHash('sha256').update(packagesGz).digest('hex')
    }

    // Generate Release file content
    const releaseContent = `Origin: NoPackageLeftBehind
Label: nplb
Suite: stable
Codename: stable
Version: 1.0
Date: ${new Date().toUTCString()}
Architectures: amd64
Components: main
Description: NoPackageLeftBehind - GitHub .deb Package Repository
MD5Sum:
 ${hashes.md5} ${packagesGz.length} main/binary-amd64/Packages.gz
SHA1:
 ${hashes.sha1} ${packagesGz.length} main/binary-amd64/Packages.gz
SHA256:
 ${hashes.sha256} ${packagesGz.length} main/binary-amd64/Packages.gz`

    // Cache the Release file content in database
    await prisma.repositoryMetadata.upsert({
      where: {
        type: 'Release'
      },
      update: {
        content: releaseContent,
      },
      create: {
        type: 'Release',
        content: releaseContent,
      }
    })

    // Send the response
    res.setHeader('Content-Type', 'text/plain')
    res.send(releaseContent)

  } catch (error) {
    console.error('Error generating Release file:', error)
    res.status(500).json({ error: 'Internal server error' })
  }
}

export async function handleReleaseForRepository(owner: string, repo: string) {
  // Find all releases with .deb files
  const releases = await findDebReleases(owner, repo)
  
  if (releases.length === 0) {
    throw new Error('No .deb packages found in repository releases')
  }

  const debFiles = []
  
  // Process each release
  for (const release of releases) {
    for (const asset of release.debAssets) {
      const fileInfo = await getDebFileInfo(owner, repo, asset.url)
      
      debFiles.push({
        name: asset.name,
        size: asset.size,
        downloadUrl: fileInfo.downloadUrl,
        release: release.tagName
      })
    }
  }

  return {
    releases: releases.length,
    debFiles: debFiles
  }
}