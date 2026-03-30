import { ref, onMounted, onUnmounted } from 'vue'
import { Capacitor } from '@capacitor/core'

const MOBILE_BREAKPOINT = 768

const isMobile = ref(false)
const isNative = ref(Capacitor.isNativePlatform())

function update() {
  isMobile.value = isNative.value || window.innerWidth < MOBILE_BREAKPOINT
}

let listeners = 0

export function useShell() {
  onMounted(() => {
    if (listeners === 0) {
      update()
      window.addEventListener('resize', update)
    }
    listeners++
  })

  onUnmounted(() => {
    listeners--
    if (listeners === 0) {
      window.removeEventListener('resize', update)
    }
  })

  return { isMobile, isNative }
}
