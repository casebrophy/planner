export interface QueryResult<T> {
  items: T[]
  total: number
  page: number
  rowsPerPage: number
}

export interface ListParams {
  page?: number
  rows?: number
  orderBy?: string
}
