<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { listBoards, createBoard, listMembers, addMember, removeMember } from '@/api/services'
import type { Board, Membership } from '@/types/models'

const route = useRoute()
const router = useRouter()
const store = useProjectStore()
const boards = ref<Board[]>([])
const members = ref<Membership[]>([])
const showBoardForm = ref(false)
const showMemberForm = ref(false)
const newBoard = ref({ name: '', width: 1440, height: 1024 })
const newMember = ref({ user_id: '', role: 'editor' as 'editor' | 'viewer' | 'owner' })

async function load() {
  const id = route.params.id as string
  await store.fetchOne(id)
  boards.value = await listBoards(id)
  members.value = await listMembers(id)
}

onMounted(load)
watch(() => route.params.id, load)

async function addBoard() {
  if (!newBoard.value.name) return
  const b = await createBoard({
    project_id: route.params.id as string,
    name: newBoard.value.name,
    width: newBoard.value.width,
    height: newBoard.value.height,
  })
  boards.value.push(b)
  newBoard.value = { name: '', width: 1440, height: 1024 }
  showBoardForm.value = false
}

async function archive() {
  await store.archive(route.params.id as string)
  await load()
}

async function addM() {
  if (!newMember.value.user_id) return
  const m = await addMember({
    project_id: route.params.id as string,
    user_id: newMember.value.user_id,
    role: newMember.value.role,
  })
  members.value.push(m)
  newMember.value = { user_id: '', role: 'editor' }
  showMemberForm.value = false
}

async function removeM(id: string) {
  await removeMember(id)
  members.value = members.value.filter((m) => m.id !== id)
}

function openBoard(id: string) {
  router.push(`/boards/${id}`)
}

function boardWidth(b: Board): number {
  if (b.width) return b.width
  if (b.size) return b.size.width
  return b.Size?.Width ?? 0
}
function boardHeight(b: Board): number {
  if (b.height) return b.height
  if (b.size) return b.size.height
  return b.Size?.Height ?? 0
}
</script>

<template>
  <div v-if="store.current">
    <div class="toolbar">
      <h2 style="margin: 0; flex: 1">{{ store.current.name }}</h2>
      <button class="btn" @click="router.push('/projects')">返回</button>
      <button class="btn btn-danger" @click="archive">归档</button>
    </div>
    <div class="card">
      <p class="muted">{{ store.current.description || '（无描述）' }}</p>
      <div class="row" style="gap: 16px; margin-top: 12px">
        <span>状态: <span class="tag" :class="store.current.status">{{ store.current.status === 'active' ? '进行中' : '已归档' }}</span></span>
        <span>创建时间: {{ new Date(store.current.created_at).toLocaleString('zh-CN') }}</span>
      </div>
    </div>

    <div class="toolbar" style="margin-top: 16px">
      <h3 style="margin: 0; flex: 1">画板</h3>
      <button class="btn btn-primary" @click="showBoardForm = !showBoardForm">新建画板</button>
    </div>
    <div v-if="showBoardForm" class="card">
      <div class="row">
        <div class="col field">
          <label>名称</label>
          <input class="input" v-model="newBoard.name" />
        </div>
        <div class="field" style="width: 120px">
          <label>宽度</label>
          <input type="number" class="input" v-model.number="newBoard.width" />
        </div>
        <div class="field" style="width: 120px">
          <label>高度</label>
          <input type="number" class="input" v-model.number="newBoard.height" />
        </div>
      </div>
      <button class="btn btn-primary" @click="addBoard">保存</button>
    </div>
    <div class="grid">
      <div v-for="b in boards" :key="b.id" class="card" style="cursor: pointer" @click="openBoard(b.id)">
        <h4 style="margin: 0 0 4px 0">{{ b.name }}</h4>
        <p class="muted">{{ boardWidth(b) }} x {{ boardHeight(b) }} px</p>
        <span class="tag" :class="b.status">{{ b.status === 'open' ? '打开' : '已关闭' }}</span>
      </div>
    </div>

    <div class="toolbar" style="margin-top: 16px">
      <h3 style="margin: 0; flex: 1">成员</h3>
      <button class="btn btn-primary" @click="showMemberForm = !showMemberForm">添加成员</button>
    </div>
    <div v-if="showMemberForm" class="card">
      <div class="row">
        <div class="col field">
          <label>用户 ID</label>
          <input class="input" v-model="newMember.user_id" placeholder="如 designer-1" />
        </div>
        <div class="field" style="width: 200px">
          <label>角色</label>
          <select class="select" v-model="newMember.role">
            <option value="owner">负责人</option>
            <option value="editor">编辑</option>
            <option value="viewer">查看</option>
          </select>
        </div>
      </div>
      <button class="btn btn-primary" @click="addM">保存</button>
    </div>
    <table class="data card" style="padding: 0">
      <thead>
        <tr><th>用户</th><th>角色</th><th>加入时间</th><th></th></tr>
      </thead>
      <tbody>
        <tr v-for="m in members" :key="m.id">
          <td>{{ m.user_id }}</td>
          <td><span class="tag">{{ m.role }}</span></td>
          <td>{{ new Date(m.joined_at).toLocaleDateString('zh-CN') }}</td>
          <td><button class="btn btn-sm btn-danger" @click="removeM(m.id)">移除</button></td>
        </tr>
        <tr v-if="members.length === 0"><td colspan="4" class="muted" style="text-align:center">暂无成员</td></tr>
      </tbody>
    </table>
  </div>
</template>
