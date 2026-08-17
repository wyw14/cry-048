<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRound, listRounds, createSnapshot, setConclusion, closeRound, exportReviewMinutesCsv } from '@/api/services'
import type { ReviewRound } from '@/types/models'

const route = useRoute()
const router = useRouter()
const round = ref<ReviewRound | null>(null)
const rounds = ref<ReviewRound[]>([])
const newConclusion = ref('')
const newRecommendation = ref<'approve' | 'approve_with_conditions' | 'reject' | 'defer'>('approve')

async function load() {
  const id = route.params.id as string
  round.value = await getRound(id)
  newConclusion.value = round.value.conclusion
  if (round.value.recommendation) {
    newRecommendation.value = round.value.recommendation as typeof newRecommendation.value
  }
  if (round.value) {
    rounds.value = await listRounds(round.value.board_id)
  }
}

onMounted(load)

async function snapshot() {
  if (!round.value) return
  await createSnapshot(round.value.id, {
    project_id: round.value.project_id,
    version_id: round.value.snapshots[0]?.version_id ?? '',
  })
  await load()
}

async function submitConclusion() {
  if (!round.value) return
  await setConclusion(round.value.id, {
    conclusion: newConclusion.value,
    recommendation: newRecommendation.value,
  })
  await load()
}

async function close() {
  if (!round.value) return
  await closeRound(round.value.id)
  await load()
}

async function exportCsv() {
  if (!round.value) return
  const csv = await exportReviewMinutesCsv(round.value.id)
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `review_minutes_${round.value.id}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

function open(id: string) {
  router.push(`/reviews/${id}`)
}
</script>

<template>
  <div v-if="round">
    <div class="toolbar">
      <button class="btn" @click="router.back()">返回</button>
      <h2 style="margin: 0; flex: 1">{{ round.title }}</h2>
      <button class="btn btn-sm" @click="exportCsv">导出纪要 CSV</button>
      <span class="tag" :class="round.status">{{ round.status === 'open' ? '进行中' : '已关闭' }}</span>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">汇总结论</h3>
      <p v-if="round.conclusion">{{ round.conclusion }}</p>
      <p v-else class="muted">尚未填写结论</p>
      <div class="row" style="gap: 16px; margin-top: 12px">
        <span>发布建议: <span class="tag">{{ round.recommendation || '-' }}</span></span>
        <span v-if="round.decided_at" class="muted">决策于 {{ new Date(round.decided_at).toLocaleString('zh-CN') }}</span>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">设置结论</h3>
      <div class="field">
        <label>结论</label>
        <textarea class="textarea" v-model="newConclusion" rows="3"></textarea>
      </div>
      <div class="field">
        <label>发布建议</label>
        <select class="select" v-model="newRecommendation">
          <option value="approve">通过</option>
          <option value="approve_with_conditions">有条件通过</option>
          <option value="reject">不通过</option>
          <option value="defer">暂缓</option>
        </select>
      </div>
      <div class="row">
        <button class="btn btn-primary" @click="submitConclusion" :disabled="round.status === 'closed'">保存结论</button>
        <button class="btn" @click="snapshot" :disabled="round.status === 'closed'">创建快照</button>
        <button class="btn btn-danger" @click="close" :disabled="round.status === 'closed'">关闭轮次</button>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">快照</h3>
      <table class="data">
        <thead>
          <tr><th>版本</th><th>创建时间</th><th>创建者</th><th>状态计数</th></tr>
        </thead>
        <tbody>
          <tr v-for="s in round.snapshots" :key="s.id">
            <td>{{ s.version_id }}</td>
            <td>{{ new Date(s.created_at).toLocaleString('zh-CN') }}</td>
            <td>{{ s.created_by }}</td>
            <td>
              <span v-for="(v, k) in s.counts" :key="k" class="tag" style="margin-right: 4px">{{ k }}: {{ v }}</span>
            </td>
          </tr>
          <tr v-if="round.snapshots.length === 0"><td colspan="4" class="muted" style="text-align: center">暂无快照</td></tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 8px 0">评审轮次列表</h3>
      <ul style="margin: 0; padding-left: 16px">
        <li v-for="r in rounds" :key="r.id">
          <a href="#" @click.prevent="open(r.id)">{{ r.title }}</a>
          <span class="tag" :class="r.status" style="margin-left: 8px">{{ r.status }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>
