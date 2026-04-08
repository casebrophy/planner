import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import { entityLinkService } from '@/services/entityLinkService'
import { makeEntityLink } from '../helpers/testFactories'
import type { NewEntityLink } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/entityLinkService', () => ({
  entityLinkService: {
    listByEntity: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
  },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('entityLinkStore', () => {
  describe('fetchLinks', () => {
    it('calls listByEntity with correct args and caches the result', async () => {
      const link = makeEntityLink({ sourceType: 'task', sourceId: 'task-1', targetType: 'note', targetId: 'note-1' })
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([link])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')

      expect(entityLinkService.listByEntity).toHaveBeenCalledWith('task', 'task-1')
      expect(store.linksByEntity['task:task-1']).toEqual([link])
    })

    it('does not call service again when cache already exists', async () => {
      const link = makeEntityLink()
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([link])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')
      await store.fetchLinks('task', 'task-1')

      expect(entityLinkService.listByEntity).toHaveBeenCalledTimes(1)
    })

    it('re-fetches when force=true even if cache exists', async () => {
      const link = makeEntityLink()
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([link])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')
      await store.fetchLinks('task', 'task-1', true)

      expect(entityLinkService.listByEntity).toHaveBeenCalledTimes(2)
    })

    it('sets loading to false after fetch completes', async () => {
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')

      expect(store.loading).toBe(false)
    })

    it('sets loading to false and shows toast on service failure', async () => {
      vi.mocked(entityLinkService.listByEntity).mockRejectedValue(new Error('Network error'))

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-2')

      expect(store.loading).toBe(false)
      // Does not throw; error is swallowed and a toast is shown
      expect(store.linksByEntity['task:task-2']).toBeUndefined()
    })
  })

  describe('getLinks', () => {
    it('returns cached links for a given entity', async () => {
      const link = makeEntityLink({ sourceType: 'task', sourceId: 'task-1' })
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([link])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')

      expect(store.getLinks('task', 'task-1')).toEqual([link])
    })

    it('returns an empty array when no cache entry exists', () => {
      const store = useEntityLinkStore()
      expect(store.getLinks('task', 'unknown')).toEqual([])
    })
  })

  describe('createLink', () => {
    it('calls create service and invalidates both side caches', async () => {
      const newLink: NewEntityLink = {
        sourceType: 'task',
        sourceId: 'task-1',
        targetType: 'note',
        targetId: 'note-1',
      }
      const created = makeEntityLink({ ...newLink, kind: 'manual' })
      vi.mocked(entityLinkService.create).mockResolvedValue(created)
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([])

      const store = useEntityLinkStore()
      // Populate both side caches first
      await store.fetchLinks('task', 'task-1')
      await store.fetchLinks('note', 'note-1')
      expect(store.linksByEntity['task:task-1']).toBeDefined()
      expect(store.linksByEntity['note:note-1']).toBeDefined()

      const result = await store.createLink(newLink)

      expect(entityLinkService.create).toHaveBeenCalledWith(newLink)
      expect(result).toEqual(created)
      // Both cache entries should be invalidated
      expect(store.linksByEntity['task:task-1']).toBeUndefined()
      expect(store.linksByEntity['note:note-1']).toBeUndefined()
    })

    it('returns null and shows toast on service failure', async () => {
      vi.mocked(entityLinkService.create).mockRejectedValue(new Error('Create failed'))

      const store = useEntityLinkStore()
      const result = await store.createLink({
        sourceType: 'task',
        sourceId: 'task-1',
        targetType: 'note',
        targetId: 'note-1',
      })

      expect(result).toBeNull()
    })
  })

  describe('deleteLink', () => {
    it('calls delete service with the link id and invalidates both side caches', async () => {
      const link = makeEntityLink({
        id: 'link-abc',
        sourceType: 'task',
        sourceId: 'task-1',
        targetType: 'note',
        targetId: 'note-1',
      })
      vi.mocked(entityLinkService.delete).mockResolvedValue(undefined)
      vi.mocked(entityLinkService.listByEntity).mockResolvedValue([link])

      const store = useEntityLinkStore()
      await store.fetchLinks('task', 'task-1')
      await store.fetchLinks('note', 'note-1')
      expect(store.linksByEntity['task:task-1']).toBeDefined()
      expect(store.linksByEntity['note:note-1']).toBeDefined()

      await store.deleteLink(link)

      expect(entityLinkService.delete).toHaveBeenCalledWith('link-abc')
      expect(store.linksByEntity['task:task-1']).toBeUndefined()
      expect(store.linksByEntity['note:note-1']).toBeUndefined()
    })

    it('shows toast and does not throw on service failure', async () => {
      vi.mocked(entityLinkService.delete).mockRejectedValue(new Error('Delete failed'))

      const store = useEntityLinkStore()
      const link = makeEntityLink({ id: 'link-xyz' })

      await expect(store.deleteLink(link)).resolves.toBeUndefined()
    })
  })

  describe('initial state', () => {
    it('starts with an empty linksByEntity map', () => {
      const store = useEntityLinkStore()
      expect(store.linksByEntity).toEqual({})
    })

    it('starts with loading = false', () => {
      const store = useEntityLinkStore()
      expect(store.loading).toBe(false)
    })
  })
})
