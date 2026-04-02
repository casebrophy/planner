import { defineStore } from 'pinia'
import { createCRUDStore } from './createCRUDStore'
import { timeBlockService } from '@/services/timeBlockService'
import type { TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter } from '@/types/timeBlock'

export const useTimeBlockStore = defineStore('timeBlock', () => {
  const crud = createCRUDStore<TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter>({
    name: 'timeBlock',
    service: timeBlockService,
    defaultOrderBy: 'starts_at',
    defaultRowsPerPage: 50,
  })

  return { ...crud }
})
