import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'

vi.mock('@/stores/clarificationStore', () => ({
  useClarificationStore: () => ({
    fetchPendingCount: vi.fn(),
    pendingCount: 0,
  }),
}))

function setInnerWidth(width: number) {
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: width,
  })
}

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ template: '<div>Home</div>' }) },
      { path: '/dashboard', component: defineComponent({ template: '<div>Dashboard</div>' }) },
      { path: '/capture', component: defineComponent({ template: '<div>Capture</div>' }) },
    ],
  })
}

async function mountShell() {
  const router = createTestRouter()
  await router.push('/')
  await router.isReady()
  return mount(AppShell, {
    global: { plugins: [createPinia(), router] },
  })
}

describe('AppShell', () => {
  let wrapper: ReturnType<typeof mount> | null = null

  beforeEach(() => {
    // Reset to a neutral width before each test
    setInnerWidth(1024)
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
      wrapper = null
    }
  })

  it('renders sidebar on desktop', async () => {
    setInnerWidth(1024)
    wrapper = await mountShell()
    expect(wrapper.find('aside').exists()).toBe(true)
  })

  it('does not render tab bar on desktop', async () => {
    setInnerWidth(1024)
    wrapper = await mountShell()
    const navs = wrapper.findAll('nav')
    const tabBar = navs.find(n => n.classes().includes('fixed') && n.classes().includes('bottom-0'))
    expect(tabBar).toBeUndefined()
  })

  it('renders tab bar on mobile', async () => {
    setInnerWidth(375)
    wrapper = await mountShell()
    expect(wrapper.find('nav').exists()).toBe(true)
  })

  it('does not render sidebar on mobile', async () => {
    setInnerWidth(375)
    wrapper = await mountShell()
    expect(wrapper.find('aside').exists()).toBe(false)
  })

  it('mobile main has bottom padding', async () => {
    setInnerWidth(375)
    wrapper = await mountShell()
    const main = wrapper.find('main')
    expect(main.classes()).toContain('pb-14')
  })
})
