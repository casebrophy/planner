import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'

describe('ConfirmDialog', () => {
  const baseProps = {
    open: true,
    title: 'Delete Item',
    message: 'Are you sure?',
  }

  it('renders nothing when closed', () => {
    const wrapper = mount(ConfirmDialog, {
      props: { ...baseProps, open: false },
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.find('.fixed').exists()).toBe(false)
  })

  it('renders title and message when open', () => {
    const wrapper = mount(ConfirmDialog, {
      props: baseProps,
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.text()).toContain('Delete Item')
    expect(wrapper.text()).toContain('Are you sure?')
  })

  it('emits confirm when confirm button clicked', async () => {
    const wrapper = mount(ConfirmDialog, {
      props: baseProps,
      global: { stubs: { Teleport: true } },
    })
    const buttons = wrapper.findAll('button')
    const confirmBtn = buttons.find(b => b.text() === 'Delete')!
    await confirmBtn.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('emits cancel when cancel button clicked', async () => {
    const wrapper = mount(ConfirmDialog, {
      props: baseProps,
      global: { stubs: { Teleport: true } },
    })
    const buttons = wrapper.findAll('button')
    const cancelBtn = buttons.find(b => b.text() === 'Cancel')!
    await cancelBtn.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('uses custom button labels', () => {
    const wrapper = mount(ConfirmDialog, {
      props: { ...baseProps, confirmLabel: 'Yes', cancelLabel: 'No' },
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.text()).toContain('Yes')
    expect(wrapper.text()).toContain('No')
  })
})
