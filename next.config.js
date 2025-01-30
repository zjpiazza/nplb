/** @type {import('next').NextConfig} */
const nextConfig = {
  rewrites: async () => [
    {
      source: '/dists/:path*',
      destination: '/api/dists/:path*',
    },
    {
      source: '/pool/:path*',
      destination: '/api/pool/:path*',
    }
  ]
}

module.exports = nextConfig