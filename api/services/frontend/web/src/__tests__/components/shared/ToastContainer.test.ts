import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ToastContainer from '@/components/shared/ToastContainer.vue'
import { useToastStore } from '@/stores/toastStore'

describe('ToastContainer', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders nothing when there are no toasts', () => {
    const wrapper = mount(ToastContainer)
    expect(wrapper.findAll('[class*="px-4"]')).toHaveLength(0)
  })

  it('renders a success toast with the correct message', async () => {
    const store = useToastStore()
    store.success('Task saved')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Task saved')
  })

  it('renders an error toast with the correct message', async () => {
    const store = useToastStore()
    store.error('Something went wrong')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Something went wrong')
  })

  it('applies green background class for success toasts', async () => {
    const store = useToastStore()
    store.success('Done')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    const toast = wrapper.find('[class*="bg-green"]')
    expect(toast.exists()).toBe(true)
  })

  it('applies red background class for error toasts', async () => {
    const store = useToastStore()
    store.error('Failed')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    const toast = wrapper.find('[class*="bg-red"]')
    expect(toast.exists()).toBe(true)
  })

  it('applies blue background class for info toasts', async () => {
    const store = useToastStore()
    store.info('FYI')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    const toast = wrapper.find('[class*="bg-blue"]')
    expect(toast.exists()).toBe(true)
  })

  it('dismisses a toast when clicked', async () => {
    const store = useToastStore()
    store.success('Click me to dismiss')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Click me to dismiss')

    const toast = wrapper.find('.cursor-pointer')
    await toast.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).not.toContain('Click me to dismiss')
  })

  it('renders multiple toasts simultaneously', async () => {
    const store = useToastStore()
    store.success('First')
    store.error('Second')
    store.info('Third')

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('First')
    expect(wrapper.text()).toContain('Second')
    expect(wrapper.text()).toContain('Third')
  })

  it('auto-dismisses a toast after its duration', async () => {
    const store = useToastStore()
    store.success('Temporary', 1000)

    const wrapper = mount(ToastContainer)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Temporary')

    vi.advanceTimersByTime(1100)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).not.toContain('Temporary')
  })
})
