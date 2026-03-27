import { createCRUDService } from './createCRUDService'

interface RawInput {
  id: string
  sourceType: string
  status: string
  rawContent: string
  processedAt?: string
  error?: string
  createdAt: string
}

interface NewRawInput {
  sourceType: string
  rawContent: string
}

type UpdateRawInput = Record<string, never>
type RawInputFilter = Record<string, never>

export const rawinputService = createCRUDService<
  RawInput,
  NewRawInput,
  UpdateRawInput,
  RawInputFilter
>({
  basePath: '/api/v1/raw-inputs',
})
