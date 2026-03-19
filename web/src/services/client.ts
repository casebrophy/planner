import { ApiError, ApiNetworkError, ApiNotFoundError, ApiValidationError } from '@/types/errors'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
const API_KEY = import.meta.env.VITE_API_KEY || ''

interface RequestOptions {
  method?: string
  body?: unknown
  params?: Record<string, string | number | undefined>
}

function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const url = new URL(`${BASE_URL}${path}`, window.location.origin)
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== '') {
        url.searchParams.set(key, String(value))
      }
    }
  }
  return url.toString()
}

async function mapError(response: Response): Promise<never> {
  let body: Record<string, unknown> = {}
  try {
    body = (await response.json()) as Record<string, unknown>
  } catch {
    // ignore parse errors
  }

  const message = (body.error as string) || response.statusText

  if (response.status === 404) {
    throw new ApiNotFoundError(message)
  }

  if (response.status === 400) {
    const fields = (body.fields as Record<string, string>) || {}
    throw new ApiValidationError(message, fields)
  }

  throw new ApiError(message, response.status, (body.code as string) || 'unknown')
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, params } = options
  const url = buildUrl(path, params)

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (API_KEY) {
    headers['X-API-Key'] = API_KEY
  }

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiNetworkError('Failed to connect to API')
  }

  if (!response.ok) {
    return mapError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export const client = { request }
