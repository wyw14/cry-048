import { defineStore } from 'pinia'
import { listNotifications, markNotificationRead, unreadCount } from '@/api/services'
import type { Notification } from '@/types/models'

interface State {
  items: Notification[]
  unread: number
  loading: boolean
}

export const useNotificationStore = defineStore('notification', {
  state: (): State => ({
    items: [],
    unread: 0,
    loading: false,
  }),
  actions: {
    async fetch(unreadOnly = false) {
      this.loading = true
      try {
        this.items = await listNotifications(unreadOnly)
      } finally {
        this.loading = false
      }
    },
    async refreshUnread() {
      this.unread = await unreadCount()
    },
    async markRead(id: string) {
      await markNotificationRead(id)
      const idx = this.items.findIndex((n) => n.id === id)
      if (idx >= 0) this.items[idx].read = true
      this.unread = Math.max(0, this.unread - 1)
    },
  },
})
