import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ClarificationCard from '@/components/clarifications/ClarificationCard.vue'
import { makeClarificationItem } from '../../helpers/testFactories'
import { ClarificationKind } from '@/types/enums'
import type { Context } from '@/types'

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

  it('disables existing context buttons and snooze while creating', async () => {
    let resolveCreate!: (v: Context) => void
    vi.mocked(contextService.create).mockReturnValue(
      new Promise<Context>(res => { resolveCreate = res })
    )
    const wrapper = mount(ClarificationCard, {
      props: { item: makeContextAssignmentItem() },
    })
    await wrapper.find('[data-testid="new-context-title"]').setValue('In Flight')
    wrapper.find('[data-testid="create-context-btn"]').trigger('click')
    await nextTick()

    const confirmBtn = wrapper.findAll('button').find(b => b.text().includes('Confirm'))
    expect(confirmBtn?.attributes('disabled')).toBeDefined()
    const snoozeBtn = wrapper.findAll('button').find(b => b.text().includes('Snooze'))
    expect(snoozeBtn?.attributes('disabled')).toBeDefined()

    resolveCreate({ id: 'x', title: 'In Flight', description: '', kind: 'project', status: 'active', summary: '', createdAt: '', updatedAt: '' })
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

describe('ClarificationCard — free-text override', () => {
  it('shows toggle link by default and hides input', () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeClarificationItem({ kind: ClarificationKind.ContextAssignment, answerOptions: null }) },
    })
    expect(wrapper.find('[data-testid="free-text-toggle"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="free-text-input"]').exists()).toBe(false)
  })

  it('shows input when toggle is clicked', async () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeClarificationItem({ kind: ClarificationKind.ContextAssignment, answerOptions: null }) },
    })
    await wrapper.find('[data-testid="free-text-toggle"]').trigger('click')
    expect(wrapper.find('[data-testid="free-text-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="free-text-toggle"]').exists()).toBe(false)
  })

  it('submit button is disabled when input is empty', async () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeClarificationItem({ kind: ClarificationKind.ContextAssignment, answerOptions: null }) },
    })
    await wrapper.find('[data-testid="free-text-toggle"]').trigger('click')
    expect(wrapper.find('[data-testid="free-text-submit"]').attributes('disabled')).toBeDefined()
  })

  it('emits resolve with free_text on submit', async () => {
    const wrapper = mount(ClarificationCard, {
      props: { item: makeClarificationItem({ kind: ClarificationKind.ContextAssignment, answerOptions: null }) },
    })
    await wrapper.find('[data-testid="free-text-toggle"]').trigger('click')
    await wrapper.find('[data-testid="free-text-input"]').setValue('treat leather on shoes')
    await wrapper.find('[data-testid="free-text-submit"]').trigger('click')
    expect(wrapper.emitted('resolve')).toEqual([[{ free_text: 'treat leather on shoes' }]])
  })

  it('resets state when item prop changes', async () => {
    const item1 = makeClarificationItem({ kind: ClarificationKind.ContextAssignment, answerOptions: null })
    const item2 = makeClarificationItem({ kind: ClarificationKind.AmbiguousAction, answerOptions: null })
    const wrapper = mount(ClarificationCard, {
      props: { item: item1 },
    })
    await wrapper.find('[data-testid="free-text-toggle"]').trigger('click')
    await wrapper.find('[data-testid="free-text-input"]').setValue('some text')
    expect(wrapper.find('[data-testid="free-text-input"]').exists()).toBe(true)

    await wrapper.setProps({ item: item2 })
    await nextTick()
    expect(wrapper.find('[data-testid="free-text-toggle"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="free-text-input"]').exists()).toBe(false)
  })
})
