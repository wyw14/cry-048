<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listAudit } from '@/api/services'
import type { AuditEntry } from '@/types/models'

const userID = ref(localStorage.getItem('user_id') || 'user-local')
const entries = ref<AuditEntry[]>([])

async function saveUser() {
  localStorage.setItem('user_id', userID.value)
}

async function loadAudit() {
  entries.value = await listAudit({ actor_id: userID.value })
}

onMounted(loadAudit)
</script>

<template>
  <div>
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">设置</h2>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">当前用户</h3>
      <p class="muted">将作为 HTTP X-User-ID 头部发送，模拟离线用户身份。</p>
      <div class="row">
        <input class="input" v-model="userID" style="max-width: 240px" />
        <button class="btn btn-primary" @click="saveUser">保存</button>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">最近审计 (本人)</h3>
      <table class="data">
        <thead>
          <tr><th>操作</th><th>实体</th><th>实体 ID</th><th>时间</th></tr>
        </thead>
        <tbody>
          <tr v-for="(e, i) in entries.slice(0, 30)" :key="i">
            <td>{{ e.action }}</td>
            <td>{{ e.entity_type }}</td>
            <td>{{ e.entity_id }}</td>
            <td>{{ new Date(e.at).toLocaleString('zh-CN') }}</td>
          </tr>
          <tr v-if="entries.length === 0"><td colspan="4" class="muted" style="text-align: center">暂无审计记录</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
