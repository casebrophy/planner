package extractor

import (
	"fmt"
	"time"
)

// BuildEmailExtractionPrompt builds the prompt for email AI extraction.
// Shared by all extractor implementations.
func BuildEmailExtractionPrompt(fromAddress, subject, bodyText string, contextsJSON []byte) string {
	return fmt.Sprintf(`Analyze this email and extract structured data. Return ONLY valid JSON with no other text.

Email:
From: %s
Subject: %s
Body:
%s

Active contexts (match this email to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary",
  "sender_name": "sender's name or empty string",
  "sender_domain": "sender's domain from email address",
  "action_items": [{"title": "short title", "description": "detail", "priority": "low|medium|high|urgent", "interpretations": ["interpretation1", "interpretation2"]}],
  "deadlines": [{"description": "what is due", "date": "YYYY-MM-DD or natural language if ambiguous", "is_ambiguous": false}],
  "suggested_context_keywords": ["keyword1", "keyword2"],
  "sentiment": "positive|neutral|negative|mixed",
  "suggested_context_id": "UUID of best matching context or null",
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": "title for new context if suggest_new_context is true"
}

Rules:
- Set context_confidence to a value between 0.0 and 1.0 reflecting how well this email matches the suggested context
- If no existing context matches well, set suggest_new_context to true and provide a suggested_context_title
- Set is_ambiguous on deadlines when the date is relative or vague (e.g. "end of month", "soon", "next week")
- Include interpretations array on action_items only when the item is genuinely ambiguous (could be a pleasantry vs. real task)`, fromAddress, subject, bodyText, string(contextsJSON))
}

// buildGenericTextExtractionPrompt builds the fallback prompt for text/voice AI extraction.
// Used when typeHint is empty or unrecognized.
func buildGenericTextExtractionPrompt(text string, contextsJSON []byte, now time.Time) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)

	return fmt.Sprintf(`This is a voice capture from the user. Extract tasks, events, deadlines, and context information. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Voice capture:
%s

Active contexts (match this input to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary of the voice capture",
  "action_items": [{"title": "short title", "description": "detail", "priority": "low|medium|high|urgent", "interpretations": ["interpretation1", "interpretation2"]}],
  "deadlines": [{"description": "what is due", "date": "YYYY-MM-DD or natural language if ambiguous", "is_ambiguous": false}],
  "events": [{"title": "event name", "description": "", "location": "", "starts_at": "2026-04-01T14:00:00Z", "ends_at": "2026-04-01T15:00:00Z (optional)", "all_day": false, "is_ambiguous": false}],
  "notes": [{"content": "the information captured", "suggested_tags": ["tag1", "tag2"]}],
  "ambiguous_references": [{"original_text": "that thing", "reference_type": "pronoun|vague_noun|implicit"}],
  "suggested_context_keywords": ["keyword1", "keyword2"],
  "suggested_context_id": "UUID of best matching context or null",
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": "title for new context if suggest_new_context is true"
}

Rules:
- Distinguish between tasks (things to do), events (fixed commitments with a specific date/time), and notes (information/knowledge/reference)
- Examples: "dentist at 2pm Thursday" = event; "wash the dishes" = task; "my PT's phone number is 555-1234" = note; "wedding June 15 in Napa" = event with location; "the best pizza place downtown is Mario's" = note
- Notes are information to remember, not actions to take. Auto-suggest 1-3 tags per note.
- If ends_at is not clear, estimate 1 hour from starts_at
- Set is_ambiguous=true for vague dates like "this weekend" or "sometime next week"
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Convert all times to UTC for the ISO 8601 output. For example, if the user says "8am" and their timezone is CST (UTC-6), output "2026-04-01T14:00:00Z"
- Always use ISO 8601 format with Z suffix (UTC) for starts_at and ends_at — never use local timezone offsets or natural language for times
- Set context_confidence to a value between 0.0 and 1.0 reflecting how well this input matches the suggested context
- If no existing context matches well, set suggest_new_context to true and provide a suggested_context_title
- Include interpretations array on action_items only when the item is genuinely ambiguous (could be a pleasantry vs. real task)`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), tzName)
}

// buildTaskExtractionPrompt builds the prompt for task-classified text/voice input.
func buildTaskExtractionPrompt(text string, contextsJSON []byte, now time.Time) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)

	return fmt.Sprintf(`This clause has been classified as a task — something the user needs to do. Extract structured task data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary",
  "action_items": [{"title": "short action-oriented title", "description": "detail or empty string", "priority": "low|medium|high|urgent", "interpretations": ["interpretation1"]}],
  "deadlines": [{"description": "what is due", "date": "YYYY-MM-DD or natural language if ambiguous", "is_ambiguous": false}],
  "events": [],
  "notes": [],
  "ambiguous_references": [{"original_text": "that thing", "reference_type": "pronoun|vague_noun|implicit"}],
  "suggested_context_keywords": ["keyword1", "keyword2"],
  "suggested_context_id": null,
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": ""
}

Rules:
- This IS a task. Do not reclassify it as an event or note.
- Extract a clear, action-oriented title (verb + object): "Call dentist", "Buy groceries", "Finish report"
- Negative examples — these are NOT tasks: "dentist at 2pm Thursday" (event), "Mario's is the best pizza" (note), "the meeting was great" (observation)
- Set priority to "medium" if no signal is present
- Include a deadline only if one is explicitly mentioned
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Convert any times to UTC ISO 8601 with Z suffix
- Include interpretations only when the title is genuinely ambiguous`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), tzName)
}

// buildEventExtractionPrompt builds the prompt for event-classified text/voice input.
func buildEventExtractionPrompt(text string, contextsJSON []byte, now time.Time) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)

	return fmt.Sprintf(`This clause has been classified as an event — a fixed commitment with a specific time or date. Extract structured event data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary",
  "action_items": [],
  "deadlines": [],
  "events": [{"title": "event name", "description": "", "location": "", "starts_at": "2026-04-01T14:00:00Z", "ends_at": "2026-04-01T15:00:00Z", "all_day": false, "is_ambiguous": false}],
  "notes": [],
  "ambiguous_references": [{"original_text": "that thing", "reference_type": "pronoun|vague_noun|implicit"}],
  "suggested_context_keywords": ["keyword1"],
  "suggested_context_id": null,
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": ""
}

Rules:
- This IS an event. Do not reclassify it as a task or note.
- Positive examples: "dentist at 2pm Thursday", "wedding June 15 in Napa", "team standup tomorrow at 10"
- Negative examples — these are NOT events: "call the dentist" (task), "the meeting was great" (past observation, skip)
- starts_at is required. If only a date is given, use all_day=true
- If ends_at is not mentioned, estimate 1 hour from starts_at
- Set is_ambiguous=true for vague times like "this weekend" or "sometime next week"
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Convert all times to UTC ISO 8601 with Z suffix — never use local offsets`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), tzName)
}

// buildNoteExtractionPrompt builds the prompt for note-classified text/voice input.
func buildNoteExtractionPrompt(text string, contextsJSON []byte, now time.Time) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)

	return fmt.Sprintf(`This clause has been classified as a note — reference information with no implied action. Extract structured note data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s

Return JSON with this exact schema:
{
  "summary": "1-2 sentence summary",
  "action_items": [],
  "deadlines": [],
  "events": [],
  "notes": [{"content": "the information captured verbatim or lightly cleaned", "suggested_tags": ["tag1", "tag2"]}],
  "ambiguous_references": [{"original_text": "that thing", "reference_type": "pronoun|vague_noun|implicit"}],
  "suggested_context_keywords": ["keyword1"],
  "suggested_context_id": null,
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": ""
}

Rules:
- This IS a note — reference info, not an action.
- Positive examples: "my PT's phone number is 555-1234", "the best pizza downtown is Mario's", "John's birthday is March 12"
- Negative examples — these are NOT notes: "call the dentist" (task), "dentist at 2pm Thursday" (event)
- Preserve the user's own words in content; clean up only filler words
- Suggest 1-3 tags that would help retrieve this note later
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Use UTC ISO 8601 with Z suffix for any dates`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), tzName)
}

// BuildTextExtractionPrompt builds the prompt for text/voice AI extraction.
// Dispatches to type-specific prompts based on typeHint, or falls back to generic.
// Shared by all extractor implementations.
func BuildTextExtractionPrompt(text string, contextsJSON []byte, now time.Time, typeHint string) string {
	switch typeHint {
	case "task":
		return buildTaskExtractionPrompt(text, contextsJSON, now)
	case "event":
		return buildEventExtractionPrompt(text, contextsJSON, now)
	case "note":
		return buildNoteExtractionPrompt(text, contextsJSON, now)
	default:
		return buildGenericTextExtractionPrompt(text, contextsJSON, now)
	}
}
