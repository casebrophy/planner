import type { CRUDService } from './createCRUDService'
import type { QueryResult } from '@/types'

export interface FetchAllPagesOptions<TFilter> {
  filter?: TFilter
  orderBy?: string
  pageSize?: number
  maxPages?: number
}

export interface FetchAllPagesResult<T> {
  items: T[]
  total: number
}

export async function fetchAllPages<T, TFilter>(
  service: CRUDService<T, unknown, unknown, TFilter>,
  opts: FetchAllPagesOptions<TFilter> = {},
): Promise<FetchAllPagesResult<T>> {
  const { filter, orderBy, pageSize = 100, maxPages = 50 } = opts

  const accumulator: T[] = []
  let lastResult: QueryResult<T> | null = null

  for (let page = 1; page <= maxPages; page++) {
    lastResult = await service.list({
      page,
      rows: pageSize,
      orderBy,
      filter,
    })

    accumulator.push(...lastResult.items)

    if (lastResult.items.length < pageSize) {
      break
    }
  }

  if (lastResult && lastResult.items.length === pageSize) {
    const filterStr = filter ? ` (filter: ${JSON.stringify(filter)})` : ''
    const orderStr = orderBy ? ` (orderBy: ${orderBy})` : ''
    console.warn(
      `fetchAllPages: hit maxPages=${maxPages}${filterStr}${orderStr} without reaching end of results`,
    )
  }

  return {
    items: accumulator,
    total: lastResult?.total ?? accumulator.length,
  }
}
