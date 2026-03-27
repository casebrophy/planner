import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TagPicker from '@/components/tags/TagPicker.vue'
import { useTagStore } from '@/stores/tagStore'
import { makeTag } from '../../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/tagService', () => ({
  tagService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getByTask: vi.fn(),
    addToTask: vi.fn(),
    removeFromTask: vi.fn(),
    getByContext: vi.fn(),
    addToContext: vi.fn(),
    removeFromContext: vi.fn(),
  },
}))

describe('TagPicker', () => {
  function mountPicker(selectedIds: string[] = []) {
    const pinia = createPinia()
    setActivePinia(pinia)
    return mount(TagPicker, {
      props: { selectedIds },
      global: { plugins: [pinia] },
    })
  }

  it('renders search input', () => {
    const wrapper = mountPicker()
    expect(wrapper.find('input').exists()).toBe(true)
  })

  it('filters tags by search term', async () => {
    const { nextTick } = await import('vue')
    const pinia = createPinia()
    setActivePinia(pinia)

    const wrapper = mount(TagPicker, {
      props: { selectedIds: [] },
      global: { plugins: [pinia] },
    })

    // Wait for onMounted fetchList to complete, then override items
    await nextTick()
    await nextTick()

    const tagStore = useTagStore()
    tagStore.items = [makeTag({ name: 'alpha' }), makeTag({ name: 'beta' })]

    await wrapper.find('input').trigger('focus')
    await wrapper.find('input').setValue('alp')
    await nextTick()

    // dropdown should show only 'alpha'
    expect(wrapper.text()).toContain('alpha')
    expect(wrapper.text()).not.toContain('beta')
  })
})
