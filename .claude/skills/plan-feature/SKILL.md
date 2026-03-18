---
name: plan-feature
description: Directed feature planning for the planner app. Use when the user has decided what to build and wants to make it concrete — updates planning docs and creates an implementation plan. Argument is the feature name (e.g., "email-ingestion", "frontend", "scheduling").
---

# Feature Planning

Plan a specific feature for the planner app. Updates the relevant `.docs/` planning files and produces an implementation plan.

## Setup

1. Read `.docs/TOC.md`
2. Search TOC for all entries matching the feature name argument
3. Read only the matched `.docs/` sections
4. Read the relevant `.docs/arch/` file if the domain already exists
5. Read the "Planner App Context" section of `CLAUDE.md` for current state

## Process

1. **Summarize what the docs say** about this feature — what's already designed, what's unspecified
2. **Guided conversation** — walk through requirements, constraints, and trade-offs with the user
3. **When aligned, update docs:**
   - Modify the relevant `.docs/` file sections in place (follow the compressed format — bullets, tables, no prose)
   - Add new `##` sections if needed
   - Update `.docs/TOC.md` if new sections were added
   - Update "Planner App Context" in `CLAUDE.md` if phase status changes
4. **Create implementation plan** — invoke the `superpowers:writing-plans` skill

## Doc update rules

- Follow the existing compressed format in each file (see any `.docs/` file for reference)
- Every new `##` section must be added to `TOC.md` in the appropriate dimension(s)
- Keep DDL exact and complete — no pseudo-SQL
- Keep Go interfaces exact and complete — no pseudocode
- One source of truth per fact — don't duplicate information across docs
