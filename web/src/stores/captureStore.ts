import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useTaskStore } from './taskStore'
import { useContextStore } from './contextStore'
import type { NewTask, NewContext } from '@/types'
import { TaskPriority, TaskEnergy } from '@/types'

export type CaptureMode = 'task' | 'context'

export const useCaptureStore = defineStore('capture', () => {
  const mode = ref<CaptureMode>('task')
  const submitting = ref(false)

  const taskStore = useTaskStore()
  const contextStore = useContextStore()

  async function submitTask(task: NewTask) {
    submitting.value = true
    try {
      return await taskStore.createTask(task)
    } finally {
      submitting.value = false
    }
  }

  async function submitContext(ctx: NewContext) {
    submitting.value = true
    try {
      return await contextStore.createContext(ctx)
    } finally {
      submitting.value = false
    }
  }

  function defaultTask(): NewTask {
    return {
      title: '',
      description: '',
      priority: TaskPriority.Medium,
      energy: TaskEnergy.Medium,
    }
  }

  function defaultContext(): NewContext {
    return {
      title: '',
      description: '',
    }
  }

  return { mode, submitting, submitTask, submitContext, defaultTask, defaultContext }
})
