export interface TimeBlock {
  id: string
  taskId: string
  startsAt: string
  endsAt: string
  confirmed: boolean
  createdAt: string
  updatedAt: string
}

export interface NewTimeBlock {
  taskId: string
  startsAt: string
  endsAt: string
}

export interface UpdateTimeBlock {
  startsAt?: string
  endsAt?: string
  confirmed?: boolean
}

export interface TimeBlockFilter {
  taskId?: string
  dateFrom?: string
  dateTo?: string
  confirmed?: string
}
