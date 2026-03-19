import { onMounted, onUnmounted, ref } from 'vue'

export function usePolling(fn: () => Promise<void> | void, intervalMs = 60000) {
  const active = ref(true)
  let timer: ReturnType<typeof setInterval> | null = null

  function start() {
    stop()
    timer = setInterval(async () => {
      if (active.value && !document.hidden) {
        await fn()
      }
    }, intervalMs)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  function onVisibilityChange() {
    if (document.hidden) {
      active.value = false
    } else {
      active.value = true
      // Immediate fetch on tab re-focus
      fn()
    }
  }

  onMounted(() => {
    document.addEventListener('visibilitychange', onVisibilityChange)
    start()
  })

  onUnmounted(() => {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    stop()
  })

  return { active, start, stop }
}
