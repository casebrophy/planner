import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

const STORAGE_KEY = 'planner-settings'

interface Settings {
  apiBaseUrl: string
  pollingIntervalMs: number
  rowsPerPage: number
  sidebarCollapsed: boolean
}

const defaults: Settings = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || '',
  pollingIntervalMs: 60000,
  rowsPerPage: 20,
  sidebarCollapsed: false,
}

function loadFromStorage(): Partial<Settings> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      return JSON.parse(raw) as Partial<Settings>
    }
  } catch {
    // ignore corrupt data
  }
  return {}
}

function saveToStorage(settings: Settings): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
}

export const useSettingsStore = defineStore('settings', () => {
  const saved = loadFromStorage()

  const apiBaseUrl = ref(saved.apiBaseUrl ?? defaults.apiBaseUrl)
  const pollingIntervalMs = ref(saved.pollingIntervalMs ?? defaults.pollingIntervalMs)
  const rowsPerPage = ref(saved.rowsPerPage ?? defaults.rowsPerPage)
  const sidebarCollapsed = ref(saved.sidebarCollapsed ?? defaults.sidebarCollapsed)

  function persist() {
    saveToStorage({
      apiBaseUrl: apiBaseUrl.value,
      pollingIntervalMs: pollingIntervalMs.value,
      rowsPerPage: rowsPerPage.value,
      sidebarCollapsed: sidebarCollapsed.value,
    })
  }

  watch(apiBaseUrl, persist)
  watch(pollingIntervalMs, persist)
  watch(rowsPerPage, persist)
  watch(sidebarCollapsed, persist)

  function reset() {
    apiBaseUrl.value = defaults.apiBaseUrl
    pollingIntervalMs.value = defaults.pollingIntervalMs
    rowsPerPage.value = defaults.rowsPerPage
    sidebarCollapsed.value = defaults.sidebarCollapsed
  }

  return { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed, reset }
})
