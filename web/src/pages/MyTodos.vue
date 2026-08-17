<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAnnotationStore } from '@/stores/annotation'
import { useNotificationStore } from '@/stores/notification'

const annoStore = useAnnotationStore()
const notif = useNotificationStore()
const router = useRouter()
const query = ref('')

onMounted(async () => {
  await annoStore.fetchTodos()
  await notif.fetch(false)
})

async function search() {
  if (query.value.trim()) {
    await annoStore.search(query.value)
  } else {
    await annoStore.fetchTodos()
  }
}

function open(id: string) {
  router.push(`/annotations/${id}`)
}

async function markRead(id: string) {
  await notif.markRead(id)
}
</script>

<template>
  <div>
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">个人待办</h2>
    </div>

    <div class="card">
      <div class="row">
        <input class="input" v-model="query" placeholder="全文搜索…" @keyup.enter="search" />
        <button class="btn btn-primary" @click="search">搜索</button>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">未读通知 ({{ notif.unread }})</h3>
      <table class="data">
        <thead>
          <tr><th>主题</th><th>正文</th><th>时间</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="n in notif.items" :key="n.id">
            <td>{{ n.subject }}</td>
            <td>{{ n.body }}</td>
            <td>{{ new Date(n.created_at).toLocaleString('zh-CN') }}</td>
            <td>
              <button v-if="!n.read" class="btn btn-sm" @click="markRead(n.id)">标为已读</button>
              <span v-else class="muted">已读</span>
            </td>
          </tr>
          <tr v-if="notif.items.length === 0"><td colspan="4" class="muted" style="text-align: center">暂无通知</td></tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">待处理批注 ({{ annoStore.todos.length }})</h3>
      <table class="data">
        <thead>
          <tr><th>标题</th><th>状态</th><th>优先级</th><th>截止</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in annoStore.todos" :key="a.id" style="cursor: pointer" @click="open(a.id)">
            <td>{{ a.title }}</td>
            <td><span class="tag" :class="a.status">{{ a.status }}</span></td>
            <td><span class="tag" :class="a.priority">{{ a.priority }}</span></td>
            <td>{{ a.due_at ? new Date(a.due_at).toLocaleDateString('zh-CN') : '-' }}</td>
          </tr>
          <tr v-if="annoStore.todos.length === 0"><td colspan="4" class="muted" style="text-align: center">暂无待办</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
