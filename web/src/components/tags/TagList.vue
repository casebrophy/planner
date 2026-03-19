<script setup lang="ts">
import type { Tag } from '@/types'
import TagBadge from './TagBadge.vue'

defineProps<{
  tags: Tag[]
  removable?: boolean
}>()

const emit = defineEmits<{
  remove: [id: string]
}>()
</script>

<template>
  <div class="flex flex-wrap gap-1.5">
    <TagBadge
      v-for="tag in tags"
      :key="tag.id"
      :tag="tag"
      :removable="removable"
      @remove="emit('remove', $event)"
    />
    <span v-if="tags.length === 0" class="text-xs text-gray-500">No tags</span>
  </div>
</template>
