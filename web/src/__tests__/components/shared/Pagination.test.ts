import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Pagination from '@/components/shared/Pagination.vue'

describe('Pagination', () => {
  it('renders page info', () => {
    const wrapper = mount(Pagination, {
      props: { page: 2, totalPages: 5, hasNext: true, hasPrev: true },
    })
    expect(wrapper.text()).toContain('Page 2 of 5')
  })

  it('emits next when next button clicked', async () => {
    const wrapper = mount(Pagination, {
      props: { page: 1, totalPages: 3, hasNext: true, hasPrev: false },
    })
    const buttons = wrapper.findAll('button')
    await buttons[1]!.trigger('click')
    expect(wrapper.emitted('next')).toHaveLength(1)
  })

  it('emits prev when previous button clicked', async () => {
    const wrapper = mount(Pagination, {
      props: { page: 2, totalPages: 3, hasNext: true, hasPrev: true },
    })
    const buttons = wrapper.findAll('button')
    await buttons[0]!.trigger('click')
    expect(wrapper.emitted('prev')).toHaveLength(1)
  })

  it('disables previous button on first page', () => {
    const wrapper = mount(Pagination, {
      props: { page: 1, totalPages: 3, hasNext: true, hasPrev: false },
    })
    const prevBtn = wrapper.findAll('button')[0]!
    expect(prevBtn.attributes('disabled')).toBeDefined()
  })

  it('disables next button on last page', () => {
    const wrapper = mount(Pagination, {
      props: { page: 3, totalPages: 3, hasNext: false, hasPrev: true },
    })
    const nextBtn = wrapper.findAll('button')[1]!
    expect(nextBtn.attributes('disabled')).toBeDefined()
  })
})
