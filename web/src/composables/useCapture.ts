import { ref, computed } from 'vue'
import { useCaptureStore, type CaptureMode } from '@/stores/captureStore'
import { storeToRefs } from 'pinia'
import type { NewTask, NewContext } from '@/types'
import { useRouter } from 'vue-router'

export function useCapture() {
  const store = useCaptureStore()
  const router = useRouter()
  const { mode, submitting } = storeToRefs(store)

  const taskForm = ref<NewTask>(store.defaultTask())
  const contextForm = ref<NewContext>(store.defaultContext())

  const isValid = computed(() => {
    if (mode.value === 'task') {
      return taskForm.value.title.trim().length > 0
    }
    return contextForm.value.title.trim().length > 0
  })

  function setMode(m: CaptureMode) {
    mode.value = m
  }

  async function submit() {
    if (!isValid.value) return

    if (mode.value === 'task') {
      const task = await store.submitTask(taskForm.value)
      if (task) {
        taskForm.value = store.defaultTask()
        router.push({ name: 'task-detail', params: { id: task.id } })
      }
    } else {
      const ctx = await store.submitContext(contextForm.value)
      if (ctx) {
        contextForm.value = store.defaultContext()
        router.push({ name: 'context-detail', params: { id: ctx.id } })
      }
    }
  }

  function reset() {
    taskForm.value = store.defaultTask()
    contextForm.value = store.defaultContext()
  }

  return {
    mode,
    submitting,
    taskForm,
    contextForm,
    isValid,
    setMode,
    submit,
    reset,
  }
}
