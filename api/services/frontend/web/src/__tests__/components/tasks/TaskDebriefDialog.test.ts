import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TaskDebriefDialog from '@/components/tasks/TaskDebriefDialog.vue'
import { makeTask } from '../../helpers/testFactories'
import { observationService } from '@/services/observationService'

vi.mock('@/services/observationService', () => ({
  observationService: {
    record: vi.fn().mockResolvedValue(undefined),
    queryByKind: vi.fn(),
  },
}))

function mountDialog(props: NonNullable<Parameters<typeof mount>[1]>['props']) {
  return mount(TaskDebriefDialog, {
    props,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } as any)
}

describe('TaskDebriefDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders dialog when open is true', () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })
    expect(wrapper.find('.fixed').exists()).toBe(true)
  })

  it('does not render dialog when open is false', () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: false,
      taskId: task.id,
      task,
    })
    expect(wrapper.find('.fixed').exists()).toBe(false)
  })

  it('handleSkip records a skip observation with task metadata', async () => {
    const task = makeTask({
      id: 'task-1',
      contextId: 'ctx-123',
      priority: 'high',
      energy: 'medium',
    })
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const skipButton = wrapper.findAll('button').find(b => b.text() === 'Skip')
    expect(skipButton).toBeTruthy()
    await skipButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith({
      subjectType: 'task',
      subjectId: 'task-1',
      kind: 'debrief',
      data: {
        outcome: 'skipped',
        contextId: 'ctx-123',
        priority: 'high',
        energy: 'medium',
      },
    })
  })

  it('handleSkip emits close event after recording', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const skipButton = wrapper.findAll('button').find(b => b.text() === 'Skip')
    await skipButton!.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('handleSkip includes null contextId when task has none', async () => {
    const task = makeTask({
      id: 'task-2',
      contextId: undefined,
      priority: 'low',
      energy: 'high',
    })
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const skipButton = wrapper.findAll('button').find(b => b.text() === 'Skip')
    await skipButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          outcome: 'skipped',
          contextId: null,
          priority: 'low',
          energy: 'high',
        }),
      })
    )
  })

  it('handleSubmit records observation with selected outcome and task metadata', async () => {
    const task = makeTask({
      id: 'task-3',
      contextId: 'ctx-456',
      priority: 'medium',
      energy: 'low',
    })
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const buttons = wrapper.findAll('button')
    const quickButton = buttons.find(b => b.text().includes('⚡ Quick'))
    await quickButton!.trigger('click')
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    await submitButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith({
      subjectType: 'task',
      subjectId: 'task-3',
      kind: 'debrief',
      data: {
        outcome: 'quick',
        note: undefined,
        contextId: 'ctx-456',
        priority: 'medium',
        energy: 'low',
      },
    })
  })

  it('handleSubmit includes note in observation data', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    // Select an outcome
    const buttons = wrapper.findAll('button')
    const normalButton = buttons.find(b => b.text().includes('✓ Normal'))
    await normalButton!.trigger('click')
    await flushPromises()

    // Enter note
    const textarea = wrapper.find('textarea')
    await textarea.setValue('Great progress!')
    await flushPromises()

    // Submit
    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    await submitButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          outcome: 'normal',
          note: 'Great progress!',
        }),
      })
    )
  })

  it('handleSubmit omits note when textarea is empty', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const buttons = wrapper.findAll('button')
    const harderButton = buttons.find(b => b.text().includes('😓'))
    await harderButton!.trigger('click')
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    await submitButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          outcome: 'harder_than_expected',
          note: undefined,
        }),
      })
    )
  })

  it('handleSubmit trims whitespace from note', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const buttons = wrapper.findAll('button')
    const blockedButton = buttons.find(b => b.text().includes('🚫'))
    await blockedButton!.trigger('click')
    await flushPromises()

    const textarea = wrapper.find('textarea')
    await textarea.setValue('  Blocked by dependency  ')
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    await submitButton!.trigger('click')
    await flushPromises()

    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          note: 'Blocked by dependency',
        }),
      })
    )
  })

  it('handleSubmit emits close event after recording', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const buttons = wrapper.findAll('button')
    const quickButton = buttons.find(b => b.text().includes('⚡'))
    await quickButton!.trigger('click')
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    await submitButton!.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('submit button is disabled when no outcome selected', () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    expect(submitButton?.attributes('disabled')).toBeDefined()
  })

  it('submit button is enabled when outcome is selected', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const buttons = wrapper.findAll('button')
    const quickButton = buttons.find(b => b.text().includes('⚡'))
    await quickButton!.trigger('click')
    await flushPromises()

    const submitButton = wrapper.findAll('button').find(b => b.text() === 'Submit')
    expect(submitButton?.attributes('disabled')).toBeUndefined()
  })

  it('clicking dialog background (self) calls handleSkip', async () => {
    const task = makeTask()
    const wrapper = mountDialog({
      open: true,
      taskId: task.id,
      task,
    })

    const backdrop = wrapper.find('.fixed')
    await backdrop.trigger('click.self')
    await flushPromises()

    // handleSkip should have been called, recording a skip observation
    expect(vi.mocked(observationService.record)).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          outcome: 'skipped',
        }),
      })
    )
  })
})
