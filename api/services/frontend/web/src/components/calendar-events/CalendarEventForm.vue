<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { CalendarEvent, NewCalendarEvent, UpdateCalendarEvent } from '@/types/calendarEvent'
import { useContextStore } from '@/stores/contextStore'

const props = defineProps<{
  event?: CalendarEvent | null
}>()

const emit = defineEmits<{
  save: [data: NewCalendarEvent | UpdateCalendarEvent]
  cancel: []
}>()

const contextStore = useContextStore()

const title = ref(props.event?.title ?? '')
const description = ref(props.event?.description ?? '')
const location = ref(props.event?.location ?? '')
const allDay = ref(props.event?.allDay ?? false)
const startsAt = ref(props.event ? props.event.startsAt.slice(0, 16) : '')
const endsAt = ref(props.event ? props.event.endsAt.slice(0, 16) : '')
const contextId = ref(props.event?.contextId ?? '')

const isValid = computed(() => title.value.trim().length > 0 && startsAt.value && endsAt.value)

const isEditing = computed(() => !!props.event)

onMounted(() => {
  contextStore.fetchList()
})

function handleSubmit() {
  if (!isValid.value) return

  if (isEditing.value && props.event) {
    const data: UpdateCalendarEvent = {
      title: title.value.trim(),
      description: description.value.trim(),
      allDay: allDay.value,
      startsAt: new Date(startsAt.value).toISOString(),
      endsAt: new Date(endsAt.value).toISOString(),
    }
    if (location.value) data.location = location.value.trim()
    if (contextId.value) data.contextId = contextId.value
    emit('save', data)
  } else {
    const data: NewCalendarEvent = {
      title: title.value.trim(),
      description: description.value.trim(),
      allDay: allDay.value,
      startsAt: new Date(startsAt.value).toISOString(),
      endsAt: new Date(endsAt.value).toISOString(),
    }
    if (location.value) data.location = location.value.trim()
    if (contextId.value) data.contextId = contextId.value
    emit('save', data)
  }
}
</script>

<template>
  <form
    class="space-y-4"
    @submit.prevent="handleSubmit"
  >
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Title</label>
      <input
        v-model="title"
        type="text"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        placeholder="Event title"
      >
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Description</label>
      <textarea
        v-model="description"
        rows="3"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        placeholder="Description"
      />
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Start Date & Time</label>
        <input
          v-model="startsAt"
          type="datetime-local"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">End Date & Time</label>
        <input
          v-model="endsAt"
          type="datetime-local"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
      </div>
    </div>

    <div class="flex items-center gap-2">
      <input
        v-model="allDay"
        type="checkbox"
        id="all-day"
        class="w-4 h-4 rounded bg-gray-800 border-gray-700 text-blue-600 focus:border-blue-500"
      >
      <label for="all-day" class="text-sm font-medium text-gray-300">All-day event</label>
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Location</label>
        <input
          v-model="location"
          type="text"
          class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          placeholder="Location"
        >
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Context</label>
        <select
          v-model="contextId"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
          <option value="">
            No context
          </option>
          <option
            v-for="ctx in contextStore.items"
            :key="ctx.id"
            :value="ctx.id"
          >
            {{ ctx.title }}
          </option>
        </select>
      </div>
    </div>

    <div class="flex justify-end gap-3 pt-2">
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
        @click="emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        :disabled="!isValid"
        class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        {{ isEditing ? 'Save Changes' : 'Add Event' }}
      </button>
    </div>
  </form>
</template>
