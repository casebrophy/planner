import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { withSetup } from '../helpers/mountHelper'
import { useShell } from '../../composables/useShell'

function setInnerWidth(width: number) {
  Object.defineProperty(window, 'innerWidth', {
    writable: true,
    configurable: true,
    value: width,
  })
}

describe('useShell', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('isMobile is true when window width < 768', () => {
    setInnerWidth(375)
    const { result, wrapper } = withSetup(() => useShell())
    expect(result.isMobile.value).toBe(true)
    wrapper.unmount()
  })

  it('isMobile is false when window width >= 768', () => {
    setInnerWidth(1024)
    const { result, wrapper } = withSetup(() => useShell())
    expect(result.isMobile.value).toBe(false)
    wrapper.unmount()
  })

  it('responds to resize events', async () => {
    setInnerWidth(1024)
    const { result, wrapper } = withSetup(() => useShell())
    expect(result.isMobile.value).toBe(false)

    setInnerWidth(375)
    window.dispatchEvent(new Event('resize'))

    expect(result.isMobile.value).toBe(true)
    wrapper.unmount()
  })

  it('cleans up resize listener on unmount', () => {
    setInnerWidth(1024)
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const { wrapper } = withSetup(() => useShell())
    wrapper.unmount()
    expect(removeSpy).toHaveBeenCalledWith('resize', expect.any(Function))
  })
})
