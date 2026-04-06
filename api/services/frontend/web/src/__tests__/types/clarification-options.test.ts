import { describe, it, expect } from 'vitest'
import type { ClarificationAnswerOptions } from '@/types/clarification'
import type { ContextAssignmentOptions } from '@/types/generated/clarification-options'

describe('ClarificationAnswerOptions', () => {
  it('ContextAssignmentOptions has correct field names', () => {
    const opts: ContextAssignmentOptions = {
      suggested_context: 'uuid-here',
      confidence: 0.7,
      available_contexts: [{ id: 'ctx1', title: 'Work' }],
    }
    expect(opts.suggested_context).toBe('uuid-here')
    expect(opts.available_contexts[0]!.title).toBe('Work')
  })

  it('ClarificationAnswerOptions is a union that accepts ContextAssignmentOptions', () => {
    const opts: ClarificationAnswerOptions = {
      suggested_context: 'uuid',
      confidence: 0.5,
      available_contexts: [],
    }
    expect(opts).toBeTruthy()
  })
})
