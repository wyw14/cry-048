<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useNotificationStore } from '@/stores/notification'

const notif = useNotificationStore()
onMounted(() => notif.refreshUnread())
const unread = computed(() => notif.unread)
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">设计稿评审</div>
      <nav class="nav">
        <RouterLink to="/projects">项目画廊</RouterLink>
        <RouterLink to="/todos">个人待办</RouterLink>
        <RouterLink to="/settings">设置</RouterLink>
      </nav>
      <div class="user">
        <RouterLink to="/todos" class="bell">
          通知 ({{ unread }})
        </RouterLink>
      </div>
    </header>
    <main class="main">
      <RouterView />
    </main>
  </div>
</template>
