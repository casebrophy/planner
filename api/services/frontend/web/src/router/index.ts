import { createRouter, createWebHistory } from 'vue-router'

const DashboardView = () => import('@/views/DashboardView.vue')
const TaskBoardView = () => import('@/views/TaskBoardView.vue')
const TaskDetailView = () => import('@/views/TaskDetailView.vue')
const ContextBoardView = () => import('@/views/ContextBoardView.vue')
const ContextDetailView = () => import('@/views/ContextDetailView.vue')
const CaptureView = () => import('@/views/CaptureView.vue')
const ClarificationView = () => import('@/views/ClarificationView.vue')
const TodayView = () => import('@/views/TodayView.vue')
const DailyPlanView = () => import('@/views/DailyPlanView.vue')
const SearchView = () => import('@/views/SearchView.vue')
const TransactionBoardView = () => import('@/views/TransactionBoardView.vue')
const SettingsView = () => import('@/views/SettingsView.vue')

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  { path: '/today', name: 'today', component: TodayView },
  { path: '/plan', name: 'plan', component: DailyPlanView },
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
  { path: '/transactions', name: 'transactions', component: TransactionBoardView },
  { path: '/capture', name: 'capture', component: CaptureView },
  { path: '/clarifications', name: 'clarifications', component: ClarificationView },
  { path: '/search', name: 'search', component: SearchView },
  { path: '/settings', name: 'settings', component: SettingsView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
