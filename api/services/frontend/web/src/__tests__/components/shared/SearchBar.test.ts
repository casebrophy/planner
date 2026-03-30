import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import SearchBar from '@/components/shared/SearchBar.vue'

describe('SearchBar', () => {
  it('renders input with placeholder', () => {
    const wrapper = mount(SearchBar, {
      props: { modelValue: '' },
    })
    const input = wrapper.find('input')
    expect(input.exists()).toBe(true)
    expect(input.attributes('placeholder')).toBe('Search tasks, contexts, and tags...')
  })

  it('renders with custom placeholder', () => {
    const wrapper = mount(SearchBar, {
      props: { modelValue: '', placeholder: 'Find something...' },
    })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Find something...')
  })

  it('emits update:modelValue on input with debounce', async () => {
    vi.useFakeTimers()
    const wrapper = mount(SearchBar, {
      props: { modelValue: '', debounceMs: 100 },
    })

    await wrapper.find('input').setValue('hello')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    vi.advanceTimersByTime(100)
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['hello'])

    vi.useRealTimers()
  })

  it('shows clear button when value is non-empty', async () => {
    const wrapper = mount(SearchBar, {
      props: { modelValue: 'test' },
    })

    // Internal value syncs from prop
    const clearBtn = wrapper.find('button[aria-label="Clear search"]')
    expect(clearBtn.exists()).toBe(true)
  })

  it('does not show clear button when value is empty', () => {
    const wrapper = mount(SearchBar, {
      props: { modelValue: '' },
    })
    const clearBtn = wrapper.find('button[aria-label="Clear search"]')
    expect(clearBtn.exists()).toBe(false)
  })

  it('clear button resets value', async () => {
    vi.useFakeTimers()
    const wrapper = mount(SearchBar, {
      props: { modelValue: 'something' },
    })

    const clearBtn = wrapper.find('button[aria-label="Clear search"]')
    await clearBtn.trigger('click')

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([''])
    vi.useRealTimers()
  })
})
