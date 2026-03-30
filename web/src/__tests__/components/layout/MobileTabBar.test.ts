import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import MobileTabBar from '@/components/layout/MobileTabBar.vue'

vi.mock('@/stores/clarificationStore', () => ({
  useClarificationStore: () => ({
    fetchPendingCount: vi.fn(),
    pendingCount: 0,
  }),
}))

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/capture', component: defineComponent({ template: '<div />' }) },
      { path: '/today', component: defineComponent({ template: '<div />' }) },
      { path: '/contexts', component: defineComponent({ template: '<div />' }) },
      { path: '/search', component: defineComponent({ template: '<div />' }) },
      { path: '/settings', component: defineComponent({ template: '<div />' }) },
    ],
  })
}

async function mountAt(path: string) {
  const router = createTestRouter()
  await router.push(path)
  await router.isReady()
  return mount(MobileTabBar, {
    global: { plugins: [createPinia(), router] },
  })
}

describe('MobileTabBar', () => {
  it('renders all 5 tab labels', async () => {
    const wrapper = await mountAt('/capture')
    const text = wrapper.text()
    expect(text).toContain('Capture')
    expect(text).toContain('Today')
    expect(text).toContain('Contexts')
    expect(text).toContain('Search')
    expect(text).toContain('Settings')
    wrapper.unmount()
  })

  it('renders correct number of router-links', async () => {
    const wrapper = await mountAt('/capture')
    expect(wrapper.findAll('a')).toHaveLength(5)
    wrapper.unmount()
  })

  it('highlights active tab', async () => {
    const wrapper = await mountAt('/capture')
    const links = wrapper.findAll('a')
    const captureLink = links.find(l => l.text().includes('Capture'))!
    expect(captureLink.classes()).toContain('text-gray-100')
    wrapper.unmount()
  })

  it('inactive tabs have muted styling', async () => {
    const wrapper = await mountAt('/capture')
    const links = wrapper.findAll('a')
    const inactiveLinks = links.filter(l => !l.text().includes('Capture'))
    for (const link of inactiveLinks) {
      expect(link.classes()).toContain('text-gray-500')
    }
    wrapper.unmount()
  })
})
