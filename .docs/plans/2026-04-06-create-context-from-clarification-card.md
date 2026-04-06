# Create New Context from Clarification Card — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an "Or create new" section to the `context_assignment` clarification card so the user can create a new context and resolve the clarification in one action.

**Architecture:** Purely frontend change. Two sequential API calls: `contextService.create()` to create the context, then `resolveWithValue({ context_id })` to resolve the clarification. All new state is local to `ClarificationCard.vue`. No backend changes required.

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), Vitest + `@vue/test-utils`, Tailwind CSS, TypeScript.

---

## File Map

| File | Change |
|---|---|
| `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue` | Add new context state, `createAndResolve` function, and "Or create new" UI section |
| `api/services/frontend/web/src/__tests__/components/clarifications/ClarificationCard.test.ts` | **Create** — component tests for the new feature |

---

## Task 1: Write the failing component test

**Files:**
- Create: `api/services/frontend/web/src/__tests__/components/clarifications/ClarificationCard.test.ts`

- [ ] **Step 1: Create the test file**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ClarificationCard from '@/components/clarifications/ClarificationCard.vue'
import { makeClarificationItem } from '../../helpers/testFactories'
import { ClarificationKind } from '@/types/enums'

// Mock contextService so no real HTTP calls are made
vi.mock('@/services/contextService', () => ({
  contextService: {
    create: vi.fn(),
  },
}))

import { contextService } from '@/services/contextService'

function makeContextAssignmentItem() {
  return makeClarificationItem({
    kind: ClarificationKind.ContextAssignment,
    answerOptions: {
      suggested_context: 'ctx-1',
      confidence: 0.9,
      available_contexts: [
        { id: 'ctx-1', title: 'Home Renovation' },
        { id: 'ctx-2', title: 'Work Projects' },
      ],
    },
  })
}

describe('ClarificationCard — context_assignment', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the "Or create new" section', () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })
    expect(wrapper.text()).toContain('Or create new')
  })

  it('disables the + button when title is empty', () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })
    const btn = wrapper.find('[data-testid="create-context-btn"]')
    expect(btn.attributes('disabled')).toBeDefined()
  })

  it('creates context and emits resolve with new context id', async () => {
    const newCtxId = 'new-ctx-99'
    vi.mocked(contextService.create).mockResolvedValue({
      id: newCtxId,
      title: 'New Thing',
      description: '',
      kind: 'project',
      status: 'active',
      summary: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    })

    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })

    await wrapper.find('[data-testid="new-context-title"]').setValue('New Thing')
    // kind defaults to 'project' — no need to change it
    await wrapper.find('[data-testid="create-context-btn"]').trigger('click')
    await vi.waitUntil(() => wrapper.emitted('resolve'))

    expect(contextService.create).toHaveBeenCalledWith({
      title: 'New Thing',
      description: '',
      kind: 'project',
    })
    expect(wrapper.emitted('resolve')).toEqual([[{ context_id: newCtxId }]])
  })

  it('selects Area kind when toggled', async () => {
    const newCtxId = 'new-ctx-100'
    vi.mocked(contextService.create).mockResolvedValue({
      id: newCtxId,
      title: 'My Area',
      description: '',
      kind: 'area',
      status: 'active',
      summary: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    })

    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })

    await wrapper.find('[data-testid="new-context-title"]').setValue('My Area')
    await wrapper.find('[data-testid="kind-area"]').trigger('click')
    await wrapper.find('[data-testid="create-context-btn"]').trigger('click')
    await vi.waitUntil(() => wrapper.emitted('resolve'))

    expect(contextService.create).toHaveBeenCalledWith({
      title: 'My Area',
      description: '',
      kind: 'area',
    })
  })

  it('shows inline error and re-enables buttons on create failure', async () => {
    vi.mocked(contextService.create).mockRejectedValue(new Error('Network error'))

    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })

    await wrapper.find('[data-testid="new-context-title"]').setValue('Fail Context')
    await wrapper.find('[data-testid="create-context-btn"]').trigger('click')
    await vi.waitUntil(() => wrapper.find('[data-testid="create-error"]').exists())

    expect(wrapper.find('[data-testid="create-error"]').text()).toContain('Failed to create')
    expect(wrapper.find('[data-testid="create-context-btn"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.emitted('resolve')).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd api/services/frontend/web && npx vitest run src/__tests__/components/clarifications/ClarificationCard.test.ts
```

Expected: multiple failures — `data-testid` attributes not found, "Or create new" text missing.

---

## Task 2: Implement ClarificationCard changes

**Files:**
- Modify: `api/services/frontend/web/src/components/clarifications/ClarificationCard.vue`

- [ ] **Step 1: Add new imports and state to `<script setup>`**

Replace the existing import block (lines 1–7) and add new refs after line 21 (`const noteText = ref('')`):

```typescript
// Add to imports at top of <script setup>
import { contextService } from '@/services/contextService'
import { ContextKind } from '@/types/enums'

// Add after existing refs (after: const noteText = ref(''))
const newContextTitle = ref('')
const newContextKind = ref<ContextKind>(ContextKind.Project)
const isCreating = ref(false)
const createError = ref<string | null>(null)
```

The full imports section becomes:

```typescript
import { ref, computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { ClarificationKind, ClarificationKindLabels, ClarificationKindColors, ContextKind } from '@/types/enums'
import type { ClarificationItem } from '@/types'
import type { ContextAssignmentOptions, AmbiguousActionOptions, ContextRef } from '@/types/generated/clarification-options'
import type { ClarificationAnswerOptions } from '@/types/clarification'
import { contextService } from '@/services/contextService'
```

- [ ] **Step 2: Add `createAndResolve` function**

Add this function after the existing `resolveDebrief` function (after line 55, before `</script>`):

```typescript
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
```

- [ ] **Step 3: Update the context_assignment template block**

Replace the existing context assignment `<div>` block (lines 90–109 in the template) with this expanded version that adds the "Or create new" section:

```html
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
  <p class="text-xs uppercase tracking-wide text-gray-500 mt-1">Or create new</p>
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
```

- [ ] **Step 4: Disable snooze/dismiss while creating**

In the Snooze / Dismiss section (lines 281–294), add `:disabled="isCreating"` to both buttons:

```html
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
```

---

## Task 3: Run tests and commit

- [ ] **Step 1: Run the component tests**

```bash
cd api/services/frontend/web && npx vitest run src/__tests__/components/clarifications/ClarificationCard.test.ts
```

Expected: all 5 tests pass.

- [ ] **Step 2: Run the full frontend test suite**

```bash
cd api/services/frontend/web && npx vitest run
```

Expected: all tests pass, no regressions.

- [ ] **Step 3: Build to verify TypeScript compiles**

```bash
cd api/services/frontend/web && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/components/clarifications/ClarificationCard.vue \
        api/services/frontend/web/src/__tests__/components/clarifications/ClarificationCard.test.ts
git commit -m "feat: add create-new-context option to context assignment clarification card"
```
