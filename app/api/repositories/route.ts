import { NextResponse } from 'next/server'
import { PrismaClient } from '@prisma/client'
import { Octokit } from 'octokit'
import { findDebReleases, getDebFileInfo } from '@/lib/github'
import { createHash } from 'crypto'
import { Buffer } from 'buffer'

const prisma = new PrismaClient()
const octokit = new Octokit({
  auth: process.env.GITHUB_TOKEN
})

export async function POST(request: Request) {
  try {
    const { githubUrl } = await request.json()

    // Basic validation
    if (!githubUrl) {
      return NextResponse.json({ error: 'GitHub URL is required' }, { status: 400 })
    }

    // Extract owner and repo from GitHub URL
    const urlMatch = githubUrl.match(/github\.com\/([^\/]+)\/([^\/]+)/)
    if (!urlMatch) {
      return NextResponse.json({ error: 'Invalid GitHub URL format' }, { status: 400 })
    }

    const [, owner, repo] = urlMatch

    // Verify repository exists and is accessible
    try {
      await octokit.rest.repos.get({
        owner,
        repo,
      })
    } catch (error) {
      return NextResponse.json({ 
        error: 'Repository not found or not accessible' 
      }, { status: 404 })
    }

    console.log(`Processing repository: ${owner}/${repo}`)
    const releases = await findDebReleases(owner, repo)
    console.log(`Found ${releases.length} releases with .deb files`)

    if (!releases.length) {
      return NextResponse.json({ 
        error: 'No .deb packages found in repository' 
      }, { status: 400 })
    }
    // Create or update repository record
    const repository = await prisma.repository.upsert({
      where: {
        owner_name: {  // Using the @@unique([owner, name]) constraint
          owner,
          name: repo
        }
      },
      update: {
        githubOwner: owner,
        githubRepo: repo,
      },
      create: {
        owner,
        name: repo,
        githubOwner: owner,
        githubRepo: repo,
      },
    })

    const processedPackages = []
    const errors = []

    // Process each .deb file
    for (const release of releases) {
      console.log(`Processing release: ${release.tagName}`)
      
      for (const asset of release.debAssets) {
        try {
          console.log(`Processing asset: ${asset.name}`)
          const { buffer, metadata, downloadUrl } = await getDebFileInfo(owner, repo, asset.url)

          // Calculate hashes
          const md5 = createHash('md5').update(buffer.toString()).digest('hex')
          const sha1 = createHash('sha1').update(buffer.toString()).digest('hex')
          const sha256 = createHash('sha256').update(buffer.toString()).digest('hex')

          // Create or update package record
          const pkg = await prisma.package.upsert({
            where: {
              name_version_architecture: {
                name: metadata.control.package,
                version: metadata.control.version,
                architecture: metadata.control.architecture,
              }
            },
            update: {
              description: metadata.control.description,
              maintainer: metadata.control.maintainer,
              depends: metadata.control.depends,
              path: `pool/main/${metadata.control.package[0]}/${metadata.control.package}/${asset.name}`,
              storageUrl: downloadUrl,
              size: asset.size,
              md5,
              sha1,
              sha256,
              repositoryId: repository.id
            },
            create: {
              name: metadata.control.package,
              version: metadata.control.version,
              architecture: metadata.control.architecture,
              description: metadata.control.description,
              maintainer: metadata.control.maintainer,
              depends: metadata.control.depends,
              path: `pool/main/${metadata.control.package[0]}/${metadata.control.package}/${asset.name}`,
              storageUrl: downloadUrl,
              size: asset.size,
              md5,
              sha1,
              sha256,
              repositoryId: repository.id
            }
          })

          processedPackages.push({
            name: pkg.name,
            version: pkg.version,
            size: pkg.size
          })

        } catch (err) {
          console.error(`Error processing ${asset.name}:`, err)
          errors.push({
            asset: asset.name,
            error: err instanceof Error ? err.message : 'Unknown error'
          })
        }
      }
    }

    return NextResponse.json({ 
      success: true, 
      repository,
      processedReleases: releases.length,
      processedPackages,
      errors: errors.length ? errors : undefined,
      totalPackages: processedPackages.length
    })

  } catch (error) {
    console.error('Error creating repository:', error)
    return NextResponse.json(
      { error: 'Failed to create repository' },
      { status: 500 }
    )
  }
}

