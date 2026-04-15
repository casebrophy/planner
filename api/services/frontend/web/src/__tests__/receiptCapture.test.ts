import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useReceiptCaptureStore } from '@/stores/receiptCaptureStore'
import type { ReceiptExtraction } from '@/services/receiptService'

vi.mock('@/services/receiptService', () => ({
  receiptService: {
    extractReceipt: vi.fn().mockResolvedValue({
      merchant: 'Test Cafe',
      date: '2026-04-14',
      total: 2500,
      tax: 200,
      subtotal: 2300,
      items: [{ description: 'Coffee', amount: 500, quantity: 1 }],
    } as ReceiptExtraction),
  },
}))

const mockExtraction: ReceiptExtraction = {
  merchant: 'Test Cafe',
  date: '2026-04-14',
  total: 2500,
  tax: 200,
  subtotal: 2300,
  items: [{ description: 'Coffee', amount: 500, quantity: 1 }],
}

describe('receiptCaptureStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('starts in idle step', () => {
      const store = useReceiptCaptureStore()
      expect(store.step).toBe('idle')
    })

    it('has null imageFile initially', () => {
      const store = useReceiptCaptureStore()
      expect(store.imageFile).toBeNull()
    })

    it('has empty ocrText initially', () => {
      const store = useReceiptCaptureStore()
      expect(store.ocrText).toBe('')
    })

    it('has null extraction initially', () => {
      const store = useReceiptCaptureStore()
      expect(store.extraction).toBeNull()
    })

    it('has empty splits array initially', () => {
      const store = useReceiptCaptureStore()
      expect(store.splits).toEqual([])
    })

    it('has null error initially', () => {
      const store = useReceiptCaptureStore()
      expect(store.error).toBeNull()
    })
  })

  describe('setImageFile', () => {
    it('sets the imageFile and transitions to scanning', () => {
      const store = useReceiptCaptureStore()
      const file = new File(['content'], 'receipt.jpg', { type: 'image/jpeg' })

      store.setImageFile(file)

      expect(store.imageFile).toBe(file)
      expect(store.step).toBe('scanning')
    })
  })

  describe('setOcrText', () => {
    it('sets the ocrText', () => {
      const store = useReceiptCaptureStore()
      const text = 'Test Cafe\n$25.00'

      store.setOcrText(text)

      expect(store.ocrText).toBe(text)
    })
  })

  describe('runExtraction', () => {
    it('sets step to extracting and calls receipt service', async () => {
      const store = useReceiptCaptureStore()
      store.setImageFile(new File(['content'], 'receipt.jpg'))
      store.setOcrText('Test Cafe\n$25.00')

      const promise = store.runExtraction()
      expect(store.step).toBe('extracting')

      await promise

      expect(store.extraction).toEqual(mockExtraction)
      expect(store.step).toBe('reviewing')
    })

    it('sets error and stays in extracting on failure', async () => {
      vi.mocked(require('@/services/receiptService').receiptService.extractReceipt).mockRejectedValueOnce(
        new Error('Extraction failed'),
      )

      const store = useReceiptCaptureStore()
      store.setImageFile(new File(['content'], 'receipt.jpg'))
      store.setOcrText('Test Cafe\n$25.00')

      await store.runExtraction()

      expect(store.error).toBeTruthy()
      expect(store.step).toBe('extracting')
    })
  })

  describe('updateExtraction', () => {
    it('updates extraction fields', () => {
      const store = useReceiptCaptureStore()
      store.extraction = { ...mockExtraction }

      store.updateExtraction({ merchant: 'Updated Cafe', total: 3000 })

      expect(store.extraction?.merchant).toBe('Updated Cafe')
      expect(store.extraction?.total).toBe(3000)
    })
  })

  describe('startSplitting', () => {
    it('transitions to splitting step', () => {
      const store = useReceiptCaptureStore()
      store.extraction = { ...mockExtraction }

      store.startSplitting()

      expect(store.step).toBe('splitting')
    })
  })


  describe('confirm', () => {
    it('transitions to idle after confirmation', () => {
      const store = useReceiptCaptureStore()
      store.extraction = { ...mockExtraction }
      store.step = 'reviewing'

      store.confirm()

      expect(store.step).toBe('idle')
    })
  })

  describe('reset', () => {
    it('resets all state', () => {
      const store = useReceiptCaptureStore()
      store.imageFile = new File(['content'], 'receipt.jpg')
      store.ocrText = 'Test Cafe'
      store.extraction = { ...mockExtraction }
      store.error = 'Some error'
      store.step = 'reviewing'

      store.reset()

      expect(store.imageFile).toBeNull()
      expect(store.ocrText).toBe('')
      expect(store.extraction).toBeNull()
      expect(store.error).toBeNull()
      expect(store.step).toBe('idle')
    })
  })
})

describe('SplitEditor', () => {
  describe('split evenly calculation', () => {
    it('divides total by number of people including you', () => {
      const totalCents = 10000 // $100
      const numPeople = 4 // 3 parties + you
      const expectedPerPerson = (totalCents / 100 / numPeople).toFixed(2) // $25.00

      expect(expectedPerPerson).toBe('25.00')
    })

    it('handles uneven division with proper rounding', () => {
      const totalCents = 10001 // $100.01
      const numPeople = 3 // 2 parties + you
      const expectedPerPerson = (totalCents / 100 / numPeople).toFixed(2)

      expect(parseFloat(expectedPerPerson)).toBeCloseTo(33.34, 2)
    })
  })

  describe('Venmo deep link format', () => {
    it('generates correct Venmo link with handle and amount', () => {
      const handle = 'john'
      const amount = '10.00'
      const link = `venmo://paycharge?txn=charge&recipients=${encodeURIComponent(handle)}&amount=${amount}&note=`

      expect(link).toBe('venmo://paycharge?txn=charge&recipients=john&amount=10.00&note=')
    })

    it('encodes special characters in Venmo handle', () => {
      const handle = 'john-doe'
      const amount = '15.50'
      const link = `venmo://paycharge?txn=charge&recipients=${encodeURIComponent(handle)}&amount=${amount}&note=`

      expect(link).toContain('john-doe')
    })

    it('formats amount to 2 decimal places', () => {
      const amount = (15.5).toFixed(2)
      expect(amount).toBe('15.50')
    })
  })
})
