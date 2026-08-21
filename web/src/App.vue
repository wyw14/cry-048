<script setup lang="ts">
import { onMounted } from 'vue'
import { useReviewStore } from './stores/review'

const review = useReviewStore()
onMounted(() => review.load('version-1'))
</script>

<template>
  <main>
    <header><h1>Design Review</h1><span>{{ review.openCount }} open annotations</span></header>
    <p v-if="review.loading">Loading review snapshot…</p>
    <p v-else-if="review.error" role="alert">{{ review.error }}</p>
    <ul v-else><li v-for="annotation in review.annotations" :key="annotation.id"><strong>{{ annotation.state }}</strong> {{ annotation.body }} <small>#{{ annotation.revision }}</small></li></ul>
  </main>
</template>

<style scoped>
main { max-width: 860px; margin: 2rem auto; font-family: system-ui, sans-serif; }
header { display: flex; justify-content: space-between; border-bottom: 1px solid #ddd; }
li { margin: .75rem 0; }
</style>
