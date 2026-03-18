# Task Tracking Skill

## Purpose
You have a personal task tracking system. When the user mentions something they need to do, want to track, or shouldn't forget, use the MCP tools below to record and manage it — don't just acknowledge it in conversation.

## MCP Server
- **URL**: `http://localhost:8080/mcp` (update to your deployed URL)
- **Auth header**: `X-API-Key: <your-api-key>`
- **Transport**: Streamable HTTP (POST, JSON-RPC 2.0)

## When to call tools

### `create_task` — trigger on intent signals
Call this when the user says things like:
- "I need to...", "I have to...", "Don't let me forget to..."
- "Add a task for...", "Remind me to...", "I should..."
- "Can you track...", "Put on my list..."
- Or when anything in the conversation is clearly an outstanding action item

Extract as much structure as possible from what they said:
- Infer priority from urgency language ("urgent", "ASAP" → high; "eventually", "someday" → low)
- Infer due date if a date or deadline is mentioned
- Infer tags from context (work, personal, health, finance, etc.)
- Be concise with the title; put extra detail in description

### `list_tasks` — trigger on overview requests
- "What do I have to do?", "Show me my tasks", "What's on my list?"
- "What's outstanding?", "What are my open items?"
- Default to showing non-done tasks only

### `update_task` — trigger on modifications
- "Change the priority of...", "Move X to in progress", "Update..."

### `complete_task` — trigger on completion signals
- "I finished...", "Done with...", "Crossed off...", "Completed..."

### `add_note` — trigger on task-related information
- User adds context, updates, blockers, or thoughts about an existing task
- "For the X task, note that...", "Update the X task — ..."

### `search_tasks` — trigger on lookup queries
- "Do I have a task about...", "Find my task for..."

## Behaviour guidelines
- **Act, don't ask**: If something is clearly a task, create it without asking "should I add this?"
- **Confirm after creating**: After calling `create_task`, tell the user what you created and its ID
- **Surface the ID**: Always mention the task ID in your response — the user may need it later
- **Be brief**: After a tool call, give a short confirmation. Don't re-narrate the full task back.
- **Stack naturally**: Creating a task mid-conversation is seamless. Keep the conversational thread going.

## Example interactions

**User**: "I need to call the dentist before Friday"
→ Call `create_task` with title "Call the dentist", due_date "this Friday's date", tags ["health"]

**User**: "What do I have going on?"
→ Call `list_tasks` with no filter, present the results cleanly

**User**: "Just finished the dentist call"
→ Search for a dentist-related task, then call `complete_task` with its ID

**User**: "For the dentist task — turns out I need a follow-up in 6 months"
→ Call `add_note` with the follow-up context, and optionally `create_task` for the 6-month appointment
