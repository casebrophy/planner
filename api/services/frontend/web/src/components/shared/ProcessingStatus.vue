<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { request } from '@/services/client'

const props = defineProps<{
  rawInputId: string
}>()

interface RawInput {
  id: string
  sourceType: string
  status: string
  rawContent: string
  processedAt?: string
  error?: string
  createdAt: string
}

const rawInput = ref<RawInput | null>(null)
const loading = ref(false)
const fetchError = ref<string | null>(null)

const steps = ['pending', 'processing', 'processed'] as const
const currentStepIndex = computed(() => {
  if (!rawInput.value) return -1
  if (rawInput.value.status === 'error') return -1
  if (rawInput.value.status === 'partial') return steps.length - 1
  return steps.indexOf(rawInput.value.status as (typeof steps)[number])
})

const isError = computed(() => rawInput.value?.status === 'error')
const isPartial = computed(() => rawInput.value?.status === 'partial')

async function fetchData() {
  loading.value = true
  fetchError.value = null
  try {
    rawInput.value = await request<RawInput>(`/api/v1/raw-inputs/${props.rawInputId}`)
  } catch (e) {
    fetchError.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<template>
  <!-- Loading -->
  <div
    v-if="loading"
    class="flex items-center justify-center py-4"
  >
    <svg
      class="animate-spin w-5 h-5 text-blue-500"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  </div>

  <!-- Error fetching -->
  <div
    v-else-if="fetchError"
    class="text-sm text-red-400"
  >
    {{ fetchError }}
  </div>

  <!-- Steps -->
  <div
    v-else-if="rawInput"
    class="flex items-center gap-0"
  >
    <template
      v-for="(step, idx) in steps"
      :key="step"
    >
      <!-- Connector line -->
      <div
        v-if="idx > 0"
        class="h-0.5 w-8"
        :class="idx <= currentStepIndex ? (isPartial && idx === currentStepIndex ? 'bg-orange-400' : 'bg-green-400') : 'bg-gray-700'"
      />

      <!-- Step circle + label -->
      <div class="flex flex-col items-center">
        <!-- Error state -->
        <div
          v-if="isError && idx === 0"
          class="w-6 h-6 rounded-full border-2 border-red-400 flex items-center justify-center"
        >
          <svg
            class="w-3 h-3 text-red-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="3"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </div>
        <!-- Completed -->
        <div
          v-else-if="idx < currentStepIndex"
          class="w-6 h-6 rounded-full border-2 border-green-400 flex items-center justify-center"
        >
          <svg
            class="w-3 h-3 text-green-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="3"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>
        <!-- Partial (final step reached with warnings) -->
        <div
          v-else-if="isPartial && idx === currentStepIndex"
          class="w-6 h-6 rounded-full border-2 border-orange-400 flex items-center justify-center"
        >
          <svg
            class="w-3 h-3 text-orange-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="3"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M12 9v2m0 4h.01M12 3l9.66 16.59A1 1 0 0120.66 21H3.34a1 1 0 01-.86-1.41L12 3z"
            />
          </svg>
        </div>
        <!-- Current -->
        <div
          v-else-if="idx === currentStepIndex"
          class="w-6 h-6 rounded-full border-2 border-blue-400 flex items-center justify-center animate-pulse"
        >
          <span class="text-xs font-medium text-blue-400">{{ idx + 1 }}</span>
        </div>
        <!-- Future -->
        <div
          v-else
          class="w-6 h-6 rounded-full border-2 border-gray-700 flex items-center justify-center"
        >
          <span class="text-xs font-medium text-gray-600">{{ idx + 1 }}</span>
        </div>

        <span
          class="text-xs mt-1 capitalize"
          :class="{
            'text-red-400': isError && idx === 0,
            'text-orange-400': isPartial && idx === currentStepIndex,
            'text-green-400': !isError && !isPartial && idx < currentStepIndex,
            'text-blue-400': !isError && !isPartial && idx === currentStepIndex,
            'text-gray-600': !isError && idx > currentStepIndex,
          }"
        >
          {{ isError && idx === 0 ? 'error' : isPartial && idx === currentStepIndex ? 'partial' : step }}
        </span>
      </div>
    </template>
  </div>
</template>
