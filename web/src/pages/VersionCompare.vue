<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getBoard, listVersions } from '@/api/services'
import { useAnnotationStore } from '@/stores/annotation'
import type { Board, Version } from '@/types/models'

const route = useRoute()
const router = useRouter()
const annoStore = useAnnotationStore()
const board = ref<Board | null>(null)
const versions = ref<Version[]>([])
const leftID = ref('')
const rightID = ref('')

async function load() {
  const bid = route.params.boardID as string
  board.value = await getBoard(bid)
  versions.value = await listVersions(bid)
  if (versions.value.length >= 2) {
    leftID.value = versions.value[0].id
    rightID.value = versions.value[versions.value.length - 1].id
  } else if (versions.value.length === 1) {
    leftID.value = versions.value[0].id
  }
  await fetchAnnotations()
}

async function fetchAnnotations() {
  if (!leftID.value || !rightID.value) return
  await annoStore.fetch({ board_id: route.params.boardID as string, version_id: leftID.value, page_size: 50 })
  const leftItems = annoStore.items
  await annoStore.fetch({ board_id: route.params.boardID as string, version_id: rightID.value, page_size: 50 })
  const rightItems = annoStore.items
  return { leftItems, rightItems }
}

onMounted(load)

const leftAnnotations = computed(() => annoStore.items.filter((a) => a.version_id === leftID.value))
const rightAnnotations = computed(() => annoStore.items.filter((a) => a.version_id === rightID.value))

async function migrate() {
  if (!leftID.value || !rightID.value) return
  const r = await annoStore.migrate({
    from_version_id: leftID.value,
    to_version_id: rightID.value,
    reason: '版本比较时人工重定位',
  })
  alert(`已迁移 ${r.migrated_count} 条批注`)
  await fetchAnnotations()
}
</script>

<template>
  <div v-if="board">
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">{{ board.name }} - 版本比较</h2>
      <button class="btn" @click="router.back()">返回</button>
    </div>

    <div class="card">
      <div class="row">
        <div class="col field">
          <label>旧版本</label>
          <select class="select" v-model="leftID" @change="fetchAnnotations">
            <option v-for="v in versions" :key="v.id" :value="v.id">v{{ v.number }} {{ v.label }}</option>
          </select>
        </div>
        <div class="col field">
          <label>新版本</label>
          <select class="select" v-model="rightID" @change="fetchAnnotations">
            <option v-for="v in versions" :key="v.id" :value="v.id">v{{ v.number }} {{ v.label }}</option>
          </select>
        </div>
        <button class="btn btn-primary" @click="migrate">从此版本迁移锚点</button>
      </div>
    </div>

    <div class="grid">
      <div class="card">
        <h3 style="margin: 0 0 8px 0">旧版本批注</h3>
        <ul style="margin: 0; padding-left: 16px">
          <li v-for="a in leftAnnotations" :key="a.id">
            <RouterLink :to="`/annotations/${a.id}`">{{ a.title }}</RouterLink>
            <span class="tag" :class="a.status" style="margin-left: 8px">{{ a.status }}</span>
          </li>
          <li v-if="leftAnnotations.length === 0" class="muted">无</li>
        </ul>
      </div>
      <div class="card">
        <h3 style="margin: 0 0 8px 0">新版本批注</h3>
        <ul style="margin: 0; padding-left: 16px">
          <li v-for="a in rightAnnotations" :key="a.id">
            <RouterLink :to="`/annotations/${a.id}`">{{ a.title }}</RouterLink>
            <span class="tag" :class="a.status" style="margin-left: 8px">{{ a.status }}</span>
          </li>
          <li v-if="rightAnnotations.length === 0" class="muted">无</li>
        </ul>
      </div>
    </div>
  </div>
</template>
