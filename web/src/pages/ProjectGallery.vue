<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'

const store = useProjectStore()
const router = useRouter()
const showForm = ref(false)
const name = ref('')
const description = ref('')

async function load() {
  await store.fetch({ page: 1, page_size: 50 })
}

onMounted(load)

async function submit() {
  if (!name.value.trim()) return
  await store.create({ name: name.value, description: description.value })
  name.value = ''
  description.value = ''
  showForm.value = false
  await load()
}

function open(id: string) {
  router.push(`/projects/${id}`)
}
</script>

<template>
  <div>
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">项目画廊</h2>
      <button class="btn btn-primary" @click="showForm = !showForm">
        {{ showForm ? '取消' : '新建项目' }}
      </button>
    </div>

    <div v-if="showForm" class="card">
      <div class="field">
        <label>项目名称</label>
        <input class="input" v-model="name" placeholder="如：首页改版" />
      </div>
      <div class="field">
        <label>项目描述</label>
        <textarea class="textarea" v-model="description" rows="3"></textarea>
      </div>
      <button class="btn btn-primary" @click="submit">保存</button>
    </div>

    <div v-if="store.loading" class="empty">加载中…</div>
    <div v-else-if="store.items.length === 0" class="empty">暂无项目，请新建</div>
    <div v-else class="grid">
      <div v-for="p in store.items" :key="p.id" class="card" style="cursor: pointer" @click="open(p.id)">
        <h3 style="margin: 0 0 8px 0">{{ p.name }}</h3>
        <p class="muted">{{ p.description || '（无描述）' }}</p>
        <div class="row" style="margin-top: 12px; justify-content: space-between">
          <span class="tag" :class="p.status">{{ p.status === 'active' ? '进行中' : '已归档' }}</span>
          <span class="muted">创建于 {{ new Date(p.created_at).toLocaleDateString('zh-CN') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
