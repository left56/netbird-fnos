import { apiURL } from './gateway'

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiURL(path), init)
  const payload = await response.json()
  if (!response.ok) throw new Error(payload.data?.message || '请求失败')
  return payload.data as T
}

export function json(body: unknown): RequestInit { return { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } }
