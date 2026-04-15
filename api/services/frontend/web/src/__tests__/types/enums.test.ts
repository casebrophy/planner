import { describe, it, expect } from 'vitest'
import {
  TaskStatus, TaskPriority, TaskEnergy, ContextStatus, ContextKind,
  ClarificationKind, ClarificationStatus,
  TaskStatusLabels, TaskPriorityLabels, TaskEnergyLabels,
  ContextStatusLabels, ContextKindLabels,
  StatusColors, PriorityColors,
  ClarificationKindLabels, ClarificationKindColors,
} from '@/types/enums'

describe('TaskStatus', () => {
  it('has correct string values', () => {
    expect(TaskStatus.Open).toBe('open')
    expect(TaskStatus.Blocked).toBe('blocked')
    expect(TaskStatus.Done).toBe('done')
    expect(TaskStatus.Dismissed).toBe('dismissed')
  })

  it('has exactly 4 values', () => {
    expect(Object.keys(TaskStatus)).toHaveLength(4)
  })
})

describe('TaskPriority', () => {
  it('has correct string values', () => {
    expect(TaskPriority.Low).toBe('low')
    expect(TaskPriority.Medium).toBe('medium')
    expect(TaskPriority.High).toBe('high')
    expect(TaskPriority.Urgent).toBe('urgent')
  })

  it('has exactly 4 values', () => {
    expect(Object.keys(TaskPriority)).toHaveLength(4)
  })
})

describe('TaskEnergy', () => {
  it('has correct string values', () => {
    expect(TaskEnergy.Low).toBe('low')
    expect(TaskEnergy.Medium).toBe('medium')
    expect(TaskEnergy.High).toBe('high')
  })

  it('has exactly 3 values', () => {
    expect(Object.keys(TaskEnergy)).toHaveLength(3)
  })
})

describe('ContextStatus', () => {
  it('has correct string values', () => {
    expect(ContextStatus.Active).toBe('active')
    expect(ContextStatus.Paused).toBe('paused')
    expect(ContextStatus.Closed).toBe('closed')
  })

  it('has exactly 3 values', () => {
    expect(Object.keys(ContextStatus)).toHaveLength(3)
  })
})

describe('ContextKind', () => {
  it('has correct string values', () => {
    expect(ContextKind.Project).toBe('project')
    expect(ContextKind.Area).toBe('area')
  })

  it('has exactly 3 values', () => {
    expect(Object.keys(ContextKind)).toHaveLength(3)
  })
})

describe('ClarificationKind', () => {
  it('has correct string values', () => {
    expect(ClarificationKind.ContextAssignment).toBe('context_assignment')
    expect(ClarificationKind.StaleTask).toBe('stale_task')
    expect(ClarificationKind.AmbiguousDeadline).toBe('ambiguous_deadline')
    expect(ClarificationKind.NewContext).toBe('new_context')
    expect(ClarificationKind.OverlappingContexts).toBe('overlapping_contexts')
    expect(ClarificationKind.AmbiguousAction).toBe('ambiguous_action')
    expect(ClarificationKind.VoiceReference).toBe('voice_reference')
    expect(ClarificationKind.InactivityPrompt).toBe('inactivity_prompt')
    expect(ClarificationKind.ContextDebrief).toBe('context_debrief')
    expect(ClarificationKind.TaskDebrief).toBe('task_debrief')
    expect(ClarificationKind.EntityLink).toBe('entity_link')
  })
})

describe('ClarificationStatus', () => {
  it('has correct string values', () => {
    expect(ClarificationStatus.Pending).toBe('pending')
    expect(ClarificationStatus.Snoozed).toBe('snoozed')
    expect(ClarificationStatus.Resolved).toBe('resolved')
    expect(ClarificationStatus.Dismissed).toBe('dismissed')
  })

  it('has exactly 4 values', () => {
    expect(Object.keys(ClarificationStatus)).toHaveLength(4)
  })
})

describe('TaskStatusLabels', () => {
  it('maps all TaskStatus values to non-empty labels', () => {
    for (const value of Object.values(TaskStatus)) {
      expect(TaskStatusLabels[value]).toBeTruthy()
      expect(typeof TaskStatusLabels[value]).toBe('string')
    }
  })

  it('has correct labels', () => {
    expect(TaskStatusLabels[TaskStatus.Open]).toBe('Open')
    expect(TaskStatusLabels[TaskStatus.Blocked]).toBe('Blocked')
    expect(TaskStatusLabels[TaskStatus.Done]).toBe('Done')
    expect(TaskStatusLabels[TaskStatus.Dismissed]).toBe('Dismissed')
  })
})

describe('TaskPriorityLabels', () => {
  it('maps all TaskPriority values to non-empty labels', () => {
    for (const value of Object.values(TaskPriority)) {
      expect(TaskPriorityLabels[value]).toBeTruthy()
    }
  })

  it('has correct labels', () => {
    expect(TaskPriorityLabels[TaskPriority.Low]).toBe('Low')
    expect(TaskPriorityLabels[TaskPriority.Medium]).toBe('Medium')
    expect(TaskPriorityLabels[TaskPriority.High]).toBe('High')
    expect(TaskPriorityLabels[TaskPriority.Urgent]).toBe('Urgent')
  })
})

describe('TaskEnergyLabels', () => {
  it('maps all TaskEnergy values to non-empty labels', () => {
    for (const value of Object.values(TaskEnergy)) {
      expect(TaskEnergyLabels[value]).toBeTruthy()
    }
  })

  it('has correct labels', () => {
    expect(TaskEnergyLabels[TaskEnergy.Low]).toBe('Low')
    expect(TaskEnergyLabels[TaskEnergy.Medium]).toBe('Medium')
    expect(TaskEnergyLabels[TaskEnergy.High]).toBe('High')
  })
})

describe('ContextStatusLabels', () => {
  it('maps all ContextStatus values to non-empty labels', () => {
    for (const value of Object.values(ContextStatus)) {
      expect(ContextStatusLabels[value]).toBeTruthy()
    }
  })

  it('has correct labels', () => {
    expect(ContextStatusLabels[ContextStatus.Active]).toBe('Active')
    expect(ContextStatusLabels[ContextStatus.Paused]).toBe('Paused')
    expect(ContextStatusLabels[ContextStatus.Closed]).toBe('Closed')
  })
})

describe('ContextKindLabels', () => {
  it('maps all ContextKind values to non-empty labels', () => {
    for (const value of Object.values(ContextKind)) {
      expect(ContextKindLabels[value]).toBeTruthy()
    }
  })

  it('has correct labels', () => {
    expect(ContextKindLabels[ContextKind.Project]).toBe('Project')
    expect(ContextKindLabels[ContextKind.Area]).toBe('Area')
  })
})

describe('StatusColors', () => {
  it('has entries for all TaskStatus values', () => {
    for (const value of Object.values(TaskStatus)) {
      expect(StatusColors[value]).toBeTruthy()
    }
  })

  it('has entries for all ContextStatus values', () => {
    for (const value of Object.values(ContextStatus)) {
      expect(StatusColors[value]).toBeTruthy()
    }
  })

  it('returns color strings in hex format', () => {
    for (const color of Object.values(StatusColors)) {
      expect(color).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })
})

describe('PriorityColors', () => {
  it('has entries for all TaskPriority values', () => {
    for (const value of Object.values(TaskPriority)) {
      expect(PriorityColors[value]).toBeTruthy()
    }
  })

  it('returns color strings in hex format', () => {
    for (const color of Object.values(PriorityColors)) {
      expect(color).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })

  it('urgent is red', () => {
    expect(PriorityColors[TaskPriority.Urgent]).toBe('#ef4444')
  })
})

describe('ClarificationKindLabels', () => {
  it('has entries for all ClarificationKind values', () => {
    for (const value of Object.values(ClarificationKind)) {
      expect(ClarificationKindLabels[value]).toBeTruthy()
    }
  })

  it('has correct labels for key kinds', () => {
    expect(ClarificationKindLabels[ClarificationKind.ContextAssignment]).toBe('Context Assignment')
    expect(ClarificationKindLabels[ClarificationKind.StaleTask]).toBe('Stale Task')
    expect(ClarificationKindLabels[ClarificationKind.AmbiguousDeadline]).toBe('Ambiguous Deadline')
    expect(ClarificationKindLabels[ClarificationKind.NewContext]).toBe('New Context')
    expect(ClarificationKindLabels[ClarificationKind.EntityLink]).toBe('Entity Link')
  })
})

describe('ClarificationKindColors', () => {
  it('has entries for all ClarificationKind values', () => {
    for (const value of Object.values(ClarificationKind)) {
      expect(ClarificationKindColors[value]).toBeTruthy()
    }
  })

  it('returns color strings in hex format', () => {
    for (const color of Object.values(ClarificationKindColors)) {
      expect(color).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })
})
