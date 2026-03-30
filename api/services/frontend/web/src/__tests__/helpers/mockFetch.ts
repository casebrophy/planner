import { vi } from 'vitest'

export function setupMockFetch() {
  const mockFetch = vi.fn()
  vi.stubGlobal('fetch', mockFetch)

  function jsonResponse(data: unknown, status = 200) {
    return Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 404 ? 'Not Found' : status === 400 ? 'Bad Request' : 'OK',
      json: () => Promise.resolve(data),
    })
  }

  function noContentResponse() {
    return Promise.resolve({
      ok: true,
      status: 204,
      statusText: 'No Content',
      json: () => Promise.resolve(undefined),
    })
  }

  function networkError() {
    return Promise.reject(new TypeError('Failed to fetch'))
  }

  return { mockFetch, jsonResponse, noContentResponse, networkError }
}
