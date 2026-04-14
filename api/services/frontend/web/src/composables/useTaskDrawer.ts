import { ref } from 'vue'

const isOpen = ref(false)
const taskId = ref<string | null>(null)

export function useTaskDrawer() {
  function open(id: string) {
    taskId.value = id
    isOpen.value = true
  }

  function close() {
    isOpen.value = false
    taskId.value = null
  }

  return {
    isOpen,
    taskId,
    open,
    close,
  }
}
