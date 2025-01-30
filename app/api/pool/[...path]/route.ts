// pages/api/pool/[...path].ts
import { NextApiRequest, NextApiResponse } from 'next'
import { createClient } from '@/utils/supabase/client'

const supabase = createClient()

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const { path } = req.query
  const pathStr = Array.isArray(path) ? path.join('/') : path

  console.log('Handling pool path:', pathStr) // Debug logging

  try {
    // Look up package by its path
    const { data: pkg } = await supabase
      .from('packages')
      .select('storage_url')
      .eq('path', pathStr)
      .single()

    if (!pkg) {
      console.log('Package not found:', pathStr) // Debug logging
      res.status(404).end()
      return
    }

    // Redirect to storage URL
    res.redirect(307, pkg.storage_url)
  } catch (error) {
    console.error('Error handling package download:', error)
    res.status(500).json({ error: 'Internal server error' })
  }
}