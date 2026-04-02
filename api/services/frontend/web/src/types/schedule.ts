export interface ScheduleItem {
  type: 'event' | 'time_block'
  id: string
  title: string
  startsAt: string
  endsAt: string
  allDay?: boolean
  location?: string
  taskId?: string
  confirmed?: boolean
}

export interface ScheduleResponse {
  items: ScheduleItem[]
}
