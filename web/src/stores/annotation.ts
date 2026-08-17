import { defineStore } from 'pinia'
import {
  listAnnotations,
  getAnnotation,
  createAnnotation,
  addReply,
  setAssignee,
  setPriority,
  setDue,
  requestReview,
  resolveAnnotation,
  reopenAnnotation,
  closeAnnotation,
  migrateAnchors,
  searchAnnotations,
  listMyTodos,
  type AnnotationListFilter,
} from '@/api/services'
import type { Annotation, Priority } from '@/types/models'

interface State {
  items: Annotation[]
  total: number
  current?: Annotation
  todos: Annotation[]
  searchResults: Annotation[]
  loading: boolean
  error?: string
}

export const useAnnotationStore = defineStore('annotation', {
  state: (): State => ({
    items: [],
    total: 0,
    current: undefined,
    todos: [],
    searchResults: [],
    loading: false,
    error: undefined,
  }),
  actions: {
    async fetch(filter: AnnotationListFilter = {}) {
      this.loading = true
      try {
        const r = await listAnnotations(filter)
        this.items = r.items
        this.total = r.total
      } catch (e: unknown) {
        this.error = (e as Error).message
      } finally {
        this.loading = false
      }
    },
    async fetchOne(id: string) {
      this.loading = true
      try {
        this.current = await getAnnotation(id)
      } catch (e: unknown) {
        this.error = (e as Error).message
      } finally {
        this.loading = false
      }
    },
    async fetchTodos() {
      try {
        const r = await listMyTodos()
        this.todos = r.items
      } catch (e: unknown) {
        this.error = (e as Error).message
      }
    },
    async search(q: string) {
      try {
        const r = await searchAnnotations(q)
        this.searchResults = r.items
      } catch (e: unknown) {
        this.error = (e as Error).message
      }
    },
    async create(input: Parameters<typeof createAnnotation>[0]) {
      const a = await createAnnotation(input)
      this.items.push(a)
      return a
    },
    async reply(id: string, body: string) {
      const a = await addReply(id, body)
      this.current = a
      return a
    },
    async assign(id: string, assigneeID: string) {
      const a = await setAssignee(id, assigneeID)
      this.current = a
      return a
    },
    async changePriority(id: string, priority: Priority) {
      const a = await setPriority(id, priority)
      this.current = a
      return a
    },
    async changeDue(id: string, due: string | null) {
      const a = await setDue(id, due)
      this.current = a
      return a
    },
    async review(id: string) {
      const a = await requestReview(id)
      this.current = a
      return a
    },
    async resolve(id: string) {
      const a = await resolveAnnotation(id)
      this.current = a
      return a
    },
    async reopen(id: string, reason?: string) {
      const a = await reopenAnnotation(id, reason)
      this.current = a
      return a
    },
    async close(id: string) {
      const a = await closeAnnotation(id)
      this.current = a
      return a
    },
    async migrate(input: Parameters<typeof migrateAnchors>[0]) {
      return migrateAnchors(input)
    },
  },
})
