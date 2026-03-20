<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useClarificationStore } from '@/stores/clarificationStore'
import ClarificationCard from './ClarificationCard.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'

const store = useClarificationStore()
const { currentItem, loading, isEmpty, progress, items } = storeToRefs(store)

// NOTE: Do NOT fetch here -- the parent ClarificationView's useClarification composable
// already calls fetchQueue() on mount. Fetching here would cause a double-fetch.

async function handleResolve(answer: Record<string, unknown>) {
  if (currentItem.value) {
    await store.resolve(currentItem.value.id, answer)
  }
}

async function handleSnooze(hours: number) {
  if (currentItem.value) {
    await store.snooze(currentItem.value.id, hours)
  }
}

async function handleDismiss() {
  if (currentItem.value) {
    await store.dismiss(currentItem.value.id)
  }
}
</script>

<template>
  <div>
    <LoadingSpinner v-if="loading && items.length === 0" />

    <EmptyState
      v-else-if="isEmpty"
      title="All caught up"
      message="No pending clarifications. Nice work!"
    />

    <div v-else>
      <!-- Progress -->
      <div class="flex items-center justify-between mb-4">
        <span class="text-sm text-gray-500">
          {{ progress.current }} of {{ progress.total }}
        </span>
      </div>

      <!-- Current Card -->
      <Transition
        name="slide"
        mode="out-in"
      >
        <ClarificationCard
          v-if="currentItem"
          :key="currentItem.id"
          :item="currentItem"
          @resolve="handleResolve"
          @snooze="handleSnooze"
          @dismiss="handleDismiss"
        />
      </Transition>

      <!-- Progress Dots -->
      <div
        v-if="items.length > 1"
        class="flex justify-center gap-2 mt-4"
      >
        <button
          v-for="(item, idx) in items"
          :key="item.id"
          class="w-2.5 h-2.5 rounded-full transition-colors"
          :class="idx === store.currentIndex ? 'bg-amber-500' : 'bg-gray-700'"
          @click="store.goTo(idx)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.2s ease;
}
.slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}
</style>
