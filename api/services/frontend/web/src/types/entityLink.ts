export type EntityKind = 'task' | 'note' | 'event'
export type LinkKind = 'manual' | 'ai_suggested'

export interface EntityLink {
  id: string
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
  confidence: number
  kind: LinkKind
  createdAt: string
}

export interface NewEntityLink {
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
}

export interface EntityLinkFilter {
  entityType?: EntityKind
  entityId?: string
}
