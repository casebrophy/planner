import { describe, it, expect, vi, beforeEach } from 'vitest'
import { taskService } from '@/services/taskService'

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(data),
  })
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('taskService', () => {
  describe('list', () => {
    it('fetches tasks with default params', async () => {
      const response = { items: [], total: 0, page: 1, rowsPerPage: 20 }
      mockFetch.mockReturnValue(jsonResponse(response))

      const result = await taskService.list()
      expect(result).toEqual(response)
      expect(mockFetch).toHaveBeenCalledOnce()

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tasks')
    })

    it('passes filter params', async () => {
      const response = { items: [], total: 0, page: 1, rowsPerPage: 20 }
      mockFetch.mockReturnValue(jsonResponse(response))

      await taskService.list({
        page: 2,
        rows: 10,
        filter: { status: 'todo', priority: 'high' },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('page=2')
      expect(url).toContain('rows=10')
      expect(url).toContain('status=todo')
      expect(url).toContain('priority=high')
    })
  })

  describe('getById', () => {
    it('fetches a single task', async () => {
      const task = { id: 'abc-123', title: 'Test', status: 'todo' }
      mockFetch.mockReturnValue(jsonResponse(task))

      const result = await taskService.getById('abc-123')
      expect(result).toEqual(task)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tasks/abc-123')
    })
  })

  describe('create', () => {
    it('posts a new task', async () => {
      const newTask = { title: 'New', description: '', priority: 'medium' as const, energy: 'medium' as const }
      const created = { ...newTask, id: 'new-id', status: 'todo' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await taskService.create(newTask)
      expect(result).toEqual(created)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
      expect(JSON.parse(options.body as string)).toEqual(newTask)
    })
  })

  describe('update', () => {
    it('puts task updates', async () => {
      const update = { title: 'Updated' }
      const updated = { id: 'abc', title: 'Updated', status: 'todo' }
      mockFetch.mockReturnValue(jsonResponse(updated))

      const result = await taskService.update('abc', update)
      expect(result).toEqual(updated)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('PUT')
    })
  })

  describe('delete', () => {
    it('deletes a task', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await taskService.delete('abc')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/abc')
      expect(options.method).toBe('DELETE')
    })
  })
})
