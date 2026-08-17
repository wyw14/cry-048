import { defineStore } from 'pinia'
import {
  listProjects,
  getProject,
  createProject,
  archiveProject,
  type ListParams,
} from '@/api/services'
import type { Project } from '@/types/models'

interface State {
  items: Project[]
  total: number
  current?: Project
  loading: boolean
  error?: string
}

export const useProjectStore = defineStore('project', {
  state: (): State => ({
    items: [],
    total: 0,
    current: undefined,
    loading: false,
    error: undefined,
  }),
  actions: {
    async fetch(params: ListParams = {}) {
      this.loading = true
      this.error = undefined
      try {
        const r = await listProjects(params)
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
        this.current = await getProject(id)
      } catch (e: unknown) {
        this.error = (e as Error).message
      } finally {
        this.loading = false
      }
    },
    async create(input: { name: string; description?: string }) {
      const p = await createProject(input)
      this.items.push(p)
      return p
    },
    async archive(id: string) {
      await archiveProject(id)
      const idx = this.items.findIndex((x) => x.id === id)
      if (idx >= 0) this.items[idx].status = 'archived'
    },
  },
})
