<script setup lang="ts">
import { onMounted, ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getBoard, listVersions, createVersion, publishVersion } from '@/api/services'
import { useAnnotationStore } from '@/stores/annotation'
import type { Board, Version, Annotation } from '@/types/models'

const route = useRoute()
const router = useRouter()
const annoStore = useAnnotationStore()
const board = ref<Board | null>(null)
const versions = ref<Version[]>([])
const currentVersionID = ref('')
const showVersionForm = ref(false)
const newVersion = ref({ number: 1, label: '', preview_key: '', notes: '' })

async function load() {
  const id = route.params.id as string
  board.value = await getBoard(id)
  versions.value = await listVersions(id)
  const latest = versions.value[versions.value.length - 1]
  currentVersionID.value = latest?.id ?? ''
  await annoStore.fetch({ board_id: id, version_id: currentVersionID.value, page_size: 50 })
}

onMounted(load)
watch(() => route.params.id, load)

async function switchVersion(vid: string) {
  currentVersionID.value = vid
  await annoStore.fetch({ board_id: route.params.id as string, version_id: vid, page_size: 50 })
}

async function addVersion() {
  const v = await createVersion({
    board_id: route.params.id as string,
    number: newVersion.value.number,
    label: newVersion.value.label,
    preview_key: newVersion.value.preview_key,
    notes: newVersion.value.notes,
  })
  versions.value.push(v)
  newVersion.value = { number: versions.value.length + 1, label: '', preview_key: '', notes: '' }
  showVersionForm.value = false
}

async function publish(id: string) {
  const v = await publishVersion(id)
  const idx = versions.value.findIndex((x) => x.id === id)
  if (idx >= 0) versions.value[idx] = v
  await load()
}

async function migrate(fromID: string) {
  if (!currentVersionID.value || currentVersionID.value === fromID) return
  const r = await annoStore.migrate({
    from_version_id: fromID,
    to_version_id: currentVersionID.value,
    reason: '版本切换迁移',
  })
  alert(`已迁移 ${r.migrated_count} 条批注`)
  await load()
}

function openAnnotation(id: string) {
  router.push(`/annotations/${id}`)
}

const canvasAnnotations = computed<Annotation[]>(() => annoStore.items)

function anchorPoint(a: Annotation): { x: number; y: number } | null {
  if (a.Anchor?.Point) return { x: a.Anchor.Point.X, y: a.Anchor.Point.Y }
  if (a.anchor?.point) return { x: a.anchor.point.x, y: a.anchor.point.y }
  return null
}

function anchorRegion(a: Annotation): { x: number; y: number; width: number; height: number } | null {
  if (a.Anchor?.Region) return { x: a.Anchor.Region.X, y: a.Anchor.Region.Y, width: a.Anchor.Region.Width, height: a.Anchor.Region.Height }
  if (a.anchor?.region) return a.anchor.region
  return null
}

function boardWidth(): number {
  if (!board.value) return 1440
  if (board.value.width) return board.value.width
  if (board.value.size) return board.value.size.width
  return board.value.Size?.Width ?? 1440
}
function boardHeight(): number {
  if (!board.value) return 1024
  if (board.value.height) return board.value.height
  if (board.value.size) return board.value.size.height
  return board.value.Size?.Height ?? 1024
}
</script>

<template>
  <div v-if="board">
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">{{ board.name }}</h2>
      <button class="btn" @click="router.push(`/projects/${board.project_id}`)">返回项目</button>
      <button class="btn" @click="router.push(`/versions/compare/${board.id}`)">版本比较</button>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">画板尺寸</h3>
      <p class="muted">{{ boardWidth() }} x {{ boardHeight() }} px</p>
      <div class="row" style="margin-top: 12px; flex-wrap: wrap">
        <button v-for="v in versions" :key="v.id"
          class="btn btn-sm"
          :class="{ 'btn-primary': v.id === currentVersionID }"
          @click="switchVersion(v.id)">
          v{{ v.number }} {{ v.label }}
        </button>
        <button class="btn btn-sm" @click="showVersionForm = !showVersionForm">+ 新版本</button>
      </div>
    </div>

    <div v-if="showVersionForm" class="card">
      <div class="row">
        <div class="field" style="width: 120px">
          <label>版本号</label>
          <input type="number" class="input" v-model.number="newVersion.number" />
        </div>
        <div class="col field">
          <label>标签</label>
          <input class="input" v-model="newVersion.label" placeholder="如：修订一版" />
        </div>
      </div>
      <div class="field">
        <label>预览图标识</label>
        <input class="input" v-model="newVersion.preview_key" placeholder="如: preview-v2.png" />
      </div>
      <div class="field">
        <label>备注</label>
        <textarea class="textarea" v-model="newVersion.notes" rows="2"></textarea>
      </div>
      <button class="btn btn-primary" @click="addVersion">创建</button>
    </div>

    <div class="card">
      <div class="row" style="justify-content: space-between">
        <h3 style="margin: 0">画布预览</h3>
        <div>
          <span v-for="v in versions" :key="v.id" class="tag" :class="v.status" style="margin-left: 4px">
            v{{ v.number }}: {{ v.status }}
            <button v-if="v.status === 'draft'" class="btn btn-sm" @click="publish(v.id)">发布</button>
            <button v-else-if="v.status === 'published'" class="btn btn-sm" @click="migrate(v.id)">从此版本迁移</button>
          </span>
        </div>
      </div>
      <div class="canvas-frame" :style="{ width: boardWidth() + 'px', height: Math.min(boardHeight(), 600) + 'px' }">
        <div v-if="canvasAnnotations.length === 0" class="empty" style="padding: 80px 16px">当前版本暂无批注</div>
        <template v-for="a in canvasAnnotations" :key="a.id">
          <div v-if="anchorPoint(a)"
            class="canvas-point"
            :style="{ left: anchorPoint(a)!.x + 'px', top: anchorPoint(a)!.y + 'px' }"
            :title="a.title"
            @click="openAnnotation(a.id)">
          </div>
          <div v-else-if="anchorRegion(a)"
            class="canvas-annotation"
            :style="{ left: anchorRegion(a)!.x + 'px', top: anchorRegion(a)!.y + 'px', width: anchorRegion(a)!.width + 'px', height: anchorRegion(a)!.height + 'px' }"
            :title="a.title"
            @click="openAnnotation(a.id)">
            <small style="color: var(--primary)">{{ a.title }}</small>
          </div>
        </template>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">批注列表 ({{ annoStore.total }})</h3>
      <table class="data">
        <thead>
          <tr><th>标题</th><th>状态</th><th>优先级</th><th>负责人</th><th>截止</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in annoStore.items" :key="a.id" style="cursor: pointer" @click="openAnnotation(a.id)">
            <td>{{ a.title }}</td>
            <td><span class="tag" :class="a.status">{{ a.status }}</span></td>
            <td><span class="tag" :class="a.priority">{{ a.priority }}</span></td>
            <td>{{ a.assignee_id || '-' }}</td>
            <td>{{ a.due_at ? new Date(a.due_at).toLocaleDateString('zh-CN') : '-' }}</td>
          </tr>
          <tr v-if="annoStore.items.length === 0"><td colspan="5" class="muted" style="text-align:center">暂无批注</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
