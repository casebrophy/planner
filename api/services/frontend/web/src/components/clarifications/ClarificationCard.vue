<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { ClarificationKind, ClarificationKindLabels, ClarificationKindColors, ContextKind } from '@/types/enums'
import type { ClarificationItem } from '@/types'
import type { ContextAssignmentOptions, AmbiguousActionOptions, ContextRef, EntityLinkOptions, TypeAssignmentOptions, KnowledgeGapOptions, VoiceReferenceOptions, AmbiguousDeadlineOptions, EventPrepOptions, AmbiguousEntityMatchOptions } from '@/types/generated/clarification-options'
import type { ClarificationAnswerOptions } from '@/types/clarification'
import { contextService } from '@/services/contextService'

const props = defineProps<{
  item: ClarificationItem
}>()

const emit = defineEmits<{
  resolve: [answer: Record<string, unknown>]
  snooze: [hours: number]
  dismiss: []
}>()

const CHIPS_THRESHOLD = 0.6

const debriefAnswer = ref('')
const showNoteInput = ref(false)
const selectedWeeklyTasks = ref(new Set<string>())

const freeTextOverride = ref('')
const showFreeTextInput = ref(false)

watch(() => props.item, () => {
  freeTextOverride.value = ''
  showFreeTextInput.value = false
})

function toggleWeeklyTask(id: string) {
  const next = new Set(selectedWeeklyTasks.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedWeeklyTasks.value = next
}
const noteText = ref('')
const newContextTitle = ref('')
const newContextKind = ref<ContextKind>(ContextKind.Project)
const isCreating = ref(false)
const createError = ref<string | null>(null)

const kindLabel = computed(() => ClarificationKindLabels[props.item.kind] ?? props.item.kind)
const kindColor = computed(() => ClarificationKindColors[props.item.kind] ?? '#6b7280')
const age = computed(() => formatDistanceToNow(new Date(props.item.createdAt), { addSuffix: true }))

const options = computed((): ClarificationAnswerOptions => {
  if (!props.item.answerOptions) return null
  return typeof props.item.answerOptions === 'string'
    ? JSON.parse(props.item.answerOptions as string)
    : props.item.answerOptions as ClarificationAnswerOptions
})

const contextAssignmentOptions = computed<ContextAssignmentOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.ContextAssignment) return null
  return options.value as ContextAssignmentOptions | null
})

const entityLinkOptions = computed<EntityLinkOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.EntityLink) return null
  return options.value as EntityLinkOptions | null
})

const typeAssignmentOptions = computed<TypeAssignmentOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.TypeAssignment) return null
  return options.value as TypeAssignmentOptions | null
})

const knowledgeGapOptions = computed<KnowledgeGapOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.KnowledgeGap) return null
  return options.value as KnowledgeGapOptions | null
})

const voiceReferenceOptions = computed<VoiceReferenceOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.VoiceReference) return null
  return options.value as VoiceReferenceOptions | null
})

const ambiguousDeadlineOptions = computed<AmbiguousDeadlineOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.AmbiguousDeadline) return null
  return options.value as AmbiguousDeadlineOptions | null
})

const eventPrepOptions = computed<EventPrepOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.EventPrep) return null
  return options.value as EventPrepOptions | null
})

const ambiguousEntityMatchOptions = computed<AmbiguousEntityMatchOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.AmbiguousEntityMatch) return null
  return options.value as AmbiguousEntityMatchOptions | null
})

interface WeeklyReviewTask { id: string; title: string }
interface WeeklyReviewOptions { tasks?: WeeklyReviewTask[] }

const weeklyReviewTasks = computed<WeeklyReviewTask[]>(() => {
  if (props.item.kind !== ClarificationKind.WeeklyReview) return []
  return (options.value as WeeklyReviewOptions | null)?.tasks ?? []
})

const availableContexts = computed<ContextRef[]>(() =>
  contextAssignmentOptions.value?.available_contexts ?? []
)

const suggestedContextId = computed<string | undefined>(() =>
  contextAssignmentOptions.value?.suggested_context
)

function resolveWithValue(answer: Record<string, unknown>) {
  emit('resolve', answer)
}

function resolveDebrief() {
  if (debriefAnswer.value.trim()) {
    emit('resolve', { response: debriefAnswer.value.trim() })
  }
}

async function createAndResolve() {
  const title = newContextTitle.value.trim()
  if (!title) return
  isCreating.value = true
  createError.value = null
  try {
    const ctx = await contextService.create({
      title,
      description: '',
      kind: newContextKind.value,
    })
    resolveWithValue({ context_id: ctx.id })
  } catch {
    createError.value = 'Failed to create — try again'
    isCreating.value = false
  }
}
</script>

<template>
  <div
    class="bg-gray-800 rounded-xl p-6 border-l-4"
    :style="{ borderLeftColor: kindColor }"
  >
    <!-- Header -->
    <div class="flex items-center gap-2 mb-3">
      <span
        class="px-2 py-0.5 rounded text-xs font-medium"
        :style="{ backgroundColor: kindColor + '22', color: kindColor }"
      >
        {{ kindLabel }}
      </span>
      <span class="text-gray-500 text-xs">{{ age }}</span>
    </div>

    <!-- Question -->
    <h3 class="text-lg font-semibold text-gray-100 mb-2">
      {{ item.question }}
    </h3>

    <!-- Subject context -->
    <div
      v-if="item.subjectType || item.subjectDescription"
      class="flex items-start gap-2 mb-2 bg-gray-700/50 rounded-lg px-3 py-2"
    >
      <span
        v-if="item.subjectType"
        class="shrink-0 mt-0.5 px-1.5 py-0.5 rounded text-xs font-medium bg-gray-600 text-gray-300 capitalize"
      >
        {{ item.subjectType }}
      </span>
      <span
        v-if="item.subjectDescription"
        class="text-sm text-gray-200"
      >
        {{ item.subjectDescription }}
      </span>
    </div>

    <!-- Reasoning (if present) -->
    <p
      v-if="item.reasoning"
      class="text-sm text-gray-400 mb-4"
    >
      {{ item.reasoning }}
    </p>

    <!-- Kind-specific actions -->
    <div class="mt-4">
      <!-- Context Assignment -->
      <div
        v-if="item.kind === ClarificationKind.ContextAssignment"
        class="flex flex-col gap-2"
      >
        <button
          v-if="suggestedContextId"
          :disabled="isCreating"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="resolveWithValue({ context_id: suggestedContextId })"
        >
          Confirm: {{ availableContexts.find(c => c.id === suggestedContextId)?.title ?? 'suggested context' }}
        </button>
        <button
          v-for="alt in availableContexts.filter(c => c.id !== suggestedContextId)"
          :key="alt.id"
          :disabled="isCreating"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="resolveWithValue({ context_id: alt.id })"
        >
          {{ alt.title }}
        </button>

        <!-- Or create new -->
        <p class="text-xs uppercase tracking-wide text-gray-500 mt-1">
          Or create new
        </p>
        <input
          v-model="newContextTitle"
          data-testid="new-context-title"
          type="text"
          placeholder="Context name…"
          :disabled="isCreating"
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-indigo-500 disabled:opacity-40"
        >
        <div class="flex gap-2 items-center">
          <div class="flex rounded-lg overflow-hidden border border-gray-600 flex-1">
            <button
              data-testid="kind-project"
              :disabled="isCreating"
              :class="[
                'flex-1 py-1.5 text-xs font-medium transition-colors',
                newContextKind === ContextKind.Project
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-400 hover:text-gray-200'
              ]"
              @click="newContextKind = ContextKind.Project"
            >
              Project
            </button>
            <button
              data-testid="kind-area"
              :disabled="isCreating"
              :class="[
                'flex-1 py-1.5 text-xs font-medium transition-colors',
                newContextKind === ContextKind.Area
                  ? 'bg-violet-600 text-white'
                  : 'bg-gray-700 text-gray-400 hover:text-gray-200'
              ]"
              @click="newContextKind = ContextKind.Area"
            >
              Area
            </button>
          </div>
          <button
            data-testid="create-context-btn"
            :disabled="isCreating || !newContextTitle.trim()"
            class="px-4 py-1.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            @click="createAndResolve"
          >
            <span v-if="isCreating">…</span>
            <span v-else>+</span>
          </button>
        </div>
        <p
          v-if="createError"
          data-testid="create-error"
          class="text-xs text-red-400"
        >
          {{ createError }}
        </p>
      </div>

      <!-- Inactivity Prompt -->
      <div
        v-if="item.kind === ClarificationKind.InactivityPrompt"
        class="flex flex-col gap-2"
      >
        <div
          v-if="showNoteInput"
          class="flex flex-col gap-2"
        >
          <textarea
            v-model="noteText"
            rows="3"
            placeholder="Add a note about this item..."
            class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-amber-500 resize-none"
          />
          <div class="flex gap-2">
            <button
              :disabled="!noteText.trim()"
              class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              @click="resolveWithValue({ action: 'note', note: noteText.trim() })"
            >
              Submit note
            </button>
            <button
              class="px-4 py-2.5 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
              @click="showNoteInput = false; noteText = ''"
            >
              Cancel
            </button>
          </div>
        </div>
        <div
          v-else
          class="flex gap-2"
        >
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
            @click="resolveWithValue({ action: 'extend' })"
          >
            Still active
          </button>
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors"
            @click="showNoteInput = true"
          >
            Add note
          </button>
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
            @click="resolveWithValue({ action: 'close' })"
          >
            Close
          </button>
        </div>
      </div>

      <!-- Stale Task -->
      <div
        v-else-if="item.kind === ClarificationKind.StaleTask"
        class="flex flex-col gap-2"
      >
        <div
          v-if="showNoteInput"
          class="flex flex-col gap-2"
        >
          <textarea
            v-model="noteText"
            rows="3"
            placeholder="Add a note about this item..."
            class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-amber-500 resize-none"
          />
          <div class="flex gap-2">
            <button
              :disabled="!noteText.trim()"
              data-testid="stale-task-note-submit"
              class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              @click="resolveWithValue({ status: 'open', note: noteText.trim() })"
            >
              Submit note
            </button>
            <button
              class="px-4 py-2.5 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
              @click="showNoteInput = false; noteText = ''"
            >
              Cancel
            </button>
          </div>
        </div>
        <div
          v-else
          class="flex gap-2"
        >
          <button
            data-testid="stale-task-still-active"
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
            @click="resolveWithValue({ status: 'open' })"
          >
            Still active
          </button>
          <button
            data-testid="stale-task-add-note"
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors"
            @click="showNoteInput = true"
          >
            Add note
          </button>
          <button
            data-testid="stale-task-close"
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
            @click="resolveWithValue({ status: 'done' })"
          >
            Close
          </button>
        </div>
      </div>

      <!-- Ambiguous Action -->
      <div
        v-else-if="item.kind === ClarificationKind.AmbiguousAction"
        class="flex flex-col gap-2"
      >
        <button
          v-for="(interp, idx) in ((options as AmbiguousActionOptions | null)?.interpretations ?? [])"
          :key="idx"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors text-left"
          @click="resolveWithValue({ selected: idx })"
        >
          {{ typeof interp === 'string' ? interp : JSON.stringify(interp) }}
        </button>
      </div>

      <!-- Ambiguous Deadline -->
      <div
        v-else-if="item.kind === ClarificationKind.AmbiguousDeadline"
        class="flex flex-col gap-2"
      >
        <div
          v-if="ambiguousDeadlineOptions"
          class="bg-gray-700 rounded-lg px-3 py-2 text-sm"
        >
          <p
            v-if="ambiguousDeadlineOptions.description"
            class="text-gray-300 mb-1"
          >
            {{ ambiguousDeadlineOptions.description }}
          </p>
          <p
            v-if="ambiguousDeadlineOptions.raw_date"
            class="text-amber-400"
          >
            <span class="text-gray-500">Parsed as: </span>"{{ ambiguousDeadlineOptions.raw_date }}"
          </p>
        </div>
        <input
          type="datetime-local"
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @change="(e) => resolveWithValue({ due_date: new Date((e.target as HTMLInputElement).value).toISOString() })"
        >
      </div>

      <!-- New Context -->
      <div
        v-else-if="item.kind === ClarificationKind.NewContext"
        class="flex gap-2"
      >
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'confirm' })"
        >
          Confirm
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'merge' })"
        >
          Merge
        </button>
      </div>

      <!-- Context Debrief -->
      <div
        v-else-if="item.kind === ClarificationKind.ContextDebrief"
        class="flex flex-col gap-2"
      >
        <textarea
          v-model="debriefAnswer"
          rows="3"
          placeholder="Your answer..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        />
        <button
          :disabled="!debriefAnswer.trim()"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="resolveDebrief"
        >
          Submit
        </button>
      </div>

      <!-- Task Debrief (importance rating) -->
      <div
        v-else-if="item.kind === ClarificationKind.TaskDebrief"
        class="flex flex-col gap-2"
      >
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="opt in (Array.isArray(options) ? (options as Array<{label: string, value: string}>) : [])"
            :key="opt.value"
            :class="[
              'px-4 py-2.5 text-sm font-medium text-white rounded-lg transition-colors',
              opt.value === 'high' ? 'bg-emerald-600 hover:bg-emerald-500' :
              opt.value === 'medium' ? 'bg-blue-600 hover:bg-blue-500' :
              opt.value === 'low' ? 'bg-amber-600 hover:bg-amber-500' :
              opt.value === 'waste' ? 'bg-red-600 hover:bg-red-500' :
              'bg-gray-600 hover:bg-gray-500 col-span-2'
            ]"
            @click="resolveWithValue({ value: opt.value })"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <!-- Voice Reference -->
      <div
        v-else-if="item.kind === ClarificationKind.VoiceReference"
        class="flex flex-col gap-2"
      >
        <div
          v-if="voiceReferenceOptions"
          class="bg-gray-700 rounded-lg px-3 py-2 text-sm"
        >
          <p
            v-if="voiceReferenceOptions.clause_text"
            class="text-gray-300 mb-1"
          >
            <span class="text-gray-500">In: </span>"{{ voiceReferenceOptions.clause_text }}"
          </p>
          <p class="text-amber-400">
            <span class="text-gray-500">Ambiguous: </span>"{{ voiceReferenceOptions.original_text }}"
          </p>
        </div>
        <input
          type="text"
          placeholder="What did you mean?"
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @keyup.enter="(e) => resolveWithValue({ resolved_text: (e.target as HTMLInputElement).value })"
        >
      </div>

      <!-- Entity Link Clarification -->
      <div
        v-else-if="item.kind === ClarificationKind.EntityLink && entityLinkOptions"
        class="flex flex-col gap-2"
      >
        <div class="bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300">
          <span class="font-medium text-gray-100">{{ entityLinkOptions.source_type }}</span>
          <span class="text-gray-500 mx-2">→</span>
          <span class="font-medium text-gray-100">{{ entityLinkOptions.target_type }}</span>
          <span class="text-gray-500 ml-2">({{ Math.round(entityLinkOptions.confidence * 100) }}% confidence)</span>
        </div>
        <div class="flex gap-2">
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
            @click="resolveWithValue({ confirmed: true })"
          >
            Confirm Link
          </button>
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
            @click="resolveWithValue({ confirmed: false })"
          >
            Reject
          </button>
        </div>
      </div>

      <!-- Weekly Review (multi-select tasks) -->
      <div
        v-else-if="item.kind === ClarificationKind.WeeklyReview"
        class="flex flex-col gap-2"
      >
        <p class="text-sm text-gray-400 mb-1">
          Select the tasks that had the most impact:
        </p>
        <button
          v-for="task in weeklyReviewTasks"
          :key="task.id"
          :class="[
            'w-full px-4 py-2.5 text-sm font-medium text-left rounded-lg transition-colors border',
            selectedWeeklyTasks.has(task.id)
              ? 'bg-emerald-600/20 border-emerald-500 text-emerald-300'
              : 'bg-gray-700 border-gray-600 text-gray-100 hover:border-gray-500'
          ]"
          @click="toggleWeeklyTask(task.id)"
        >
          {{ task.title }}
        </button>
        <button
          :disabled="selectedWeeklyTasks.size === 0"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed mt-1"
          @click="resolveWithValue({ selected_task_ids: [...selectedWeeklyTasks] })"
        >
          Submit ({{ selectedWeeklyTasks.size }} selected)
        </button>
      </div>

      <!-- Type Assignment -->
      <div
        v-else-if="item.kind === ClarificationKind.TypeAssignment && typeAssignmentOptions"
        class="flex flex-col gap-3"
      >
        <div class="bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300">
          <p class="text-xs text-gray-500 mb-1">
            Original text
          </p>
          <p class="text-gray-100">
            {{ typeAssignmentOptions.clause_text }}
          </p>
        </div>
        <p class="text-xs text-gray-400">
          Classified as <span class="text-amber-400 font-medium">{{ typeAssignmentOptions.predicted_type }}</span>
          with {{ Math.round(typeAssignmentOptions.confidence * 100) }}% confidence.
          What is it?
        </p>
        <div class="flex flex-col gap-2">
          <button
            v-for="opt in typeAssignmentOptions.options"
            :key="opt"
            :class="[
              'w-full px-4 py-2.5 text-sm font-medium text-white rounded-lg transition-colors capitalize',
              opt === 'task' ? 'bg-emerald-600 hover:bg-emerald-500' :
              opt === 'note' ? 'bg-violet-600 hover:bg-violet-500' :
              opt === 'event' ? 'bg-blue-600 hover:bg-blue-500' :
              'bg-gray-600 hover:bg-gray-500'
            ]"
            @click="resolveWithValue({ actual_type: opt })"
          >
            {{ opt }}
          </button>
        </div>
      </div>

      <!-- Knowledge Gap -->
      <div
        v-else-if="item.kind === ClarificationKind.KnowledgeGap && knowledgeGapOptions"
        class="flex flex-col gap-2"
      >
        <!-- Collapsible existing knowledge summary -->
        <div
          v-if="knowledgeGapOptions.existing_knowledge_summary"
          class="flex flex-col gap-1"
        >
          <button
            class="text-xs text-blue-400 hover:text-blue-300 underline underline-offset-2 text-left transition-colors"
            @click="showNoteInput = !showNoteInput"
          >
            {{ showNoteInput ? '▼' : '▶' }} What I already know
          </button>
          <p
            v-if="showNoteInput"
            class="text-sm text-gray-400 bg-gray-700/50 rounded px-3 py-2"
          >
            {{ knowledgeGapOptions.existing_knowledge_summary }}
          </p>
        </div>

        <!-- Option chips -->
        <div
          v-if="knowledgeGapOptions.options?.length && knowledgeGapOptions.confidence >= CHIPS_THRESHOLD"
          class="flex flex-col gap-2"
        >
          <p class="text-xs text-gray-400">
            Select an option ({{ Math.round(knowledgeGapOptions.confidence * 100) }}% confidence):
          </p>
          <div class="flex flex-col gap-2">
            <button
              v-for="opt in knowledgeGapOptions.options"
              :key="opt"
              class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
              @click="resolveWithValue({ selected_option: opt })"
            >
              {{ opt }}
            </button>
          </div>
        </div>

        <!-- Answer textarea -->
        <textarea
          v-model="noteText"
          rows="3"
          placeholder="Your answer..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        />

        <!-- Action buttons -->
        <div class="flex gap-2">
          <button
            :disabled="!noteText.trim()"
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            @click="resolveWithValue({ answer_text: noteText.trim(), create_note: true })"
          >
            Save as note
          </button>
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
            @click="resolveWithValue({ dismissed: true })"
          >
            Not useful
          </button>
        </div>
      </div>

      <!-- Event Prep -->
      <div
        v-else-if="item.kind === ClarificationKind.EventPrep && eventPrepOptions"
        class="flex flex-col gap-2"
      >
        <div class="bg-gray-700 rounded-lg px-3 py-2 text-sm">
          <p class="text-gray-300">
            <span class="text-gray-500">Event: </span>{{ eventPrepOptions.event_title }}
          </p>
          <p
            v-if="eventPrepOptions.prep_task_titles?.length"
            class="text-gray-300 mt-1"
          >
            <span class="text-gray-500">Related tasks: </span>{{ eventPrepOptions.prep_task_titles.join(', ') }}
          </p>
        </div>
        <button
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ acknowledged: true })"
        >
          Acknowledge
        </button>
      </div>

      <!-- Ambiguous Entity Match -->
      <div
        v-else-if="item.kind === ClarificationKind.AmbiguousEntityMatch && ambiguousEntityMatchOptions"
        class="flex flex-col gap-2"
      >
        <div class="bg-gray-700 rounded-lg px-3 py-2 text-sm">
          <p class="text-gray-300">
            <span class="text-gray-500">Found {{ ambiguousEntityMatchOptions.candidate_type }}: </span>{{ ambiguousEntityMatchOptions.candidate_title }}
          </p>
          <p class="text-gray-500 text-xs mt-1">
            {{ Math.round(ambiguousEntityMatchOptions.similarity * 100) }}% match confidence
          </p>
        </div>
        <div class="flex gap-2">
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
            @click="resolveWithValue({ choice: 'use_existing', candidate_id: ambiguousEntityMatchOptions.candidate_id })"
          >
            Use existing
          </button>
          <button
            class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
            @click="resolveWithValue({ choice: 'create_new' })"
          >
            Create new
          </button>
        </div>
      </div>

      <!-- Fallback -->
      <div
        v-else
        class="flex gap-2"
      >
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ acknowledged: true })"
        >
          Acknowledge
        </button>
      </div>

      <!-- Free-text override (available for all kinds) -->
      <div class="mt-3">
        <div
          v-if="showFreeTextInput"
          class="flex flex-col gap-2"
        >
          <input
            v-model="freeTextOverride"
            data-testid="free-text-input"
            type="text"
            placeholder="Describe what you meant..."
            class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-purple-500"
            @keyup.enter="freeTextOverride.trim() && resolveWithValue({ free_text: freeTextOverride.trim() })"
          >
          <div class="flex gap-2">
            <button
              data-testid="free-text-submit"
              :disabled="!freeTextOverride.trim()"
              class="flex-1 px-4 py-2 text-sm font-medium text-white bg-purple-600 hover:bg-purple-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              @click="resolveWithValue({ free_text: freeTextOverride.trim() })"
            >
              Submit
            </button>
            <button
              data-testid="free-text-cancel"
              class="px-4 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
              @click="showFreeTextInput = false; freeTextOverride = ''"
            >
              Cancel
            </button>
          </div>
        </div>
        <button
          v-else
          data-testid="free-text-toggle"
          class="text-xs text-gray-500 hover:text-gray-300 underline underline-offset-2 transition-colors"
          @click="showFreeTextInput = true"
        >
          None of these? Type your own
        </button>
      </div>
    </div>

    <!-- Snooze / Dismiss -->
    <div class="flex gap-2 mt-3">
      <button
        :disabled="isCreating"
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        @click="emit('snooze', 24)"
      >
        Snooze 24h
      </button>
      <button
        :disabled="isCreating"
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        @click="emit('dismiss')"
      >
        Dismiss
      </button>
    </div>
  </div>
</template>
