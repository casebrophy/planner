<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTagStore } from '@/stores/tagStore'
import type { Tag } from '@/types'

const props = defineProps<{
  selectedIds: string[]
}>()

const emit = defineEmits<{
  add: [tagId: string]
  create: [name: string]
}>()

const tagStore = useTagStore()
const search = ref('')
const showDropdown = ref(false)

function hideDropdownDelayed() {
  setTimeout(() => (showDropdown.value = false), 200)
}

const available = computed(() =>
  tagStore.items.filter(
    (t) =>
      !props.selectedIds.includes(t.id) &&
      t.name.toLowerCase().includes(search.value.toLowerCase()),
  ),
)

const canCreate = computed(
  () =>
    search.value.trim().length > 0 &&
    !tagStore.items.some((t) => t.name.toLowerCase() === search.value.trim().toLowerCase()),
)

onMounted(() => {
  tagStore.fetchTags()
})

function select(tag: Tag) {
  emit('add', tag.id)
  search.value = ''
  showDropdown.value = false
}

async function createAndAdd() {
  const name = search.value.trim()
  if (!name) return
  emit('create', name)
  search.value = ''
  showDropdown.value = false
}
</script>

<template>
  <div class="relative">
    <input
      v-model="search"
      type="text"
      placeholder="Add tag..."
      class="w-full bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-1.5 focus:outline-none focus:border-blue-500"
      @focus="showDropdown = true"
      @blur="hideDropdownDelayed"
    >

    <div
      v-if="showDropdown && (available.length > 0 || canCreate)"
      class="absolute top-full mt-1 w-full bg-gray-800 border border-gray-700 rounded-lg shadow-xl z-10 max-h-48 overflow-y-auto"
    >
      <button
        v-for="tag in available"
        :key="tag.id"
        class="w-full text-left px-3 py-2 text-sm text-gray-300 hover:bg-gray-700 transition-colors"
        @mousedown.prevent="select(tag)"
      >
        {{ tag.name }}
      </button>
      <button
        v-if="canCreate"
        class="w-full text-left px-3 py-2 text-sm text-blue-400 hover:bg-gray-700 transition-colors"
        @mousedown.prevent="createAndAdd"
      >
        Create "{{ search.trim() }}"
      </button>
    </div>
  </div>
</template>
