import { ref } from 'vue'
import { useSettingsStore } from '@/stores/settingsStore'
import { storeToRefs } from 'pinia'

export function useSettings() {
  const settingsStore = useSettingsStore()
  const { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed } =
    storeToRefs(settingsStore)
  const saved = ref(false)

  function save() {
    saved.value = true
    setTimeout(() => {
      saved.value = false
    }, 2000)
  }

  function reset() {
    settingsStore.reset()
  }

  return { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed, saved, save, reset }
}
