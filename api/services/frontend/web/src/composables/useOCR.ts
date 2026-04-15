import { ref } from 'vue'
import { createWorker } from 'tesseract.js'
import type Tesseract from 'tesseract.js'

export function useOCR() {
  const isProcessing = ref(false)
  const progress = ref(0)
  const error = ref<string | null>(null)

  async function recognize(imageFile: File): Promise<string> {
    isProcessing.value = true
    progress.value = 0
    error.value = null
    try {
      const worker = await createWorker('eng', 1, {
        logger: (m: Tesseract.LoggerMessage) => {
          if (m.status === 'recognizing text') progress.value = m.progress
        }
      })
      const { data: { text } } = await worker.recognize(imageFile)
      await worker.terminate()
      return text
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'OCR failed'
      throw e
    } finally {
      isProcessing.value = false
    }
  }

  return { recognize, isProcessing, progress, error }
}
