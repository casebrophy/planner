import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ContextForm from '@/components/contexts/ContextForm.vue'
import { makeContext } from '../../helpers/testFactories'

describe('ContextForm', () => {
  it('renders create mode', () => {
    const wrapper = mount(ContextForm, { props: { mode: 'create' } })
    expect(wrapper.find('button[type="submit"]').text()).toBe('Create Context')
  })

  it('renders edit mode with populated fields', () => {
    const ctx = makeContext({ title: 'My Context' })
    const wrapper = mount(ContextForm, { props: { mode: 'edit', context: ctx } })
    expect(wrapper.find('button[type="submit"]').text()).toBe('Save Changes')
  })

  it('shows status and summary fields only in edit mode', () => {
    const createWrapper = mount(ContextForm, { props: { mode: 'create' } })
    const editWrapper = mount(ContextForm, { props: { mode: 'edit', context: makeContext() } })
    expect(createWrapper.findAll('select').length).toBe(0)
    expect(editWrapper.findAll('select').length).toBe(1)
  })

  it('emits submit with NewContext in create mode', async () => {
    const wrapper = mount(ContextForm, { props: { mode: 'create' } })
    await wrapper.find('input').setValue('New Context')
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toMatchObject({ title: 'New Context' })
  })

  it('emits cancel when cancel clicked', async () => {
    const wrapper = mount(ContextForm, { props: { mode: 'create' } })
    const cancelBtn = wrapper.findAll('button').find(b => b.text() === 'Cancel')!
    await cancelBtn.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})
