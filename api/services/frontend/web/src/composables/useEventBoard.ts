import { onMounted, computed, ref } from 'vue'
import { useCalendarEventStore } from '@/stores/calendarEventStore'
import { storeToRefs } from 'pinia'
import { usePolling } from './usePolling'
import type { CalendarEvent, NewCalendarEvent, UpdateCalendarEvent } from '@/types/calendarEvent'

export function useEventBoard() {
  const store = useCalendarEventStore()
  const { items, total, loading } = storeToRefs(store)

  // Upcoming events (startsAt >= now, sorted ascending)
  const upcomingEvents = computed(() =>
    items.value
      .filter(e => new Date(e.startsAt) >= new Date())
      .sort((a, b) => new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime())
  )

  // Past events (startsAt < now, sorted descending)
  const pastEvents = computed(() =>
    items.value
      .filter(e => new Date(e.startsAt) < new Date())
      .sort((a, b) => new Date(b.startsAt).getTime() - new Date(a.startsAt).getTime())
  )

  // Form state
  const showForm = ref(false)
  const editingEvent = ref<CalendarEvent | null>(null)

  function openCreate() {
    editingEvent.value = null
    showForm.value = true
  }

  function openEdit(event: CalendarEvent) {
    editingEvent.value = event
    showForm.value = true
  }

  function closeForm() {
    showForm.value = false
    editingEvent.value = null
  }

  async function load() {
    await store.fetchList(true)
  }

  async function create(data: NewCalendarEvent) {
    await store.create(data)
  }

  async function update(id: string, data: UpdateCalendarEvent) {
    await store.update(id, data)
  }

  async function remove(id: string) {
    await store.remove(id)
  }

  onMounted(load)
  usePolling(() => store.fetchList(true))

  return {
    items,
    loading,
    total,
    upcomingEvents,
    pastEvents,
    showForm,
    editingEvent,
    openCreate,
    openEdit,
    closeForm,
    refresh: load,
    create,
    update,
    remove,
  }
}
