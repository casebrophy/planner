import { defineStore } from 'pinia'
import { ref } from 'vue'
import { dailyPlanService } from '@/services/dailyPlanService'
import type { DailyPlan } from '@/types/dailyPlan'
import { useToastStore } from './toastStore'

export const useDailyPlanStore = defineStore('dailyPlan', () => {
  const plan = ref<DailyPlan | null>(null)
  const loading = ref(false)
  const generating = ref(false)
  const toasts = useToastStore()

  async function fetchPlan(date?: string) {
    loading.value = true
    try {
      plan.value = await dailyPlanService.getPlan(date)
    } catch {
      toasts.error('Failed to load daily plan')
    } finally {
      loading.value = false
    }
  }

  async function regenerate(date?: string) {
    generating.value = true
    try {
      plan.value = await dailyPlanService.generate(date)
      toasts.success('Daily plan generated')
    } catch {
      toasts.error('Failed to generate plan')
    } finally {
      generating.value = false
    }
  }

  async function completeItem(itemId: string) {
    // Optimistic update
    if (plan.value) {
      const item = plan.value.items.find((i) => i.id === itemId)
      if (item) {
        const prev = item.status
        item.status = 'completed'
        item.completedAt = new Date().toISOString()
        try {
          await dailyPlanService.completeItem(itemId)
        } catch {
          item.status = prev
          item.completedAt = undefined
          toasts.error('Failed to complete item')
        }
      }
    }
  }

  async function dismissItem(itemId: string, reason: string, note?: string) {
    if (plan.value) {
      const item = plan.value.items.find((i) => i.id === itemId)
      if (item) {
        const prev = { status: item.status, dismissReason: item.dismissReason, dismissNote: item.dismissNote }
        item.status = 'dismissed'
        item.dismissReason = reason
        item.dismissNote = note
        try {
          await dailyPlanService.dismissItem(itemId, { reason, note })
        } catch {
          item.status = prev.status
          item.dismissReason = prev.dismissReason
          item.dismissNote = prev.dismissNote
          toasts.error('Failed to dismiss item')
        }
      }
    }
  }

  async function reorderItem(itemId: string, userPosition: number) {
    if (plan.value) {
      const item = plan.value.items.find((i) => i.id === itemId)
      if (item) {
        item.userPosition = userPosition
        try {
          await dailyPlanService.updateItem(itemId, { userPosition })
        } catch {
          toasts.error('Failed to reorder item')
        }
      }
    }
  }

  return { plan, loading, generating, fetchPlan, regenerate, completeItem, dismissItem, reorderItem }
})
