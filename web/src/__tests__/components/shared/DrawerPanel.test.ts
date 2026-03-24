import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'

describe('DrawerPanel', () => {
  it('renders nothing when closed', () => {
    const wrapper = mount(DrawerPanel, {
      props: { open: false },
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.find('.fixed').exists()).toBe(false)
  })

  it('renders title and slot content when open', () => {
    const wrapper = mount(DrawerPanel, {
      props: { open: true, title: 'My Drawer' },
      slots: { default: '<p>Content</p>' },
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.text()).toContain('My Drawer')
    expect(wrapper.text()).toContain('Content')
  })

  it('emits close when close button clicked', async () => {
    const wrapper = mount(DrawerPanel, {
      props: { open: true, title: 'Test' },
      global: { stubs: { Teleport: true } },
    })
    const closeBtn = wrapper.find('button')
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('emits close when backdrop clicked', async () => {
    const wrapper = mount(DrawerPanel, {
      props: { open: true, title: 'Test' },
      global: { stubs: { Teleport: true } },
    })
    const backdrop = wrapper.find('.bg-black\\/40')
    await backdrop.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
