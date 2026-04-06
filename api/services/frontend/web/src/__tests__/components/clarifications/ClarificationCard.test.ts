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
