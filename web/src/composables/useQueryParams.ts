import { watch, type Ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'

export function useQueryParams(
  params: Record<string, Ref<string | number | undefined>>,
  onUpdate?: () => void,
) {
  const router = useRouter()
  const route = useRoute()

  // Initialize refs from URL query params
  for (const [key, ref] of Object.entries(params)) {
    const val = route.query[key]
    if (val && typeof val === 'string') {
      if (typeof ref.value === 'number') {
        ref.value = parseInt(val, 10) || ref.value
      } else {
        ref.value = val
      }
    }
  }

  // Watch refs and sync to URL
  const refs = Object.values(params)
  watch(
    refs,
    () => {
      const query: Record<string, string> = {}
      for (const [key, ref] of Object.entries(params)) {
        if (ref.value !== undefined && ref.value !== '') {
          query[key] = String(ref.value)
        }
      }
      router.replace({ query }).catch(() => {
        // ignore navigation duplicated
      })
      onUpdate?.()
    },
    { deep: true },
  )
}
