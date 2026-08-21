import { defineStore } from 'pinia'

export type Annotation = { id: string; body: string; state: string; revision: number; node_key: string }
export const useReviewStore = defineStore('review', {
  state: () => ({ annotations: [] as Annotation[], loading: false, error: '' }),
  getters: { openCount: (state) => state.annotations.filter((item) => item.state !== 'resolved').length },
  actions: {
    async load(versionId: string) {
      this.loading = true; this.error = ''
      try { const response = await fetch(`/api/v1/annotations?version_id=${encodeURIComponent(versionId)}`); const payload = await response.json(); if (!response.ok) throw new Error(payload.message); this.annotations = payload.items }
      catch (error) { this.error = error instanceof Error ? error.message : 'Unable to load annotations' }
      finally { this.loading = false }
    },
    replace(updated: Annotation) { const index = this.annotations.findIndex((item) => item.id === updated.id); if (index >= 0) this.annotations.splice(index, 1, updated) }
  }
})
