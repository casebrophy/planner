import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fetchAllPages } from '@/services/fetchAllPages'
import { createMockService } from '../helpers/mockService'
import { makeQueryResult } from '../helpers/testFactories'

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
}

function makeItem(id: string, name = `Item ${id}`): TestItem {
  return { id, name }
}

let mockService: ReturnType<typeof createMockService<TestItem, NewTestItem, UpdateTestItem, TestFilter>>

beforeEach(() => {
  mockService = createMockService()
})

describe('fetchAllPages', () => {
  it('returns single page when first call returns less than pageSize', async () => {
    const items = [makeItem('1'), makeItem('2')]
    mockService.list.mockResolvedValue(makeQueryResult(items, 2))

    const result = await fetchAllPages(mockService, { pageSize: 100 })

    expect(result.items).toEqual(items)
    expect(result.total).toBe(2)
    expect(mockService.list).toHaveBeenCalledOnce()
  })

  it('fetches multiple pages and accumulates items', async () => {
    const page1 = Array.from({ length: 100 }, (_, i) => makeItem(`1-${i}`))
    const page2 = Array.from({ length: 100 }, (_, i) => makeItem(`2-${i}`))
    const page3 = Array.from({ length: 30 }, (_, i) => makeItem(`3-${i}`))

    mockService.list
      .mockResolvedValueOnce(makeQueryResult(page1, 230))
      .mockResolvedValueOnce(makeQueryResult(page2, 230))
      .mockResolvedValueOnce(makeQueryResult(page3, 230))

    const result = await fetchAllPages(mockService, { pageSize: 100 })

    expect(result.items).toHaveLength(230)
    expect(result.items).toEqual([...page1, ...page2, ...page3])
    expect(result.total).toBe(230)
    expect(mockService.list).toHaveBeenCalledTimes(3)
  })

  it('stops at exact boundary when second page returns empty', async () => {
    const page1 = Array.from({ length: 100 }, (_, i) => makeItem(`1-${i}`))
    mockService.list
      .mockResolvedValueOnce(makeQueryResult(page1, 100))
      .mockResolvedValueOnce(makeQueryResult([], 100))

    const result = await fetchAllPages(mockService, { pageSize: 100 })

    expect(result.items).toHaveLength(100)
    expect(result.total).toBe(100)
    expect(mockService.list).toHaveBeenCalledTimes(2)
  })

  it('warns when hitting maxPages without termination', async () => {
    const fullPage = Array.from({ length: 100 }, (_, i) => makeItem(`item-${i}`))
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    mockService.list.mockResolvedValue(makeQueryResult(fullPage, 1000))

    const result = await fetchAllPages(mockService, { pageSize: 100, maxPages: 3 })

    expect(result.items).toHaveLength(300)
    expect(warnSpy).toHaveBeenCalledOnce()
    expect(warnSpy.mock.calls[0]![0]).toContain('hit maxPages=3')
    expect(mockService.list).toHaveBeenCalledTimes(3)

    warnSpy.mockRestore()
  })

  it('passes filter to service.list', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const filter = { status: 'active' }
    await fetchAllPages(mockService, { filter, pageSize: 100 })

    expect(mockService.list).toHaveBeenCalledWith(
      expect.objectContaining({ filter, rows: 100 }),
    )
  })

  it('passes orderBy to service.list', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    await fetchAllPages(mockService, { orderBy: 'name', pageSize: 100 })

    expect(mockService.list).toHaveBeenCalledWith(
      expect.objectContaining({ orderBy: 'name', rows: 100 }),
    )
  })

  it('uses default pageSize of 100', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    await fetchAllPages(mockService)

    expect(mockService.list).toHaveBeenCalledWith(
      expect.objectContaining({ rows: 100 }),
    )
  })
})
