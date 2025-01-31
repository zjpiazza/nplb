import { Octokit } from 'octokit'
import { Readable } from 'stream'
import { parseDebFile } from './debParser'

const octokit = new Octokit({
  auth: process.env.GITHUB_TOKEN
})

interface DebAsset {
  name: string
  url: string
  size: number
  downloadUrl: string
}

interface ReleaseInfo {
  tagName: string
  name: string
  debAssets: DebAsset[]
}

export async function findDebReleases(owner: string, repo: string): Promise<ReleaseInfo[]> {
  const releases = await octokit.rest.repos.listReleases({
    owner,
    repo,
  })

  return releases.data.map(release => {
    const debAssets = release.assets
      .filter(asset => asset.name.endsWith('.deb'))
      .map(asset => ({
        name: asset.name,
        url: asset.url,
        size: asset.size,
        downloadUrl: asset.browser_download_url
      }))

    return {
      tagName: release.tag_name,
      name: release.name || release.tag_name,
      debAssets
    }
  }).filter(release => release.debAssets.length > 0)
}

export async function getDebFileInfo(owner: string, repo: string, assetUrl: string) {
  console.log(`Fetching .deb file from: ${assetUrl}`)
  
  // Get asset metadata and content using Octokit
  const response = await octokit.request('GET ' + assetUrl, {
    headers: {
      'Accept': 'application/octet-stream'
    }
  })

  // Log file size for monitoring
  const contentLength = response.headers['content-length']
  console.log(`File size: ${(Number(contentLength) / 1024 / 1024).toFixed(2)}MB`)

  // Convert response data to buffer/stream for parsing
  const buffer = Buffer.from(await response.data)
  const stream = Readable.from(buffer)

  // Parse the .deb file
  console.log('Parsing .deb file metadata...')
  const debInfo = await parseDebFile(buffer)
  console.log('Package:', debInfo.control.package, 'Version:', debInfo.control.version)

  return {
    downloadUrl: response.url,
    headers: response.headers,
    buffer,
    metadata: debInfo
  }
} 