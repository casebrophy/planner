import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import SettingsView from '@/views/SettingsView.vue'
import { useSettingsStore } from '@/stores/settingsStore'
import { nextTick } from 'vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/', component: { template: '<div />' } }],
})

function mountView() {
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(SettingsView, {
    global: { plugins: [pinia, router] },
  })
}

describe('SettingsView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders all setting sections', () => {
    const wrapper = mountView()
    const headings = wrapper.findAll('h2')
    const headingTexts = headings.map((h) => h.text())

    expect(headingTexts).toContain('API Configuration')
    expect(headingTexts).toContain('Display Preferences')
    expect(headingTexts).toContain('About')
  })

  it('input fields display current values', () => {
    const wrapper = mountView()
    const store = useSettingsStore()

    const rowsInput = wrapper.find('#rows-per-page')
    expect((rowsInput.element as HTMLInputElement).value).toBe(String(store.rowsPerPage))
  })

  it('changing input updates store', async () => {
    const wrapper = mountView()
    const store = useSettingsStore()

    const rowsInput = wrapper.find('#rows-per-page')
    await rowsInput.setValue(50)

    expect(store.rowsPerPage).toBe(50)
  })

  it('save button shows "Saved!" flash', async () => {
    const wrapper = mountView()

    expect(wrapper.text()).not.toContain('Saved!')

    const saveButton = wrapper.findAll('button').find((b) => b.text() === 'Save')!
    await saveButton.trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('Saved!')

    vi.advanceTimersByTime(2100)
    await nextTick()

    expect(wrapper.text()).not.toContain('Saved!')
  })

  it('reset button restores defaults', async () => {
    const wrapper = mountView()
    const store = useSettingsStore()
    store.rowsPerPage = 99
    await nextTick()

    const resetButton = wrapper
      .findAll('button')
      .find((b) => b.text() === 'Reset to Defaults')!
    await resetButton.trigger('click')
    await nextTick()

    expect(store.rowsPerPage).toBe(20)
  })
})
