import { apiClient, unwrap, unwrapList } from './client'
import type {
  Annotation,
  AnnotationStatus,
  AuditEntry,
  Board,
  Membership,
  Notification,
  Paginated,
  Priority,
  Project,
  ReviewRound,
  Snapshot,
  Version,
} from '@/types/models'

export interface ListParams {
  page?: number
  page_size?: number
  sort_by?: string
  order?: 'asc' | 'desc'
  [k: string]: string | number | undefined
}

export interface AnnotationListFilter extends ListParams {
  project_id?: string
  board_id?: string
  version_id?: string
  status?: AnnotationStatus
  priority?: Priority
  assignee_id?: string
  reporter_id?: string
}

// === projects ===
export const createProject = (input: {
  id?: string
  name: string
  description?: string
}) => unwrap<Project>(apiClient.post('/projects', input))

export const listProjects = (params: ListParams = {}) =>
  unwrapList<Project>(apiClient.get('/projects', { params }))

export const getProject = (id: string) =>
  unwrap<Project>(apiClient.get(`/projects/${id}`))

export const archiveProject = (id: string) =>
  unwrap<{ id: string; status: string }>(apiClient.post(`/projects/${id}/archive`))

// === boards ===
export const createBoard = (input: {
  id?: string
  project_id: string
  name: string
  width: number
  height: number
}) => unwrap<Board>(apiClient.post('/boards', input))

export const getBoard = (id: string) => unwrap<Board>(apiClient.get(`/boards/${id}`))

export const listBoards = (projectID: string) =>
  apiClient
    .get<{ data: { items: Board[] } }>(`/projects/${projectID}/boards`)
    .then((r) => r.data.data.items)

export const closeBoard = (id: string) =>
  unwrap<{ id: string; status: string }>(apiClient.post(`/boards/${id}/close`))

// === versions ===
export const createVersion = (input: {
  id?: string
  board_id: string
  number: number
  label?: string
  preview_key: string
  notes?: string
  created_by?: string
}) => unwrap<Version>(apiClient.post('/versions', input))

export const getVersion = (id: string) =>
  unwrap<Version>(apiClient.get(`/versions/${id}`))

export const listVersions = (boardID: string) =>
  apiClient
    .get<{ data: { items: Version[] } }>(`/boards/${boardID}/versions`)
    .then((r) => r.data.data.items)

export const publishVersion = (id: string) =>
  unwrap<Version>(apiClient.post(`/versions/${id}/publish`))

// === members ===
export const addMember = (input: {
  id?: string
  project_id: string
  user_id: string
  role: 'owner' | 'editor' | 'viewer'
}) => unwrap<Membership>(apiClient.post('/members', input))

export const listMembers = (projectID: string) =>
  apiClient
    .get<{ data: { items: Membership[] } }>(`/members/${projectID}`)
    .then((r) => r.data.data.items)

export const removeMember = (id: string) =>
  unwrap<{ id: string; removed: boolean }>(apiClient.delete(`/members/${id}`))

// === annotations ===
export const createAnnotation = (input: {
  id?: string
  project_id: string
  board_id: string
  version_id: string
  title: string
  body?: string
  reporter_id?: string
  point?: { x: number; y: number }
  region?: { x: number; y: number; width: number; height: number }
  priority?: Priority
  assignee_id?: string
  due_at?: string | null
}) => unwrap<Annotation>(apiClient.post('/annotations', input))

export const getAnnotation = (id: string) =>
  unwrap<Annotation>(apiClient.get(`/annotations/${id}`))

export const listAnnotations = (filter: AnnotationListFilter = {}) =>
  unwrapList<Annotation>(apiClient.get('/annotations', { params: filter }))

export const searchAnnotations = (q: string, limit = 20) =>
  apiClient
    .get<{ data: { items: Annotation[]; count: number } }>('/annotations/search', {
      params: { q, limit },
    })
    .then((r) => r.data.data)

export const listMyTodos = () =>
  apiClient
    .get<{ data: { items: Annotation[]; count: number } }>(
      '/annotations/my-todos',
    )
    .then((r) => r.data.data)

export const addReply = (id: string, body: string) =>
  unwrap<Annotation>(apiClient.post(`/annotations/${id}/replies`, { body }))

export const setAssignee = (id: string, assignee_id: string) =>
  unwrap<Annotation>(apiClient.patch(`/annotations/${id}/assignee`, { assignee_id }))

export const setDue = (id: string, due_at: string | null) =>
  unwrap<Annotation>(apiClient.patch(`/annotations/${id}/due`, { due_at }))

export const setPriority = (id: string, priority: Priority) =>
  unwrap<Annotation>(apiClient.patch(`/annotations/${id}/priority`, { priority }))

export const requestReview = (id: string) =>
  unwrap<Annotation>(apiClient.post(`/annotations/${id}/request-review`))

export const resolveAnnotation = (id: string) =>
  unwrap<Annotation>(apiClient.post(`/annotations/${id}/resolve`))

export const reopenAnnotation = (id: string, reason?: string) =>
  unwrap<Annotation>(apiClient.post(`/annotations/${id}/reopen`, { reason }))

export const closeAnnotation = (id: string) =>
  unwrap<Annotation>(apiClient.post(`/annotations/${id}/close`))

export const addAttachment = (
  id: string,
  input: {
    filename: string
    media_type: string
    size: number
    storage_key: string
  },
) => unwrap<Annotation>(apiClient.post(`/annotations/${id}/attachments`, input))

export const migrateAnchors = (input: {
  from_version_id: string
  to_version_id: string
  reason?: string
}) =>
  unwrap<{ migrated_count: number }>(
    apiClient.post('/annotations/migrate-anchors', input),
  )

// === reviews ===
export const createRound = (input: {
  id?: string
  project_id: string
  board_id: string
  title: string
}) => unwrap<ReviewRound>(apiClient.post('/reviews/rounds', input))

export const getRound = (id: string) =>
  unwrap<ReviewRound>(apiClient.get(`/reviews/rounds/${id}`))

export const listRounds = (boardID: string) =>
  apiClient
    .get<{ data: { items: ReviewRound[] } }>(`/reviews/rounds/board/${boardID}`)
    .then((r) => r.data.data.items)

export const createSnapshot = (roundID: string, input: {
  project_id: string
  version_id: string
}) =>
  unwrap<Snapshot>(apiClient.post(`/reviews/rounds/${roundID}/snapshots`, input))

export const setConclusion = (id: string, input: {
  conclusion: string
  recommendation: 'approve' | 'approve_with_conditions' | 'reject' | 'defer'
}) => unwrap<ReviewRound>(apiClient.post(`/reviews/rounds/${id}/conclusion`, input))

export const closeRound = (id: string) =>
  unwrap<ReviewRound>(apiClient.post(`/reviews/rounds/${id}/close`))

// === notifications ===
export const listNotifications = (unreadOnly = false) =>
  apiClient
    .get<{ data: Notification[]; total: number; page: number; page_size: number }>(
      '/notifications',
      { params: { unread: unreadOnly ? 'true' : 'false' } },
    )
    .then((r) => r.data.data as Notification[])

export const markNotificationRead = (id: string) =>
  unwrap<{ id: string; read: boolean }>(apiClient.post(`/notifications/${id}/read`))

export const unreadCount = () =>
  apiClient
    .get<{ data: { unread: number } }>('/notifications/unread-count')
    .then((r) => r.data.data.unread)

// === audit ===
export interface AuditQuery {
  actor_id?: string
  entity_type?: string
  entity_id?: string
  from?: string
  to?: string
}
export const listAudit = (q: AuditQuery = {}) =>
  apiClient
    .get<{ data: AuditEntry[]; total: number }>('/audit', { params: q })
    .then((r) => r.data.data as AuditEntry[])

// === export ===
export const exportAnnotationsCsv = (filter: AnnotationListFilter = {}) =>
  apiClient.get<string>('/export/annotations.csv', {
    params: filter,
    responseType: 'text',
  }).then((r) => r.data)

export const exportReviewMinutesCsv = (roundID: string) =>
  apiClient
    .get<string>(`/export/reviews/${roundID}/minutes.csv`, {
      responseType: 'text',
    })
    .then((r) => r.data)

export type { Paginated }
