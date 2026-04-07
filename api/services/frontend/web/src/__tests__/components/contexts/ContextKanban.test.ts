import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ContextKanban from '@/components/contexts/ContextKanban.vue'
import { makeContext } from '../../helpers/testFactories'
import { ContextKind } from '@/types'

describe('ContextKanban', () => {
  const columns = {
    project: [makeContext({ kind: ContextKind.Project }), makeContext({ kind: ContextKind.Project })],
    area: [makeContext({ kind: ContextKind.Area })],
  }

  it('renders two columns: Projects and Areas', () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    expect(wrapper.text()).toContain('Projects')
    expect(wrapper.text()).toContain('Areas')
  })

  it('renders correct count per column', () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    expect(wrapper.text()).toContain('(2)')
    expect(wrapper.text()).toContain('(1)')
  })

  it('shows empty state text for empty columns', () => {
    const emptyColumns = { project: [], area: [] }
    const wrapper = mount(ContextKanban, { props: { columns: emptyColumns } })
    expect(wrapper.text()).toContain('No projects yet')
    expect(wrapper.text()).toContain('No areas yet')
  })

  it('emits select when a context card is clicked', async () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    const firstCard = wrapper.findComponent({ name: 'ContextCard' })
    await firstCard.trigger('click')
    expect(wrapper.emitted('select')).toBeTruthy()
  })
})
