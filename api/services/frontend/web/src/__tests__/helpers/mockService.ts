import { vi } from 'vitest'
import type { CRUDService } from '@/services/createCRUDService'

export function createMockService<T, TNew, TUpdate, TFilter>(): CRUDService<T, TNew, TUpdate, TFilter> & {
  list: ReturnType<typeof vi.fn>
  getById: ReturnType<typeof vi.fn>
  create: ReturnType<typeof vi.fn>
  update: ReturnType<typeof vi.fn>
  delete: ReturnType<typeof vi.fn>
} {
  return {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  }
}
