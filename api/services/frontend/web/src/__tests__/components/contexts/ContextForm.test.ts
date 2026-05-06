import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import ContextForm from '@/components/contexts/ContextForm.vue'
import { makeContext } from '../../helpers/testFactories'
import { ContextKind, ContextStatus } from '@/types'
import { useContextStore } from '@/stores/contextStore'

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

function mountForm(props: Record<string, unknown> & { mode: 'create' | 'edit' }) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(ContextForm, {
    props,
    global: { plugins: [pinia] },
  })
  return { wrapper, pinia }
}

describe('ContextForm', () => {
  it('renders create mode', () => {
    const { wrapper } = mountForm({ mode: 'create' })
    expect(wrapper.find('button[type="submit"]').text()).toBe('Create Context')
  })

  it('renders edit mode with populated fields', () => {
    const ctx = makeContext({ title: 'My Context' })
    const { wrapper } = mountForm({ mode: 'edit', context: ctx })
    expect(wrapper.find('button[type="submit"]').text()).toBe('Save Changes')
  })

  it('shows status field only in edit mode', () => {
    const { wrapper: createWrapper } = mountForm({ mode: 'create' })
    const { wrapper: editWrapper } = mountForm({ mode: 'edit', context: makeContext() })
    // create mode: 1 select (kind). edit mode: 2 selects (kind + status)
    expect(createWrapper.findAll('select').length).toBe(1)
    expect(editWrapper.findAll('select').length).toBe(2)
  })

  it('emits submit with NewContext in create mode', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    await wrapper.find('input').setValue('New Context')
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toMatchObject({ title: 'New Context' })
  })

  it('emits cancel when cancel clicked', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    const cancelBtn = wrapper.findAll('button').find(b => b.text() === 'Cancel')!
    await cancelBtn.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('kind selector contains a List option', () => {
    const { wrapper } = mountForm({ mode: 'create' })
    const kindSelect = wrapper.find('select')
    const options = kindSelect.findAll('option')
    const values = options.map(o => o.element.value)
    expect(values).toContain(ContextKind.List)
  })

  it('area selector is hidden when kind is not list', () => {
    const { wrapper } = mountForm({ mode: 'create' })
    // default kind is Project — only the kind select should be present
    expect(wrapper.findAll('select').length).toBe(1)
  })

  it('area selector appears when kind is changed to list', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    const store = useContextStore()
    const area = makeContext({ kind: ContextKind.Area, status: ContextStatus.Active })
    store.items = [area]
    await nextTick()

    const kindSelect = wrapper.find('select')
    await kindSelect.setValue(ContextKind.List)
    // now kind + area selects
    expect(wrapper.findAll('select').length).toBe(2)
  })

  it('emits kind=list when list kind is selected and submitted', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    await wrapper.find('input').setValue('My List')
    await wrapper.find('select').setValue(ContextKind.List)
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect((emitted![0]![0] as Record<string, unknown>).kind).toBe(ContextKind.List)
  })

  it('emits parentContextId from area selector when kind=list and area is selected', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    const store = useContextStore()
    const area = makeContext({ kind: ContextKind.Area, status: ContextStatus.Active })
    store.items = [area]
    await nextTick()

    await wrapper.find('input').setValue('My List')
    // select kind=list first
    await wrapper.find('select').setValue(ContextKind.List)
    await nextTick()

    // now find the area select (second select) and set the area id
    const selects = wrapper.findAll('select')
    await selects[1]!.setValue(area.id)
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect((emitted![0]![0] as Record<string, unknown>).parentContextId).toBe(area.id)
  })

  it('does not emit parentContextId when kind=list and no area selected', async () => {
    const { wrapper } = mountForm({ mode: 'create' })
    await wrapper.find('input').setValue('My List')
    await wrapper.find('select').setValue(ContextKind.List)
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect((emitted![0]![0] as Record<string, unknown>).parentContextId).toBeUndefined()
  })
})
