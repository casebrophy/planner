import { describe, it, expect, beforeEach } from 'vitest'
import { createCRUDService } from '@/services/createCRUDService'
import { setupMockFetch } from '../helpers/mockFetch'
import { ApiNotFoundError, ApiValidationError, ApiNetworkError } from '@/types'

interface TestItem {
  id: string
  name: string
}
interface NewTestItem {
  name: string
}
interface UpdateTestItem {
  name?: string
}
interface TestFilter {
  status?: string
  category?: string
}

const { mockFetch, jsonResponse, noContentResponse, networkError } = setupMockFetch()

function createTestService(mapFilter?: (f: TestFilter) => Record<string, string | number | undefined>) {
  return createCRUDService<TestItem, NewTestItem, UpdateTestItem, TestFilter>({
    basePath: '/api/v1/tests',
    mapFilter,
  })
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('createCRUDService', () => {
  describe('list', () => {
    it('fetches with default params', async () => {
      const data = { items: [], total: 0, page: 1, rowsPerPage: 20 }
      mockFetch.mockReturnValue(jsonResponse(data))

      const service = createTestService()
      const result = await service.list()

      expect(result).toEqual(data)
      expect(mockFetch).toHaveBeenCalledOnce()
      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tests')
    })

    it('passes pagination and ordering params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 2, rowsPerPage: 10 }))

      const service = createTestService()
      await service.list({ page: 2, rows: 10, orderBy: 'name' })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('page=2')
      expect(url).toContain('rows=10')
      expect(url).toContain('orderBy=name')
    })

    it('applies mapFilter to convert domain filter to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      const service = createTestService((f) => ({
        status: f.status,
        cat: f.category,
      }))
      await service.list({ filter: { status: 'active', category: 'work' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).toContain('cat=work')
    })

    it('omits undefined filter values from query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      const service = createTestService((f) => ({
        status: f.status,
        cat: f.category,
      }))
      await service.list({ filter: { status: 'active' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).not.toContain('cat=')
    })
  })

  describe('getById', () => {
    it('fetches a single item by ID', async () => {
      const item = { id: 'abc', name: 'Test' }
      mockFetch.mockReturnValue(jsonResponse(item))

      const service = createTestService()
      const result = await service.getById('abc')

      expect(result).toEqual(item)
      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tests/abc')
    })
  })

  describe('create', () => {
    it('posts a new item', async () => {
      const newItem = { name: 'New' }
      const created = { id: 'new-1', name: 'New' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const service = createTestService()
      const result = await service.create(newItem)

      expect(result).toEqual(created)
      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
      expect(JSON.parse(options.body as string)).toEqual(newItem)
    })
  })

  describe('update', () => {
    it('puts updates to an item', async () => {
      const update = { name: 'Updated' }
      const updated = { id: 'abc', name: 'Updated' }
      mockFetch.mockReturnValue(jsonResponse(updated))

      const service = createTestService()
      const result = await service.update('abc', update)

      expect(result).toEqual(updated)
      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tests/abc')
      expect(options.method).toBe('PUT')
    })
  })

  describe('delete', () => {
    it('deletes an item', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      const service = createTestService()
      await service.delete('abc')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tests/abc')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('error handling', () => {
    it('throws ApiNotFoundError on 404', async () => {
      mockFetch.mockReturnValue(jsonResponse({ error: 'not found' }, 404))

      const service = createTestService()
      await expect(service.getById('missing')).rejects.toThrow(ApiNotFoundError)
    })

    it('throws ApiValidationError on 400', async () => {
      mockFetch.mockReturnValue(
        jsonResponse({ error: 'invalid', fields: { name: 'required' } }, 400),
      )

      const service = createTestService()
      await expect(service.create({ name: '' })).rejects.toThrow(ApiValidationError)
    })

    it('throws ApiNetworkError on fetch failure', async () => {
      mockFetch.mockReturnValue(networkError())

      const service = createTestService()
      await expect(service.list()).rejects.toThrow(ApiNetworkError)
    })
  })
})
