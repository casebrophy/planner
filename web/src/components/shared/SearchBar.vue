<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'

const STORAGE_KEY = 'planner-recent-searches'
const MAX_RECENT = 5

const props = withDefaults(
  defineProps<{
    modelValue: string
    placeholder?: string
    debounceMs?: number
  }>(),
  {
    placeholder: 'Search tasks, contexts, and tags...',
    debounceMs: 300,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const internalValue = ref(props.modelValue)
const focused = ref(false)
const recentSearches = ref<string[]>([])
const inputRef = ref<HTMLInputElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

function loadRecent() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      recentSearches.value = JSON.parse(stored) as string[]
    }
  } catch {
    recentSearches.value = []
  }
}

function saveRecent(term: string) {
  const trimmed = term.trim()
  if (!trimmed) return
  const filtered = recentSearches.value.filter((s) => s !== trimmed)
  filtered.unshift(trimmed)
  recentSearches.value = filtered.slice(0, MAX_RECENT)
  localStorage.setItem(STORAGE_KEY, JSON.stringify(recentSearches.value))
}

watch(
  () => props.modelValue,
  (val) => {
    internalValue.value = val
  },
)

watch(internalValue, (val) => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    emit('update:modelValue', val)
    if (val.trim().length >= 2) {
      saveRecent(val.trim())
    }
  }, props.debounceMs)
})

function clear() {
  internalValue.value = ''
  emit('update:modelValue', '')
  inputRef.value?.focus()
}

function selectRecent(term: string) {
  internalValue.value = term
  emit('update:modelValue', term)
  focused.value = false
}

function handleClickOutside(e: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(e.target as Node)) {
    focused.value = false
  }
}

const showRecent = ref(false)

watch([focused, internalValue], () => {
  showRecent.value = focused.value && !internalValue.value && recentSearches.value.length > 0
})

onMounted(() => {
  loadRecent()
  document.addEventListener('mousedown', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', handleClickOutside)
  if (debounceTimer) clearTimeout(debounceTimer)
})
</script>

<template>
  <div
    ref="containerRef"
    class="relative"
  >
    <div class="relative">
      <!-- Search icon -->
      <svg
        class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
        />
      </svg>

      <input
        ref="inputRef"
        v-model="internalValue"
        type="text"
        :placeholder="placeholder"
        class="w-full bg-gray-900 border border-gray-800 text-gray-100 text-sm rounded-lg pl-10 pr-10 py-3 placeholder-gray-500 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 outline-none transition-colors"
        @focus="focused = true"
      >

      <!-- Clear button -->
      <button
        v-if="internalValue"
        class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300 transition-colors"
        aria-label="Clear search"
        @click="clear"
      >
        <svg
          class="w-4 h-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Recent searches dropdown -->
    <div
      v-if="showRecent"
      class="absolute z-10 mt-1 w-full bg-gray-900 border border-gray-800 rounded-lg shadow-lg overflow-hidden"
    >
      <div class="px-3 py-2 text-xs text-gray-500 font-medium">
        Recent searches
      </div>
      <button
        v-for="term in recentSearches"
        :key="term"
        class="w-full text-left px-3 py-2 text-sm text-gray-300 hover:bg-gray-800 transition-colors"
        @click="selectRecent(term)"
      >
        {{ term }}
      </button>
    </div>
  </div>
</template>
