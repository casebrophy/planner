import { useToastStore } from '@/stores/toastStore'
import { storeToRefs } from 'pinia'

export function useToast() {
  const store = useToastStore()
  const { toasts } = storeToRefs(store)

  return {
    toasts,
    success: store.success,
    error: store.error,
    info: store.info,
    dismiss: store.dismiss,
  }
}
