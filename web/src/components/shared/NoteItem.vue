<script setup lang="ts">
import { computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'

interface ThreadEntry {
  id: string
  subjectType: string
  subjectId: string
  kind: string
  content: string
  source: string
  metadata?: Record<string, unknown>
  sourceId?: string
  sentiment?: string
  requiresAction: boolean
  createdAt: string
}

const props = defineProps<{
  entry: ThreadEntry
}>()

const timeAgo = computed(() =>
  formatDistanceToNow(new Date(props.entry.createdAt), { addSuffix: true }),
)
</script>

<template>
  <div class="flex gap-3 py-3">
    <!-- Source icon -->
    <div class="shrink-0 mt-0.5">
      <!-- user -->
      <svg
        v-if="entry.source === 'user'"
        class="w-4 h-4 text-blue-400"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z"
        />
      </svg>
      <!-- claude -->
      <svg
        v-else-if="entry.source === 'claude'"
        class="w-4 h-4 text-purple-400"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456z"
        />
      </svg>
      <!-- system -->
      <svg
        v-else-if="entry.source === 'system'"
        class="w-4 h-4 text-gray-500"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z"
        />
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
        />
      </svg>
      <!-- email -->
      <svg
        v-else-if="entry.source === 'email'"
        class="w-4 h-4 text-amber-400"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75"
        />
      </svg>
      <!-- default -->
      <svg
        v-else
        class="w-4 h-4 text-gray-500"
        fill="currentColor"
        viewBox="0 0 24 24"
      >
        <circle
          cx="12"
          cy="12"
          r="4"
        />
      </svg>
    </div>

    <!-- Content -->
    <div class="flex-1 min-w-0">
      <div class="flex items-baseline gap-2">
        <span class="text-xs font-medium text-gray-400 capitalize">{{ entry.source }}</span>
        <span class="text-xs text-gray-600">{{ timeAgo }}</span>
        <span
          v-if="entry.requiresAction"
          class="text-xs text-amber-400 font-medium"
        >Action needed</span>
      </div>
      <p class="mt-0.5 text-sm text-gray-300 whitespace-pre-wrap">
        {{ entry.content }}
      </p>
      <span
        v-if="entry.sentiment"
        class="mt-1 inline-block text-xs text-gray-500"
      >{{ entry.sentiment }}</span>
    </div>
  </div>
</template>
