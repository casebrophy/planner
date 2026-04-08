import { createCRUDService } from './createCRUDService'
import type {
  Context,
  NewContext,
  UpdateContext,
  ContextFilter,
} from '@/types'

const crud = createCRUDService<Context, NewContext, UpdateContext, ContextFilter>({
  basePath: '/api/v1/contexts',
  mapFilter: (f) => ({
    status: f.status,
    kind: f.kind,
    title: f.title,
    parent_context_id: f.parentContextId,
  }),
})

export const contextService = {
  ...crud,
}
