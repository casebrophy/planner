import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ProcessingStatus from '@/components/shared/ProcessingStatus.vue'

vi.mock('@/services/client', () => ({
  request: vi.fn(),
}))

import { request } from '@/services/client'

const mockRawInput = (status: string) => ({
  id: 'ri-001',
  sourceType: 'email',
  status,
  rawContent: 'test content',
  createdAt: '2026-01-01T00:00:00Z',
})

describe('ProcessingStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading state initially', async () => {
    vi.mocked(request).mockReturnValue(new Promise(() => {})) // never resolves
    const wrapper = mount(ProcessingStatus, {
      props: { rawInputId: 'ri-001' },
    })
    await nextTick()

    expect(wrapper.find('.animate-spin').exists()).toBe(true)
  })

  it('renders 3-step indicator on success', async () => {
    vi.mocked(request).mockResolvedValue(mockRawInput('processed'))

    const wrapper = mount(ProcessingStatus, {
      props: { rawInputId: 'ri-001' },
    })
    await nextTick()
    await nextTick()

    const labels = wrapper.findAll('span.capitalize')
    expect(labels).toHaveLength(3)
    expect(labels.map((l) => l.text())).toEqual(['pending', 'processing', 'processed'])
  })

  it('highlights current step with blue', async () => {
    vi.mocked(request).mockResolvedValue(mockRawInput('processing'))

    const wrapper = mount(ProcessingStatus, {
      props: { rawInputId: 'ri-001' },
    })
    await nextTick()
    await nextTick()

    const labels = wrapper.findAll('span.capitalize')
    // "processing" is the current step (index 1) -- should be blue
    expect(labels[1]!.classes()).toContain('text-blue-400')
    // "pending" is completed (index 0) -- should be green
    expect(labels[0]!.classes()).toContain('text-green-400')
    // "processed" is future (index 2) -- should be gray
    expect(labels[2]!.classes()).toContain('text-gray-600')
  })

  it('shows error state for status=error', async () => {
    vi.mocked(request).mockResolvedValue(mockRawInput('error'))

    const wrapper = mount(ProcessingStatus, {
      props: { rawInputId: 'ri-001' },
    })
    await nextTick()
    await nextTick()

    const labels = wrapper.findAll('span.capitalize')
    expect(labels[0]!.text()).toBe('error')
    expect(labels[0]!.classes()).toContain('text-red-400')
  })

  it('shows fetch error message', async () => {
    vi.mocked(request).mockRejectedValue(new Error('Network failed'))

    const wrapper = mount(ProcessingStatus, {
      props: { rawInputId: 'ri-001' },
    })
    await nextTick()
    await nextTick()

    expect(wrapper.text()).toContain('Network failed')
  })
})
