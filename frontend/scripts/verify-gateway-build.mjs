import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const gatewayPrefix = '/app/netbird-fnos'
const pageURL = new URL('https://fnos.example/app/netbird-fnos')
const dist = join(import.meta.dirname, '..', 'dist')
const index = readFileSync(join(dist, 'index.html'), 'utf8')
const urls = [...index.matchAll(/(?:src|href)="([^"]+)"/g)].map((match) => match[1])

if (urls.length === 0) throw new Error('dist/index.html has no script or stylesheet URLs')
for (const url of urls) {
  if (url.startsWith('./assets/') || url.startsWith('/assets/')) throw new Error(`unsafe asset URL: ${url}`)
  if (!url.startsWith(`${gatewayPrefix}/assets/`)) throw new Error(`asset URL does not use gateway prefix: ${url}`)
  const resolved = new URL(url, pageURL)
  if (!resolved.pathname.startsWith(`${gatewayPrefix}/assets/`)) throw new Error(`asset URL resolves outside gateway prefix: ${resolved.pathname}`)
}

const assets = readdirSync(join(dist, 'assets'))
if (!assets.some((asset) => asset.endsWith('.js')) || !assets.some((asset) => asset.endsWith('.css'))) {
  throw new Error('dist/assets must contain JavaScript and CSS output')
}

const gatewaySource = readFileSync(join(import.meta.dirname, '..', 'src', 'gateway.ts'), 'utf8')
if (!gatewaySource.includes(`GATEWAY_PREFIX = '${gatewayPrefix}'`) || !gatewaySource.includes('return pathname === GATEWAY_PREFIX')) {
  throw new Error('frontend API gateway prefix helper is missing or changed')
}

console.log(`verified ${urls.length} gateway-prefixed asset URL(s); ${gatewayPrefix}/api/... is selected at the gateway entry`)
