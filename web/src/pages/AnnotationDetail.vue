<script setup lang="ts">
import { onMounted, ref, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAnnotationStore } from '@/stores/annotation'
import type { Priority } from '@/types/models'

const route = useRoute()
const router = useRouter()
const store = useAnnotationStore()
const replyBody = ref('')
const showAssignForm = ref(false)
const assigneeID = ref('')
const newPriority = ref<Priority>('normal')

const statusOrder: string[] = ['open', 'replied', 'review', 'resolved', 'closed']

async function load() {
  await store.fetchOne(route.params.id as string)
  if (store.current) {
    newPriority.value = store.current.priority
    assigneeID.value = store.current.assignee_id
  }
}

onMounted(load)
watch(() => route.params.id, load)

const currentStatusIndex = computed(() => {
  if (!store.current) return -1
  return statusOrder.indexOf(store.current.status)
})

async function reply() {
  if (!replyBody.value.trim()) return
  await store.reply(route.params.id as string, replyBody.value)
  replyBody.value = ''
}

async function review() {
  await store.review(route.params.id as string)
}
async function resolve() {
  try {
    await store.resolve(route.params.id as string)
  } catch (e) {
    alert((e as Error).message)
  }
}
async function reopen() {
  await store.reopen(route.params.id as string, '复核重新打开')
}
async function close() {
  await store.close(route.params.id as string)
}

async function saveAssignee() {
  await store.assign(route.params.id as string, assigneeID.value)
  showAssignForm.value = false
}

async function savePriority() {
  await store.changePriority(route.params.id as string, newPriority.value)
}
</script>

<template>
  <div v-if="store.current">
    <div class="toolbar">
      <button class="btn" @click="router.back()">返回</button>
      <h2 style="margin: 0; flex: 1">{{ store.current.title }}</h2>
      <span class="tag" :class="store.current.status">{{ store.current.status }}</span>
      <span class="tag" :class="store.current.priority">{{ store.current.priority }}</span>
    </div>

    <div class="card">
      <div class="status-steps">
        <span v-for="(s, i) in statusOrder" :key="s"
          class="status-step"
          :class="{ active: i === currentStatusIndex, done: i < currentStatusIndex }">
          {{ s }}
        </span>
      </div>
      <p style="white-space: pre-wrap">{{ store.current.body || '（无正文）' }}</p>
      <div class="row" style="gap: 16px; margin-top: 12px">
        <span class="muted">报告人: {{ store.current.reporter_id }}</span>
        <span class="muted">负责人:
          <button class="btn btn-sm" @click="showAssignForm = !showAssignForm">
            {{ store.current.assignee_id || '未指派' }}
          </button>
        </span>
        <span class="muted">创建于 {{ new Date(store.current.created_at).toLocaleString('zh-CN') }}</span>
      </div>
      <div v-if="showAssignForm" class="row" style="margin-top: 8px">
        <input class="input" v-model="assigneeID" placeholder="负责人 ID" style="max-width: 240px" />
        <button class="btn btn-primary btn-sm" @click="saveAssignee">保存</button>
      </div>
      <div class="row" style="gap: 16px; margin-top: 12px">
        <span class="muted">优先级:</span>
        <select class="select" v-model="newPriority" style="max-width: 160px" @change="savePriority">
          <option value="critical">紧急</option>
          <option value="high">高</option>
          <option value="normal">普通</option>
          <option value="low">低</option>
        </select>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">状态操作</h3>
      <div class="row" style="flex-wrap: wrap">
        <button class="btn btn-primary btn-sm" @click="review"
          :disabled="store.current.status !== 'replied' && store.current.status !== 'open'">
          提交复核
        </button>
        <button class="btn btn-primary btn-sm" @click="resolve"
          :disabled="store.current.status !== 'review'">
          解决
        </button>
        <button class="btn btn-sm" @click="reopen"
          :disabled="store.current.status !== 'resolved'">
          复核重新打开
        </button>
        <button class="btn btn-danger btn-sm" @click="close"
          :disabled="store.current.status === 'closed' || store.current.status === 'resolved'">
          关闭
        </button>
      </div>
      <p class="muted" style="margin-top: 8px">
        规则：已解决的批注只能通过复核重新打开；解决前必须至少有一条回复。
      </p>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">回复 ({{ store.current.replies?.length || 0 }})</h3>
      <div class="reply-list">
        <div v-for="r in store.current.replies" :key="r.id" class="reply-item">
          <div class="reply-meta">
            <strong>{{ r.author_id }}</strong> · {{ new Date(r.created_at).toLocaleString('zh-CN') }}
          </div>
          <div>{{ r.body }}</div>
        </div>
        <div v-if="!store.current.replies || store.current.replies.length === 0" class="muted">
          暂无回复
        </div>
      </div>
      <div class="row" style="margin-top: 12px">
        <textarea class="textarea" v-model="replyBody" rows="2" placeholder="输入回复…" style="flex: 1"></textarea>
        <button class="btn btn-primary" @click="reply">回复</button>
      </div>
    </div>
  </div>
</template>
