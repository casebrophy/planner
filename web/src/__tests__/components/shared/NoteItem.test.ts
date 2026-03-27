import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import NoteItem from '@/components/shared/NoteItem.vue'

function makeEntry(overrides: Record<string, unknown> = {}) {
  return {
    id: 'entry-1',
    subjectType: 'task',
    subjectId: 'task-1',
    kind: 'note',
    content: 'This is a test note',
    source: 'user',
    requiresAction: false,
    createdAt: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    ...overrides,
  }
}

describe('NoteItem', () => {
  it('renders content text', () => {
    const entry = makeEntry({ content: 'Hello world' })
    const wrapper = mount(NoteItem, { props: { entry } })

    expect(wrapper.text()).toContain('Hello world')
  })

  it('shows source label', () => {
    const entry = makeEntry({ source: 'claude' })
    const wrapper = mount(NoteItem, { props: { entry } })

    const sourceLabel = wrapper.find('.capitalize')
    expect(sourceLabel.text()).toBe('claude')
  })

  it('shows relative timestamp', () => {
    const entry = makeEntry()
    const wrapper = mount(NoteItem, { props: { entry } })

    // date-fns formatDistanceToNow should produce something like "about 1 hour ago"
    expect(wrapper.text()).toMatch(/hour ago|hours ago|minutes ago/)
  })

  it('shows "Action needed" badge when requiresAction is true', () => {
    const entry = makeEntry({ requiresAction: true })
    const wrapper = mount(NoteItem, { props: { entry } })

    expect(wrapper.text()).toContain('Action needed')
  })

  it('does not show "Action needed" when requiresAction is false', () => {
    const entry = makeEntry({ requiresAction: false })
    const wrapper = mount(NoteItem, { props: { entry } })

    expect(wrapper.text()).not.toContain('Action needed')
  })

  it('shows user icon for source=user', () => {
    const entry = makeEntry({ source: 'user' })
    const wrapper = mount(NoteItem, { props: { entry } })

    const icon = wrapper.find('.text-blue-400')
    expect(icon.exists()).toBe(true)
  })

  it('shows claude icon for source=claude', () => {
    const entry = makeEntry({ source: 'claude' })
    const wrapper = mount(NoteItem, { props: { entry } })

    const icon = wrapper.find('.text-purple-400')
    expect(icon.exists()).toBe(true)
  })

  it('shows system icon for source=system', () => {
    const entry = makeEntry({ source: 'system' })
    const wrapper = mount(NoteItem, { props: { entry } })

    const icons = wrapper.findAll('.text-gray-500')
    expect(icons.length).toBeGreaterThan(0)
  })

  it('shows email icon for source=email', () => {
    const entry = makeEntry({ source: 'email' })
    const wrapper = mount(NoteItem, { props: { entry } })

    const icon = wrapper.find('.text-amber-400')
    expect(icon.exists()).toBe(true)
  })

  it('shows sentiment when present', () => {
    const entry = makeEntry({ sentiment: 'positive' })
    const wrapper = mount(NoteItem, { props: { entry } })

    expect(wrapper.text()).toContain('positive')
  })
})
