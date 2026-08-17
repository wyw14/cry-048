import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/projects' },
  { path: '/projects', name: 'gallery', component: () => import('@/pages/ProjectGallery.vue') },
  { path: '/projects/:id', name: 'project-detail', component: () => import('@/pages/ProjectDetail.vue') },
  { path: '/boards/:id', name: 'canvas-review', component: () => import('@/pages/CanvasReview.vue') },
  { path: '/annotations/:id', name: 'annotation-detail', component: () => import('@/pages/AnnotationDetail.vue') },
  { path: '/versions/compare/:boardID', name: 'version-compare', component: () => import('@/pages/VersionCompare.vue') },
  { path: '/reviews/:id', name: 'review-summary', component: () => import('@/pages/ReviewSummary.vue') },
  { path: '/todos', name: 'my-todos', component: () => import('@/pages/MyTodos.vue') },
  { path: '/settings', name: 'settings', component: () => import('@/pages/Settings.vue') },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
