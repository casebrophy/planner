# Capture Frontend Architecture

## Overview

Top-level navigation hub for capture modes (Task, Context, Receipt).

## Components

### CaptureView.vue
Navigation view showing three capture options.

**Route:** `/capture`

**Props:** None  
**Emits:** None (navigation via router)

**Features:**
- Displays three buttons: Task, Context, Receipt
- Task button → `/tasks`
- Context button → `/contexts`
- Receipt button → `/receipts`

**Styling:**
- Centered layout, dark theme
- 20px top padding, 6px gap between buttons
- Buttons are `bg-gray-800` with hover effect except Receipt which is `bg-blue-600`

## Router Integration

Added to `/Users/casebrophy/personal/planner/api/services/frontend/web/src/router/index.ts`:
```ts
{ path: '/capture', name: 'capture', component: CaptureView }
```

## Related Views

- `/receipts` → ReceiptCaptureView (multi-step receipt capture)
- `/tasks` → TaskBoardView (existing)
- `/contexts` → ContextBoardView (existing)

## No State or Services

CaptureView is a simple navigation component with no Pinia store dependencies or API calls.
