export const GATEWAY_PREFIX = '/app/netbird-fnos'

export function getApiBase(pathname = window.location.pathname): string {
  return pathname === GATEWAY_PREFIX || pathname.startsWith(`${GATEWAY_PREFIX}/`) ? GATEWAY_PREFIX : ''
}

export function apiURL(path: string, pathname?: string): string {
  return `${getApiBase(pathname)}${path.startsWith('/') ? path : `/${path}`}`
}
