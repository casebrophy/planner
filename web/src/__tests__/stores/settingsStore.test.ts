import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsStore } from '@/stores/settingsStore'
import { nextTick } from 'vue'

const STORAGE_KEY = 'planner-settings'

const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }
})()
Object.defineProperty(window, 'localStorage', { value: localStorageMock })

describe('settingsStore', () => {
  beforeEach(() => {
    localStorageMock.clear()
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('initializes with defaults', () => {
    const store = useSettingsStore()
    expect(store.apiBaseUrl).toBe('')
    expect(store.pollingIntervalMs).toBe(60000)
    expect(store.rowsPerPage).toBe(20)
    expect(store.sidebarCollapsed).toBe(false)
  })

  it('persists changes to localStorage', async () => {
    const store = useSettingsStore()
    store.rowsPerPage = 50
    await nextTick()

    expect(localStorageMock.setItem).toHaveBeenCalledWith(
      STORAGE_KEY,
      expect.stringContaining('"rowsPerPage":50'),
    )
  })

  it('loads saved values from localStorage', () => {
    const saved = {
      apiBaseUrl: 'http://example.com',
      pollingIntervalMs: 30000,
      rowsPerPage: 10,
      sidebarCollapsed: true,
    }
    localStorageMock.setItem(STORAGE_KEY, JSON.stringify(saved))

    // Need a fresh pinia so the store re-initializes
    setActivePinia(createPinia())
    const store = useSettingsStore()

    expect(store.apiBaseUrl).toBe('http://example.com')
    expect(store.pollingIntervalMs).toBe(30000)
    expect(store.rowsPerPage).toBe(10)
    expect(store.sidebarCollapsed).toBe(true)
  })

  it('reset() restores defaults', async () => {
    const store = useSettingsStore()
    store.rowsPerPage = 50
    store.pollingIntervalMs = 5000
    store.sidebarCollapsed = true
    await nextTick()

    store.reset()
    await nextTick()

    expect(store.rowsPerPage).toBe(20)
    expect(store.pollingIntervalMs).toBe(60000)
    expect(store.sidebarCollapsed).toBe(false)
  })

  it('individual fields are independently reactive', async () => {
    const store = useSettingsStore()
    store.rowsPerPage = 50
    await nextTick()

    expect(store.rowsPerPage).toBe(50)
    expect(store.pollingIntervalMs).toBe(60000)
  })
})
