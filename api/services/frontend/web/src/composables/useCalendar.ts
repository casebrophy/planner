import { ref, computed, onMounted } from 'vue'
import { scheduleService } from '@/services/scheduleService'
import { useTimeBlockStore } from '@/stores/timeBlockStore'
import { useToastStore } from '@/stores/toastStore'
import { usePolling } from './usePolling'
import type { ScheduleItem } from '@/types/schedule'
import type { NewTimeBlock, UpdateTimeBlock } from '@/types/timeBlock'
import {
  startOfWeek,
  endOfWeek,
  addWeeks,
  subWeeks,
  format,
  eachDayOfInterval,
  isSameDay,
  parseISO,
} from 'date-fns'

export function useCalendar() {
  const currentDate = ref(new Date())
  const items = ref<ScheduleItem[]>([])
  const loading = ref(false)

  const timeBlockStore = useTimeBlockStore()
  const toasts = useToastStore()

  // Week boundaries (Monday start)
  const weekStart = computed(() => startOfWeek(currentDate.value, { weekStartsOn: 1 }))
  const weekEnd = computed(() => endOfWeek(currentDate.value, { weekStartsOn: 1 }))

  const weekDays = computed(() =>
    eachDayOfInterval({ start: weekStart.value, end: weekEnd.value }),
  )

  const weekLabel = computed(() => {
    const s = format(weekStart.value, 'MMM d')
    const e = format(weekEnd.value, 'MMM d, yyyy')
    return `${s} – ${e}`
  })

  // All-day items separated from timed items
  const allDayItems = computed(() => items.value.filter((i) => i.allDay))

  const timedItems = computed(() => items.value.filter((i) => !i.allDay))

  // Group items by day
  function itemsForDay(day: Date): ScheduleItem[] {
    return timedItems.value.filter((item) => isSameDay(parseISO(item.startsAt), day))
  }

  function allDayItemsForDay(day: Date): ScheduleItem[] {
    return allDayItems.value.filter((item) => isSameDay(parseISO(item.startsAt), day))
  }

  // Navigation
  function nextWeek() {
    currentDate.value = addWeeks(currentDate.value, 1)
    load()
  }

  function prevWeek() {
    currentDate.value = subWeeks(currentDate.value, 1)
    load()
  }

  function goToToday() {
    currentDate.value = new Date()
    load()
  }

  // Data loading
  async function load() {
    loading.value = true
    try {
      const start = weekStart.value.toISOString()
      const end = weekEnd.value.toISOString()
      const response = await scheduleService.getSchedule(start, end)
      items.value = response.items
    } catch (e) {
      console.error('calendar load failed', e)
      items.value = []
      toasts.error(e instanceof Error ? e.message : 'Failed to load calendar')
    } finally {
      loading.value = false
    }
  }

  // Time block CRUD
  async function createTimeBlock(data: NewTimeBlock) {
    await timeBlockStore.create(data)
    await load()
  }

  async function updateTimeBlock(id: string, data: UpdateTimeBlock) {
    await timeBlockStore.update(id, data)
    await load()
  }

  async function removeTimeBlock(id: string) {
    await timeBlockStore.remove(id)
    await load()
  }

  async function confirmTimeBlock(id: string) {
    await timeBlockStore.update(id, { confirmed: true })
    await load()
  }

  // Form state
  const showForm = ref(false)
  const formStartsAt = ref('')
  const formEndsAt = ref('')

  function openCreateAtSlot(startsAt: string, endsAt: string) {
    formStartsAt.value = startsAt
    formEndsAt.value = endsAt
    showForm.value = true
  }

  function closeForm() {
    showForm.value = false
    formStartsAt.value = ''
    formEndsAt.value = ''
  }

  onMounted(load)
  usePolling(load)

  return {
    currentDate,
    weekStart,
    weekEnd,
    weekDays,
    weekLabel,
    items,
    loading,
    allDayItems,
    timedItems,
    itemsForDay,
    allDayItemsForDay,
    nextWeek,
    prevWeek,
    goToToday,
    refresh: load,
    createTimeBlock,
    updateTimeBlock,
    removeTimeBlock,
    confirmTimeBlock,
    showForm,
    formStartsAt,
    formEndsAt,
    openCreateAtSlot,
    closeForm,
  }
}
