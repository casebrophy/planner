# PWA + Responsive Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the planner app installable as a PWA and add responsive layout (sidebar on desktop, bottom tab bar on mobile) that maps directly to the Phase 4b Capacitor mobile spec.

**Architecture:** `vite-plugin-pwa` generates the service worker and injects the manifest. A `useShell` composable detects mobile vs desktop via screen width (swappable to `Capacitor.isNativePlatform()` later). `AppShell.vue` conditionally renders `AppSidebar` or `MobileTabBar`. Router default route switches based on shell.

**Tech Stack:** vite-plugin-pwa, workbox (via plugin), Vue 3 composables, Tailwind responsive utilities

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `web/public/manifest.json` | PWA manifest (name, icons, theme, display) |
| Create | `web/public/icons/icon-192.svg` | App icon 192x192 (SVG for now — works as PWA icon) |
| Create | `web/public/icons/icon-512.svg` | App icon 512x512 |
| Modify | `web/index.html` | PWA meta tags, manifest link, apple-mobile-web-app tags |
| Modify | `web/package.json` | Add `vite-plugin-pwa` dependency |
| Modify | `web/vite.config.ts` | Configure VitePWA plugin |
| Create | `web/src/composables/useShell.ts` | Shell detection: mobile vs desktop |
| Create | `web/src/components/layout/MobileTabBar.vue` | Bottom 5-tab navigation bar |
| Modify | `web/src/components/layout/AppShell.vue` | Switch between sidebar/tabbar based on shell |
| Modify | `web/src/router/index.ts` | Dynamic default route based on shell |
| Modify | `api/services/frontend/main.go` | Cache-Control headers for service worker correctness |

---

### Task 1: Install vite-plugin-pwa

**Files:**
- Modify: `web/package.json`
- Modify: `web/vite.config.ts`

- [ ] **Step 1: Install the dependency**

Run from `web/`:
```bash
npm install -D vite-plugin-pwa
```

- [ ] **Step 2: Configure the plugin in vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: false, // we provide our own in public/
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
        navigateFallback: 'index.html',
        navigateFallbackDenylist: [/^\/api\//],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

Key decisions:
- `manifest: false` — we maintain our own `manifest.json` in `public/` for full control
- `navigateFallbackDenylist: [/^\/api\//]` — don't cache API calls
- `registerType: 'autoUpdate'` — new version auto-activates without user prompt

- [ ] **Step 3: Verify build still works**

```bash
cd web && npm run build
```

Expected: Build succeeds, `dist/` contains `sw.js` and `workbox-*.js`

- [ ] **Step 4: Commit**

```bash
git add web/package.json web/package-lock.json web/vite.config.ts
git commit -m "feat: add vite-plugin-pwa for service worker generation"
```

---

### Task 2: PWA manifest, icons, and meta tags

**Files:**
- Create: `web/public/manifest.json`
- Create: `web/public/icons/icon-192.svg`
- Create: `web/public/icons/icon-512.svg`
- Modify: `web/index.html`

- [ ] **Step 1: Create the public directory and icons**

```bash
mkdir -p web/public/icons
```

- [ ] **Step 2: Create icon-192.svg**

Create `web/public/icons/icon-192.svg` — a simple dark circle with "P" lettermark:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 192 192">
  <rect width="192" height="192" rx="40" fill="#111827"/>
  <text x="96" y="128" font-family="system-ui, sans-serif" font-size="120" font-weight="700" fill="#f9fafb" text-anchor="middle">P</text>
</svg>
```

- [ ] **Step 3: Create icon-512.svg**

Create `web/public/icons/icon-512.svg` — same design at 512:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <rect width="512" height="512" rx="96" fill="#111827"/>
  <text x="256" y="340" font-family="system-ui, sans-serif" font-size="320" font-weight="700" fill="#f9fafb" text-anchor="middle">P</text>
</svg>
```

- [ ] **Step 4: Create manifest.json**

Create `web/public/manifest.json`:

```json
{
  "name": "Planner",
  "short_name": "Planner",
  "description": "Personal intelligence layer",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#030712",
  "theme_color": "#111827",
  "icons": [
    {
      "src": "/icons/icon-192.svg",
      "sizes": "192x192",
      "type": "image/svg+xml",
      "purpose": "any"
    },
    {
      "src": "/icons/icon-512.svg",
      "sizes": "512x512",
      "type": "image/svg+xml",
      "purpose": "any maskable"
    }
  ]
}
```

Colors match the app's dark theme (`gray-950` = `#030712`, `gray-900` = `#111827`).

- [ ] **Step 5: Update index.html with PWA meta tags**

Replace the full `web/index.html`:

```html
<!DOCTYPE html>
<html lang="en" class="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover" />
    <title>Planner</title>

    <!-- PWA -->
    <link rel="manifest" href="/manifest.json" />
    <meta name="theme-color" content="#111827" />
    <link rel="icon" type="image/svg+xml" href="/icons/icon-192.svg" />

    <!-- iOS -->
    <meta name="apple-mobile-web-app-capable" content="yes" />
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
    <meta name="apple-mobile-web-app-title" content="Planner" />
    <link rel="apple-touch-icon" href="/icons/icon-192.svg" />
  </head>
  <body class="bg-gray-950 text-gray-100 antialiased">
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

Key additions:
- `viewport-fit=cover` — needed for iOS safe areas (notch/home indicator)
- `apple-mobile-web-app-capable` — makes it open as standalone on iOS
- `black-translucent` status bar — blends with our dark theme
- manifest link

- [ ] **Step 6: Build and verify manifest is served**

```bash
cd web && npm run build && ls dist/manifest.json dist/icons/
```

Expected: `manifest.json` and icon files present in `dist/`.

- [ ] **Step 7: Commit**

```bash
git add web/public/ web/index.html
git commit -m "feat: add PWA manifest, icons, and mobile meta tags"
```

---

### Task 3: Shell detection composable

**Files:**
- Create: `web/src/composables/useShell.ts`

- [ ] **Step 1: Create useShell composable**

Create `web/src/composables/useShell.ts`:

```typescript
import { ref, onMounted, onUnmounted } from 'vue'

const MOBILE_BREAKPOINT = 768

const isMobile = ref(false)

function update() {
  isMobile.value = window.innerWidth < MOBILE_BREAKPOINT
}

let listeners = 0

export function useShell() {
  onMounted(() => {
    if (listeners === 0) {
      update()
      window.addEventListener('resize', update)
    }
    listeners++
  })

  onUnmounted(() => {
    listeners--
    if (listeners === 0) {
      window.removeEventListener('resize', update)
    }
  })

  return { isMobile }
}
```

Design notes:
- Shared reactive `isMobile` ref — all components see the same value
- Single resize listener with refcount — no duplicate handlers
- 768px breakpoint matches Tailwind's `md:` — sidebar shows at `md` and above
- When Capacitor is added later, replace the body with: `isMobile.value = Capacitor.isNativePlatform() || window.innerWidth < MOBILE_BREAKPOINT`

- [ ] **Step 2: Commit**

```bash
git add web/src/composables/useShell.ts
git commit -m "feat: add useShell composable for mobile/desktop detection"
```

---

### Task 4: Mobile tab bar component

**Files:**
- Create: `web/src/components/layout/MobileTabBar.vue`

- [ ] **Step 1: Create MobileTabBar.vue**

Create `web/src/components/layout/MobileTabBar.vue`:

```vue
<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useClarificationStore } from '@/stores/clarificationStore'
import { onMounted, onUnmounted } from 'vue'

const route = useRoute()
const clarificationStore = useClarificationStore()

let countInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  clarificationStore.fetchPendingCount()
  countInterval = setInterval(() => clarificationStore.fetchPendingCount(), 60000)
})

onUnmounted(() => {
  if (countInterval) clearInterval(countInterval)
})

const tabs = [
  { name: 'Capture', path: '/capture', icon: 'plus-circle' },
  { name: 'Today', path: '/today', icon: 'sun' },
  { name: 'Contexts', path: '/contexts', icon: 'layers' },
  { name: 'Search', path: '/search', icon: 'search' },
  { name: 'Settings', path: '/settings', icon: 'settings' },
]

function isActive(path: string): boolean {
  return route.path.startsWith(path)
}
</script>

<template>
  <nav class="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-800 z-40 pb-[env(safe-area-inset-bottom)]">
    <div class="flex items-center justify-around h-14">
      <router-link
        v-for="tab in tabs"
        :key="tab.path"
        :to="tab.path"
        class="flex flex-col items-center justify-center flex-1 h-full text-xs transition-colors"
        :class="isActive(tab.path) ? 'text-gray-100' : 'text-gray-500'"
      >
        <!-- Capture / plus-circle -->
        <svg
          v-if="tab.icon === 'plus-circle'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <!-- Today / sun -->
        <svg
          v-else-if="tab.icon === 'sun'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
          />
        </svg>
        <!-- Contexts / layers -->
        <svg
          v-else-if="tab.icon === 'layers'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
          />
        </svg>
        <!-- Search -->
        <svg
          v-else-if="tab.icon === 'search'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <!-- Settings -->
        <svg
          v-else-if="tab.icon === 'settings'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
          />
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
          />
        </svg>
        <span class="mt-0.5">{{ tab.name }}</span>
      </router-link>
    </div>
  </nav>
</template>
```

Key details:
- 5 tabs matching Phase 4b spec: Capture, Today, Contexts, Search, Settings
- `pb-[env(safe-area-inset-bottom)]` — respects iPhone home indicator
- Fixed bottom positioning, same `z-40` as sidebar
- Icons reuse the exact SVG paths from `AppSidebar.vue`

- [ ] **Step 2: Commit**

```bash
git add web/src/components/layout/MobileTabBar.vue
git commit -m "feat: add MobileTabBar component with 5-tab Phase 4b layout"
```

---

### Task 5: Responsive AppShell

**Files:**
- Modify: `web/src/components/layout/AppShell.vue`

- [ ] **Step 1: Update AppShell.vue to switch layouts**

Replace `web/src/components/layout/AppShell.vue`:

```vue
<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import MobileTabBar from './MobileTabBar.vue'
import { useShell } from '@/composables/useShell'
import { ref, onMounted } from 'vue'

const { isMobile } = useShell()

const collapsed = ref(false)

onMounted(() => {
  const saved = localStorage.getItem('sidebar-collapsed')
  if (saved !== null) collapsed.value = saved === 'true'
})

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem('sidebar-collapsed', String(collapsed.value))
}
</script>

<template>
  <div class="flex h-screen bg-gray-950">
    <!-- Desktop: sidebar -->
    <template v-if="!isMobile">
      <AppSidebar
        :collapsed="collapsed"
        @toggle="toggleSidebar"
      />
      <main
        class="flex-1 overflow-auto"
        :class="collapsed ? 'ml-16' : 'ml-60'"
      >
        <router-view />
      </main>
    </template>

    <!-- Mobile: full-width content + bottom tab bar -->
    <template v-else>
      <main class="flex-1 overflow-auto pb-14">
        <router-view />
      </main>
      <MobileTabBar />
    </template>
  </div>
</template>
```

Changes from original:
- Imports `useShell` and `MobileTabBar`
- `v-if="!isMobile"` shows sidebar layout on desktop
- `v-else` shows full-width content + bottom tab bar on mobile
- `pb-14` on mobile main — prevents content from hiding behind the tab bar

- [ ] **Step 2: Build and verify**

```bash
cd web && npm run build
```

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/components/layout/AppShell.vue
git commit -m "feat: responsive AppShell — sidebar on desktop, tab bar on mobile"
```

---

### Task 6: Dynamic default route

**Files:**
- Modify: `web/src/router/index.ts`

- [ ] **Step 1: Update router to detect shell for redirect**

Replace `web/src/router/index.ts`:

```typescript
import { createRouter, createWebHistory } from 'vue-router'

const MOBILE_BREAKPOINT = 768

const DashboardView = () => import('@/views/DashboardView.vue')
const TaskBoardView = () => import('@/views/TaskBoardView.vue')
const TaskDetailView = () => import('@/views/TaskDetailView.vue')
const ContextBoardView = () => import('@/views/ContextBoardView.vue')
const ContextDetailView = () => import('@/views/ContextDetailView.vue')
const CaptureView = () => import('@/views/CaptureView.vue')
const ClarificationView = () => import('@/views/ClarificationView.vue')
const TodayView = () => import('@/views/TodayView.vue')
const SearchView = () => import('@/views/SearchView.vue')
const TransactionBoardView = () => import('@/views/TransactionBoardView.vue')
const SettingsView = () => import('@/views/SettingsView.vue')

const routes = [
  {
    path: '/',
    redirect: () => {
      return window.innerWidth < MOBILE_BREAKPOINT ? '/capture' : '/dashboard'
    },
  },
  { path: '/dashboard', name: 'dashboard', component: DashboardView },
  { path: '/today', name: 'today', component: TodayView },
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
```

Only change: the `/` redirect is now a function that checks screen width. On mobile it opens to Capture (the primary mobile action), on desktop to Dashboard.

Note: uses `window.innerWidth` directly rather than importing `useShell` because the composable requires a component lifecycle (onMounted). The redirect runs before any component mounts. When Capacitor is added, this becomes `Capacitor.isNativePlatform() || window.innerWidth < MOBILE_BREAKPOINT`.

- [ ] **Step 2: Build and verify**

```bash
cd web && npm run build
```

Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add web/src/router/index.ts
git commit -m "feat: dynamic default route — /capture on mobile, /dashboard on desktop"
```

---

### Task 7: Go server cache headers for service worker

**Files:**
- Modify: `api/services/frontend/main.go`

- [ ] **Step 1: Add cache headers to spaHandler**

In `api/services/frontend/main.go`, replace the `spaHandler` function:

```go
// spaHandler serves static files from dir. If the requested file does not
// exist, it falls back to index.html for SPA client-side routing.
// Service worker and manifest are served with no-cache to ensure updates propagate.
func spaHandler(dir string) http.Handler {
	fs := http.Dir(dir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		// Service worker and manifest must not be cached by intermediaries
		// so that updates propagate immediately.
		switch path {
		case "/sw.js", "/manifest.json":
			w.Header().Set("Cache-Control", "no-cache")
		}

		f, err := fs.Open(path)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		f.Close()

		http.FileServer(fs).ServeHTTP(w, r)
	})
}
```

Why: Without `no-cache` on `sw.js`, browsers may serve a stale service worker from their HTTP cache, delaying updates by hours. The workbox-generated SW handles its own internal versioning, but it needs to actually be fetched to check.

- [ ] **Step 2: Verify Go builds**

```bash
go build ./api/services/frontend/...
```

Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/main.go
git commit -m "feat: no-cache headers for sw.js and manifest.json"
```

---

### Task 8: Full build verification and lint

**Files:** None (verification only)

- [ ] **Step 1: Run frontend lint**

```bash
cd web && npm run lint
```

Expected: No errors.

- [ ] **Step 2: Run frontend build**

```bash
cd web && npm run build
```

Expected: Build succeeds. `dist/` contains `sw.js`, `manifest.json`, `icons/`.

- [ ] **Step 3: Run Go build**

```bash
go build ./...
```

Expected: All Go packages build.

- [ ] **Step 4: Verify SW and manifest in dist**

```bash
ls web/dist/sw.js web/dist/manifest.json web/dist/icons/
```

Expected: All files present.

- [ ] **Step 5: Commit any lint fixes if needed**

```bash
git add -A && git status
# Only commit if there are changes from lint --fix
```

---

### Task 9: Test on device

**Files:** None (manual verification)

- [ ] **Step 1: Start the backend and frontend**

```bash
make db-up && make dev &
make frontend-dev
```

Or if using Docker:
```bash
make up
```

- [ ] **Step 2: Open on phone**

Navigate to `http://<your-machine-ip>:3000` on your phone's browser.

Verify:
- Bottom tab bar shows (5 tabs: Capture, Today, Contexts, Search, Settings)
- No sidebar visible
- Navigating between tabs works
- Default route goes to Capture

- [ ] **Step 3: Add to Home Screen**

On iOS Safari: Share → Add to Home Screen → Add
On Android Chrome: Menu → Add to Home Screen

Verify:
- App opens in standalone mode (no browser chrome)
- Status bar blends with dark theme
- App icon shows the "P" lettermark

- [ ] **Step 4: Test on desktop**

Navigate to `http://localhost:3000` on your desktop browser.

Verify:
- Sidebar shows (not tab bar)
- Default route goes to Dashboard
- Sidebar collapse/expand still works
