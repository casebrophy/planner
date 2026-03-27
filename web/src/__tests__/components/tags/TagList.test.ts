import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TagList from '@/components/tags/TagList.vue'
import { makeTag } from '../../helpers/testFactories'

describe('TagList', () => {
  it('renders all tags', () => {
    const tags = [makeTag({ name: 'a' }), makeTag({ name: 'b' })]
    const wrapper = mount(TagList, { props: { tags } })
    expect(wrapper.text()).toContain('a')
    expect(wrapper.text()).toContain('b')
  })

  it('shows "No tags" when empty', () => {
    const wrapper = mount(TagList, { props: { tags: [] } })
    expect(wrapper.text()).toContain('No tags')
  })

  it('emits remove when a tag remove button is clicked', async () => {
    const tag = makeTag()
    const wrapper = mount(TagList, { props: { tags: [tag], removable: true } })
    const badge = wrapper.findComponent({ name: 'TagBadge' })
    await badge.find('button').trigger('click')
    expect(wrapper.emitted('remove')).toEqual([[tag.id]])
  })
})
