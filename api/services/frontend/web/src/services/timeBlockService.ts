import { createCRUDService } from './createCRUDService'
import type { TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter } from '@/types/timeBlock'

export const timeBlockService = createCRUDService<TimeBlock, NewTimeBlock, UpdateTimeBlock, TimeBlockFilter>({
  basePath: '/api/v1/time-blocks',
  mapFilter: (f) => ({
    task_id: f.taskId,
    date_from: f.dateFrom,
    date_to: f.dateTo,
    confirmed: f.confirmed,
  }),
})
