import { describe, it, expectTypeOf } from 'vitest'
import type { ClarificationAnswerOptions } from '@/types/clarification'
import type {
  ContextAssignmentOptions,
  NewContextOptions,
  AmbiguousActionOptions,
  AmbiguousDeadlineOptions,
  ContextRef,
} from '@/types/generated/clarification-options'

describe('ClarificationAnswerOptions types', () => {
  it('ContextAssignmentOptions has correct field names', () => {
    expectTypeOf<ContextAssignmentOptions>().toHaveProperty('suggested_context')
    expectTypeOf<ContextAssignmentOptions>().toHaveProperty('confidence')
    expectTypeOf<ContextAssignmentOptions>().toHaveProperty('available_contexts')
    expectTypeOf<ContextAssignmentOptions['available_contexts']>().toEqualTypeOf<ContextRef[]>()
  })

  it('ContextRef has id and title', () => {
    expectTypeOf<ContextRef>().toHaveProperty('id')
    expectTypeOf<ContextRef>().toHaveProperty('title')
  })

  it('ClarificationAnswerOptions union accepts all option types', () => {
    expectTypeOf<ContextAssignmentOptions>().toMatchTypeOf<ClarificationAnswerOptions>()
    expectTypeOf<NewContextOptions>().toMatchTypeOf<ClarificationAnswerOptions>()
    expectTypeOf<AmbiguousActionOptions>().toMatchTypeOf<ClarificationAnswerOptions>()
    expectTypeOf<AmbiguousDeadlineOptions>().toMatchTypeOf<ClarificationAnswerOptions>()
  })

  it('ClarificationAnswerOptions accepts null', () => {
    expectTypeOf<null>().toMatchTypeOf<ClarificationAnswerOptions>()
  })
})
