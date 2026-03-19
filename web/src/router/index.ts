import { createRouter, createWebHistory } from 'vue-router'

const DashboardView = () => import('@/views/DashboardView.vue')
const TaskBoardView = () => import('@/views/TaskBoardView.vue')
const TaskDetailView = () => import('@/views/TaskDetailView.vue')
const ContextBoardView = () => import('@/views/ContextBoardView.vue')
const ContextDetailView = () => import('@/views/ContextDetailView.vue')
const CaptureView = () => import('@/views/CaptureView.vue')

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  {
    path: '/tasks',
    name: 'tasks',
    component: TaskBoardView,
    children: [{ path: ':id', name: 'task-detail', component: TaskDetailView, props: true }],
  },
  {
    path: '/contexts',
    name: 'contexts',
    component: ContextBoardView,
    children: [
      { path: ':id', name: 'context-detail', component: ContextDetailView, props: true },
    ],
  },
  { path: '/capture', name: 'capture', component: CaptureView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
