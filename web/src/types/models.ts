export interface Project {
  id: string
  name: string
  description: string
  status: 'active' | 'archived'
  created_at: string
  updated_at: string
  version: number
}

export interface Board {
  id: string
  project_id: string
  name: string
  Size: { Width: number; Height: number }
  size?: { width: number; height: number }
  width?: number
  height?: number
  status: 'open' | 'closed'
  created_at: string
  updated_at: string
  version: number
}

export interface Version {
  id: string
  board_id: string
  number: number
  label: string
  preview_key: string
  notes: string
  status: 'draft' | 'published' | 'superseded'
  created_at: string
  created_by: string
  version: number
}

export type Priority = 'low' | 'normal' | 'high' | 'critical'
export type AnnotationStatus = 'open' | 'replied' | 'review' | 'resolved' | 'closed'

export interface Reply {
  id: string
  annotation_id: string
  author_id: string
  body: string
  created_at: string
  edited_at?: string
}

export interface Attachment {
  id: string
  annotation_id: string
  filename: string
  media_type: string
  size: number
  storage_key: string
  uploaded_by: string
  uploaded_at: string
}

export interface Annotation {
  id: string
  project_id: string
  board_id: string
  version_id: string
  title: string
  body: string
  Anchor: {
    VersionID: string
    Region?: { X: number; Y: number; Width: number; Height: number } | null
    Point?: { X: number; Y: number } | null
  }
  anchor?: {
    version_id: string
    region?: { x: number; y: number; width: number; height: number } | null
    point?: { x: number; y: number } | null
  }
  priority: Priority
  status: AnnotationStatus
  assignee_id: string
  reporter_id: string
  due_at?: string | null
  replies: Reply[]
  attachments: Attachment[]
  created_at: string
  updated_at: string
  version: number
  resolved_at?: string | null
  resolved_by: string
}

export interface Membership {
  id: string
  project_id: string
  user_id: string
  role: 'owner' | 'editor' | 'viewer'
  joined_at: string
}

export type Recommendation =
  | 'approve'
  | 'approve_with_conditions'
  | 'reject'
  | 'defer'

export interface ReviewRound {
  id: string
  project_id: string
  board_id: string
  title: string
  status: 'open' | 'closed'
  snapshots: Snapshot[]
  conclusion: string
  recommendation: Recommendation | ''
  decided_by: string
  decided_at?: string | null
  created_at: string
  updated_at: string
  version: number
  closed_at?: string | null
}

export interface Snapshot {
  id: string
  round_id: string
  project_id: string
  board_id: string
  version_id: string
  created_at: string
  created_by: string
  counts: Record<string, number>
}

export interface Notification {
  id: string
  recipient: string
  subject: string
  body: string
  entity_type: string
  entity_id: string
  created_at: string
  read: boolean
}

export interface AuditEntry {
  id?: number
  actor_id: string
  action: string
  entity_type: string
  entity_id: string
  before: string
  after: string
  at: string
  request_id: string
}

export interface Paginated<T> {
  data: T[]
  total: number
  page: number
  page_size: number
  request_id?: string
}

export interface APIError {
  code: string
  message: string
  request_id: string
  field_errors?: { field: string; tag: string; value?: string; reason: string }[]
}

export interface APIErrorEnvelope {
  error: APIError
}
