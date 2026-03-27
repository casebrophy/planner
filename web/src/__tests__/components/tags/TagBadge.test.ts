import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TagBadge from '@/components/tags/TagBadge.vue'
import { makeTag } from '../../helpers/testFactories'

describe('TagBadge', () => {
  it('renders tag name', () => {
    const tag = makeTag({ name: 'urgent' })
    const wrapper = mount(TagBadge, { props: { tag } })
    expect(wrapper.text()).toContain('urgent')
  })

  it('shows remove button when removable', () => {
    const tag = makeTag()
    const wrapper = mount(TagBadge, { props: { tag, removable: true } })
    expect(wrapper.find('button').exists()).toBe(true)
  })

  it('hides remove button when not removable', () => {
    const tag = makeTag()
    const wrapper = mount(TagBadge, { props: { tag } })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('emits remove with tag id when remove clicked', async () => {
    const tag = makeTag()
    const wrapper = mount(TagBadge, { props: { tag, removable: true } })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('remove')).toEqual([[tag.id]])
  })
})
