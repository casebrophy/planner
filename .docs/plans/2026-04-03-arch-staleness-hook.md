# Arch Staleness Hook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A pre-commit git hook that blocks commits when `.docs/arch/` files are stale relative to staged source files, using convention-based domain extraction, and prints exact `/go-arch` or `/vue-arch` commands to fix it.

**Architecture:** A pure bash script at `zarf/hooks/pre-commit` (committed to the repo, installed into `.git/hooks/` via `make install-hooks`). It reads staged file paths from `git diff --cached --name-only`, derives arch domain names from path/filename conventions with no hardcoded map, compares modification times, and blocks with a clear message if stale. No AI runs in the hook itself — the hook only detects staleness, not fixes it.

**Tech Stack:** bash (4+), git, GNU/BSD stat (cross-platform mtime), Make

---

## Domain Extraction Conventions

### Backend
Matches any staged file under `business/domain/FOObus/` or `app/domain/FOOapp/` (including nested `stores/FOOdb/`).

```
business/domain/threadbus/threadbus.go     →  thread-backend.md
app/domain/threadapp/threadapp.go          →  thread-backend.md
business/domain/threadbus/stores/threaddb/ →  thread-backend.md
```

Regex: `^(business|app)/domain/([a-z]+)(bus|app)/` — capture group 2 is the domain name.

### Frontend
Matches any staged file under `api/services/frontend/web/src/**/*.{ts,vue}`.

Extraction pipeline (applied to filename without extension):
1. Strip `use` prefix
2. Strip `Store`, `Service`, `View` suffix
3. Extract first PascalCase/camelCase word (regex `[A-Za-z][a-z]*`)
4. Lowercase

```
useTaskBoard.ts      →  use → TaskBoard → Task   → task   → task-frontend.md   ✓
useContextDetail.ts  →  use → ContextDetail → Context → context → context-frontend.md ✓
taskStore.ts         →  Store → task → task       → task   → task-frontend.md   ✓
captureStore.ts      →  Store → capture → capture → capture → capture-frontend.md ✓
CaptureView.vue      →  View → Capture → Capture  → capture → capture-frontend.md ✓
TaskBoardView.vue    →  View → TaskBoard → Task   → task   → task-frontend.md   ✓
calendarEventStore.ts → Store → calendarEvent → calendar → no arch file → skip  ✓
createCRUDStore.ts   →  Store → createCRUD → create → no arch file → skip       ✓
```

**Key rule:** if the derived arch file doesn't exist, the check is skipped silently. This means newly-created domains are never false-positives, and shared utilities (createCRUD, client, router) are ignored.

---

## Files

| Action | Path |
|--------|------|
| Create | `zarf/hooks/pre-commit` |
| Modify | `Makefile` — add `install-hooks` target |
| Modify | `CLAUDE.md` — add hook install to setup, note in session close |

---

## Task 1: Write the hook script

**Files:**
- Create: `zarf/hooks/pre-commit`

- [ ] **Step 1: Create `zarf/hooks/` directory and write the script**

```bash
mkdir -p zarf/hooks
```

Create `zarf/hooks/pre-commit` with this exact content:

```bash
#!/usr/bin/env bash
# zarf/hooks/pre-commit
#
# Blocks commits when .docs/arch/ files are stale relative to staged source files.
# Install: make install-hooks
#
# Convention-based domain extraction (no hardcoded map):
#   business/domain/FOObus/**  →  .docs/arch/FOO-backend.md
#   app/domain/FOOapp/**       →  .docs/arch/FOO-backend.md
#   api/services/frontend/web/src/**/*.{ts,vue}  →  .docs/arch/FOO-frontend.md

set -euo pipefail

ARCH_DIR=".docs/arch"
STALE=()

# mtime: cross-platform file modification time (macOS + Linux)
mtime() {
    stat -f%m "$1" 2>/dev/null || stat -c%Y "$1" 2>/dev/null || echo 0
}

# mark_stale: appends arch_name to STALE if the arch file is older than the source file.
# Skips silently if the arch file doesn't exist (new domain — not yet documented).
mark_stale() {
    local arch_name="$1"
    local source_file="$2"
    local arch_path="$ARCH_DIR/${arch_name}.md"

    [[ -f "$arch_path" ]] || return 0  # new domain, no arch yet — skip

    local arch_mt source_mt
    arch_mt=$(mtime "$arch_path")
    source_mt=$(mtime "$source_file")

    if [[ "$source_mt" -gt "$arch_mt" ]]; then
        STALE+=("$arch_name")
    fi
}

# Process each staged file
while IFS= read -r file; do
    [[ -f "$file" ]] || continue  # skip deletions/renames

    # ── Backend ───────────────────────────────────────────────────────────────
    # Matches: business/domain/FOObus/ or app/domain/FOOapp/ (and nested subdirs)
    if [[ "$file" =~ ^(business|app)/domain/([a-z]+)(bus|app)/ ]]; then
        mark_stale "${BASH_REMATCH[2]}-backend" "$file"
        continue
    fi

    # ── Frontend ──────────────────────────────────────────────────────────────
    # Matches: api/services/frontend/web/src/**/*.ts or *.vue
    if [[ "$file" =~ ^api/services/frontend/web/src/.*\.(ts|vue)$ ]]; then
        base=$(basename "$file")
        name="${base%.*}"  # strip extension

        # Derive domain:
        #   1. strip common prefixes/suffixes
        #   2. extract first PascalCase/camelCase word
        #   3. lowercase
        domain=$(printf '%s' "$name" \
            | sed 's/^use//' \
            | sed 's/Store$//' \
            | sed 's/Service$//' \
            | sed 's/View$//' \
            | sed 's/^\([A-Za-z][a-z]*\).*/\1/' \
            | tr '[:upper:]' '[:lower:]')

        # Only check if arch file exists — skip shared utilities and unmapped domains
        [[ -f "$ARCH_DIR/${domain}-frontend.md" ]] && mark_stale "${domain}-frontend" "$file"
    fi

done < <(git diff --cached --name-only)

# ── Report ────────────────────────────────────────────────────────────────────
[[ ${#STALE[@]} -eq 0 ]] && exit 0

# Deduplicate
unique_stale=()
while IFS= read -r line; do
    unique_stale+=("$line")
done < <(printf '%s\n' "${STALE[@]}" | sort -u)

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  ⚠  Stale arch docs detected. Update before committing. ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
for name in "${unique_stale[@]}"; do
    if [[ "$name" == *-backend ]]; then
        echo "  /go-arch ${name%-backend}"
    else
        echo "  /vue-arch ${name%-frontend}"
    fi
done
echo ""
echo "  After updating, stage the arch files and commit again:"
echo "  git add .docs/arch/ && git commit"
echo ""
exit 1
```

- [ ] **Step 2: Make the script executable**

```bash
chmod +x zarf/hooks/pre-commit
```

- [ ] **Step 3: Smoke test the hook directly (no install yet)**

First, create a fake staged change in a backend domain:
```bash
# Touch a file in threadbus to make it newer than thread-backend.md
touch business/domain/threadbus/threadbus.go
git add business/domain/threadbus/threadbus.go
```

Run the hook manually:
```bash
bash zarf/hooks/pre-commit
```

Expected output:
```
╔══════════════════════════════════════════════════════════╗
║  ⚠  Stale arch docs detected. Update before committing. ║
╚══════════════════════════════════════════════════════════╝

  /go-arch thread

  After updating, stage the arch files and commit again:
  git add .docs/arch/ && git commit

```

Expected exit code: `1`

- [ ] **Step 4: Verify the hook passes when arch is current**

Touch the arch file to make it newer than the source:
```bash
touch .docs/arch/thread-backend.md
git add .docs/arch/thread-backend.md
bash zarf/hooks/pre-commit
echo "Exit: $?"
```

Expected: no output, exit code `0`.

- [ ] **Step 5: Reset the test changes**

```bash
git restore --staged business/domain/threadbus/threadbus.go .docs/arch/thread-backend.md
git restore business/domain/threadbus/threadbus.go
```

- [ ] **Step 6: Commit the hook script**

```bash
git add zarf/hooks/pre-commit
git commit -m "feat: add arch staleness pre-commit hook"
```

---

## Task 2: Add `make install-hooks` and wire it up

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add the `install-hooks` target to Makefile**

Find the end of the development section in `Makefile` (after the existing targets) and add:

```makefile
# ==============================================================================
# Dev tooling

install-hooks:
	cp zarf/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "✓ pre-commit hook installed"
```

- [ ] **Step 2: Run it and verify**

```bash
make install-hooks
```

Expected:
```
✓ pre-commit hook installed
```

```bash
ls -la .git/hooks/pre-commit
```

Expected: file exists and is executable (`-rwxr-xr-x`).

- [ ] **Step 3: End-to-end test through `git commit`**

Touch a frontend file that maps to an existing arch domain:
```bash
touch api/services/frontend/web/src/stores/taskStore.ts
git add api/services/frontend/web/src/stores/taskStore.ts
git commit -m "test"
```

Expected: commit is blocked with:
```
╔══════════════════════════════════════════════════════════╗
║  ⚠  Stale arch docs detected. Update before committing. ║
╚══════════════════════════════════════════════════════════╝

  /vue-arch task
  ...
```

Reset:
```bash
git restore --staged api/services/frontend/web/src/stores/taskStore.ts
git restore api/services/frontend/web/src/stores/taskStore.ts
```

- [ ] **Step 4: Commit the Makefile change**

```bash
git add Makefile
git commit -m "feat: add install-hooks Makefile target"
```

---

## Task 3: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

Two additions needed:

**Addition 1** — Setup section: after the `make dev-up` / `make migrate` commands, add a note that `make install-hooks` must be run on a fresh clone.

Find this block in `CLAUDE.md`:
```markdown
# One-shot local dev (DB + migrate + API + Vite frontend)
make dev-up       # Start everything; Ctrl-C to stop
make dev-down     # Stop the dev database
```

Add after it:
```markdown
# Git hooks (run once after cloning)
make install-hooks  # Installs pre-commit arch staleness check
```

**Addition 2** — Session Completion section: in the "Run quality gates" step, add arch doc guidance.

Find:
```markdown
2. **Run quality gates** (if code changed) - Tests, linters, builds
```

Replace with:
```markdown
2. **Run quality gates** (if code changed) - Tests, linters, builds
   - For each domain touched, run `/go-arch <domain>` (backend) or `/vue-arch <domain>` (frontend)
   - Stage updated arch files before the final commit — the pre-commit hook will block if they're stale
```

- [ ] **Step 1: Make both edits to CLAUDE.md**

(Use the Edit tool to make the two changes above.)

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add install-hooks to setup and arch update to session close protocol"
```

---

## Self-Review

**Spec coverage:**
- [x] Pre-commit hook that blocks on stale arch — Task 1
- [x] Convention-based backend domain extraction (`FOObus`/`FOOapp`) — Task 1
- [x] Convention-based frontend domain extraction (filename pattern) — Task 1
- [x] `/go-arch` and `/vue-arch` both covered in output — Task 1
- [x] `make install-hooks` for setup — Task 2
- [x] CLAUDE.md session close protocol updated — Task 3
- [x] Cross-platform mtime (macOS + Linux) — Task 1, `mtime()` function

**Gaps / known limitations (acceptable):**
- `foundation/claudecli/` and `zarf/sidecar/` changes don't trigger `server-backend.md` staleness — these paths don't match the backend convention. Acceptable: CLAUDE.md session close protocol covers it.
- Frontend false negatives for shared utilities (`createCRUDStore`, `client.ts`, `router/`) — all correctly skipped because no arch file exists for derived domain names.
- Hook is not installed automatically on `git clone` — requires `make install-hooks`. This is standard git hook behavior (hooks are not committed to `.git/hooks/`).
