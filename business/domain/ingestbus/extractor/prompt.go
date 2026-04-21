package extractor

import (
	"fmt"
	"strings"
	"time"
)

// correctionPreamble returns a preamble string when userCorrection is non-empty,
// to be prepended to extraction prompts. Returns empty string if userCorrection is empty.
func correctionPreamble(userCorrection string) string {
	if userCorrection == "" {
		return ""
	}
	return fmt.Sprintf("IMPORTANT — The user has provided a correction for this input: '%s'. Treat this as the authoritative interpretation. The original content may contain transcription errors.\n\n", userCorrection)
}

// BuildCandidateBlock formats candidate entities for injection into extraction prompts.
func BuildCandidateBlock(candidates []EntityMatch) string {
	if len(candidates) == 0 {
		return ""
	}
	block := "\n## Existing Entities (potential matches)\nThe following existing entities were found to be semantically similar to this input.\nIf the input is clearly referring to or updating one of these, set entity_resolutions\nwith action \"update\" and the matched_id. If ambiguous between multiple, use \"ambiguous\".\nIf this is genuinely new content, use \"create\".\n\n"
	for i, cand := range candidates {
		block += fmt.Sprintf("%d. [%s] id=%s title=\"%s\" content=\"%s\" (similarity: %.2f)\n", i+1, strings.ToUpper(cand.SourceType), cand.ID, cand.Title, cand.Content, cand.Similarity)
	}
	return block
}

// BuildContextAnnotationsBlock formats context annotations for injection into extraction prompts.
func BuildContextAnnotationsBlock(annotations []string) string {
	if len(annotations) == 0 {
		return ""
	}
	block := "\n## Context Annotations\nThe following annotations describe relevant context for this input:\n"
	for _, annotation := range annotations {
		block += fmt.Sprintf("- %s\n", annotation)
	}
	return block
}

// BuildEmailExtractionPrompt builds the prompt for email AI extraction.
// Shared by all extractor implementations.
func BuildEmailExtractionPrompt(fromAddress, subject, bodyText, userCorrection string, contextsJSON []byte, candidates []EntityMatch, contextAnnotations []string) string {
	candidateBlock := BuildCandidateBlock(candidates)
	contextAnnotationsBlock := BuildContextAnnotationsBlock(contextAnnotations)
	return correctionPreamble(userCorrection) + fmt.Sprintf(`Analyze this email and extract structured data. Return ONLY valid JSON with no other text.

Email:
From: %s
Subject: %s
Body:
%s

Active contexts (match this email to one if relevant):
%s%s%s

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
  "suggested_context_title": "title for new context if suggest_new_context is true",
  "entity_resolutions": [{"action": "update|create|ambiguous", "matched_id": "UUID if action is update", "matched_type": "event|task|note", "confidence": 0.0, "reasoning": "why this decision"}]
}

Rules:
- Set context_confidence to a value between 0.0 and 1.0 reflecting how well this email matches the suggested context
- If no existing context matches well, set suggest_new_context to true and provide a suggested_context_title
- Set is_ambiguous on deadlines when the date is relative or vague (e.g. "end of month", "soon", "next week")
- Include interpretations array on action_items only when the item is genuinely ambiguous (could be a pleasantry vs. real task)`, fromAddress, subject, bodyText, string(contextsJSON), candidateBlock, contextAnnotationsBlock)
}

// buildGenericTextExtractionPrompt builds the fallback prompt for text/voice AI extraction.
// Used when typeHint is empty or unrecognized.
func buildGenericTextExtractionPrompt(text, userCorrection string, contextsJSON []byte, now time.Time, candidates []EntityMatch, contextAnnotations []string) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)
	candidateBlock := BuildCandidateBlock(candidates)
	contextAnnotationsBlock := BuildContextAnnotationsBlock(contextAnnotations)

	return correctionPreamble(userCorrection) + fmt.Sprintf(`This is a voice capture from the user. Extract tasks, events, deadlines, and context information. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Voice capture:
%s

Active contexts (match this input to one if relevant):
%s%s%s

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
  "suggested_context_title": "title for new context if suggest_new_context is true",
  "entity_resolutions": [{"action": "update|create|ambiguous", "matched_id": "UUID if action is update", "matched_type": "event|task|note", "confidence": 0.0, "reasoning": "why this decision"}]
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
- Include interpretations array on action_items only when the item is genuinely ambiguous (could be a pleasantry vs. real task)
- If the input contains a list with 3+ items, expand each item as a separate actionable item rather than lumping them into a single multi-line entry`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), candidateBlock, contextAnnotationsBlock, tzName)
}

// buildTaskExtractionPrompt builds the prompt for task-classified text/voice input.
func buildTaskExtractionPrompt(text, userCorrection string, contextsJSON []byte, now time.Time, candidates []EntityMatch, contextAnnotations []string) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)
	candidateBlock := BuildCandidateBlock(candidates)
	contextAnnotationsBlock := BuildContextAnnotationsBlock(contextAnnotations)

	return correctionPreamble(userCorrection) + fmt.Sprintf(`This clause has been classified as a task — something the user needs to do. Extract structured task data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s%s%s

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
  "suggested_context_title": "",
  "entity_resolutions": [{"action": "update|create|ambiguous", "matched_id": "UUID if action is update", "matched_type": "event|task|note", "confidence": 0.0, "reasoning": "why this decision"}]
}

Rules:
- This IS a task. Do not reclassify it as an event or note.
- Extract a clear, action-oriented title (verb + object): "Call dentist", "Buy groceries", "Finish report"
- Negative examples — these are NOT tasks: "dentist at 2pm Thursday" (event), "Mario's is the best pizza" (note), "the meeting was great" (observation)
- Set priority to "medium" if no signal is present
- Include a deadline only if one is explicitly mentioned
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Convert any times to UTC ISO 8601 with Z suffix
- Include interpretations only when the title is genuinely ambiguous
- If the input contains a list with 3+ items, expand each item as a separate actionable item rather than lumping them into a single multi-line entry`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), candidateBlock, contextAnnotationsBlock, tzName)
}

// buildEventExtractionPrompt builds the prompt for event-classified text/voice input.
func buildEventExtractionPrompt(text, userCorrection string, contextsJSON []byte, now time.Time, candidates []EntityMatch, contextAnnotations []string) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)
	candidateBlock := BuildCandidateBlock(candidates)
	contextAnnotationsBlock := BuildContextAnnotationsBlock(contextAnnotations)

	return correctionPreamble(userCorrection) + fmt.Sprintf(`This clause has been classified as an event — a fixed commitment with a specific time or date. Extract structured event data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s%s%s

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
  "suggested_context_title": "",
  "entity_resolutions": [{"action": "update|create|ambiguous", "matched_id": "UUID if action is update", "matched_type": "event|task|note", "confidence": 0.0, "reasoning": "why this decision"}]
}

Rules:
- This IS an event. Do not reclassify it as a task or note.
- Positive examples: "dentist at 2pm Thursday", "wedding June 15 in Napa", "team standup tomorrow at 10"
- Negative examples — these are NOT events: "call the dentist" (task), "the meeting was great" (past observation, skip)
- starts_at is required. If only a date is given, use all_day=true
- If ends_at is not mentioned, estimate 1 hour from starts_at
- Set is_ambiguous=true for vague times like "this weekend" or "sometime next week"
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Convert all times to UTC ISO 8601 with Z suffix — never use local offsets
- If the input contains a list with 3+ items, expand each item as a separate actionable item rather than lumping them into a single multi-line entry`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), candidateBlock, contextAnnotationsBlock, tzName)
}

// buildNoteExtractionPrompt builds the prompt for note-classified text/voice input.
func buildNoteExtractionPrompt(text, userCorrection string, contextsJSON []byte, now time.Time, candidates []EntityMatch, contextAnnotations []string) string {
	tzName, tzOffset := now.Zone()
	currentTime := now.Format(time.RFC3339)
	candidateBlock := BuildCandidateBlock(candidates)
	contextAnnotationsBlock := BuildContextAnnotationsBlock(contextAnnotations)

	return correctionPreamble(userCorrection) + fmt.Sprintf(`This clause has been classified as a note — reference information with no implied action. Extract structured note data. Return ONLY valid JSON with no other text.

Current time: %s
User timezone: %s (UTC%+d)

Clause:
%s

Active contexts (match this input to one if relevant):
%s%s%s

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
  "suggested_context_title": "",
  "entity_resolutions": [{"action": "update|create|ambiguous", "matched_id": "UUID if action is update", "matched_type": "event|task|note", "confidence": 0.0, "reasoning": "why this decision"}]
}

Rules:
- This IS a note — reference info, not an action.
- Positive examples: "my PT's phone number is 555-1234", "the best pizza downtown is Mario's", "John's birthday is March 12"
- Negative examples — these are NOT notes: "call the dentist" (task), "dentist at 2pm Thursday" (event)
- Preserve the user's own words in content; clean up only filler words
- Suggest 1-3 tags that would help retrieve this note later
- Flag ambiguous_references when the text contains vague pronouns ("it", "that thing"), unclear nouns ("the project", "the meeting"), or implicit references that can't be resolved from context alone. reference_type should be "pronoun", "vague_noun", or "implicit"
- The user speaks in their local timezone (%s). Use UTC ISO 8601 with Z suffix for any dates
- If the input contains a list with 3+ items, expand each item as a separate actionable item rather than lumping them into a single multi-line entry`, currentTime, tzName, tzOffset/3600, text, string(contextsJSON), candidateBlock, contextAnnotationsBlock, tzName)
}

// buildTransactionExtractionPrompt builds the prompt for transaction enrichment.
// Used by OllamaExtractor when typeHint is "transaction".
func buildTransactionExtractionPrompt(text, userCorrection string, contextsJSON []byte, contextAnnotations []string) string {
	return correctionPreamble(userCorrection) + fmt.Sprintf(`Analyze this bank transaction description and extract structured data. Return ONLY valid JSON with no other text.

Transaction description:
%s

Active contexts (suggest one if this transaction is relevant):
%s

Return JSON with this exact schema:
{
  "summary": "cleaned merchant/payee name (strip card numbers, transaction codes, store numbers)",
  "action_items": [],
  "deadlines": [],
  "events": [],
  "notes": [],
  "suggested_context_keywords": ["spending_category"],
  "suggested_context_id": "UUID of best matching context or null",
  "context_confidence": 0.0,
  "suggest_new_context": false,
  "suggested_context_title": ""
}

Rules:
- summary should be a clean, human-readable merchant name. Examples:
  - "AMZN MKTP US*AB1CD2EF3" → "Amazon"
  - "TST* JOE'S COFFEE #1234" → "Joe's Coffee"
  - "UBER *EATS 8765-4321" → "Uber Eats"
  - "SQ *MARIO'S PIZZA" → "Mario's Pizza"
- suggested_context_keywords should contain exactly ONE spending category from: groceries, dining, transport, utilities, entertainment, shopping, health, travel, subscription, transfer, income, other
- Only set suggested_context_id if one of the active contexts clearly relates to this spending
- Set context_confidence between 0.0 and 1.0`, text, string(contextsJSON))
}

// BuildTextExtractionPrompt builds the prompt for text/voice AI extraction.
// Dispatches to type-specific prompts based on typeHint, or falls back to generic.
// Shared by all extractor implementations.
func BuildTextExtractionPrompt(text, userCorrection string, contextsJSON []byte, now time.Time, typeHint string, candidates []EntityMatch, contextAnnotations []string) string {
	switch typeHint {
	case "task":
		return buildTaskExtractionPrompt(text, userCorrection, contextsJSON, now, candidates, contextAnnotations)
	case "event":
		return buildEventExtractionPrompt(text, userCorrection, contextsJSON, now, candidates, contextAnnotations)
	case "note":
		return buildNoteExtractionPrompt(text, userCorrection, contextsJSON, now, candidates, contextAnnotations)
	case "transaction":
		return buildTransactionExtractionPrompt(text, userCorrection, contextsJSON, contextAnnotations)
	default:
		return buildGenericTextExtractionPrompt(text, userCorrection, contextsJSON, now, candidates, contextAnnotations)
	}
}

// BuildReceiptExtractionPrompt builds the prompt for receipt OCR extraction.
// Shared by all extractor implementations.
func BuildReceiptExtractionPrompt(ocrText string) string {
	return fmt.Sprintf(`Parse the following OCR text from a receipt into structured JSON. The text may contain OCR noise (misspellings, broken lines, extra characters) — do your best to interpret it.

OCR Text:
%s

Rules:
- All monetary amounts must be in cents (multiply dollars by 100, e.g. $12.50 → 1250)
- Date must be in YYYY-MM-DD format; if ambiguous use today's date
- merchant: the store or restaurant name
- total: the final charged amount in cents
- tax: sales tax in cents (0 if not shown)
- subtotal: pre-tax total in cents (total - tax if not explicitly shown)
- items: individual line items with description, amount in cents, and quantity (default 1 if not shown)
- notes: any useful notes (e.g. "tip not included", "cash transaction") or empty string

Return ONLY valid JSON with no other text.`, ocrText)
}

// BuildGapAnalysisPrompt builds the prompt for gap analysis.
// Given a new entity and related entities found via semantic search, ask the AI what's missing.
func BuildGapAnalysisPrompt(entityType, entityContent string, relatedEntities []RelatedEntity) string {
	relatedBlock := ""
	if len(relatedEntities) > 0 {
		relatedBlock = "\n## Related Entities Already in System\n"
		for i, e := range relatedEntities {
			relatedBlock += fmt.Sprintf("%d. [%s] id=%s title=%q\n   %s\n", i+1, strings.ToUpper(e.SourceType), e.ID, e.Title, e.Content)
		}
	}
	return fmt.Sprintf(`You are a personal knowledge assistant. A new %s was just created. Based on the entity content and related entities already in the system, identify what useful information is currently missing that would help manage this entity more effectively.

Focus on actionable gaps: contact info, locations, deadlines, dependencies, or important context that would be useful to know.

## Scope constraints

- Every gap MUST be about information missing from the NEW entity.
- You MAY reference related entities when they inform a gap (e.g. a potential dependency, stakeholder, or context link).
- Do NOT ask meta-questions or observations about the system itself. Explicitly forbidden:
  * "Should we consolidate these near-identical tasks?" ← NO
  * "Why are there multiple copies of 'Walk Daily'?" ← NO
  * "This looks like a duplicate — should we clean it up?" ← NO
  * "We should consider a recurrence pattern" ← NO
  * "This may be redundant with the existing task" ← NO
  * Any observations about data quality, task hygiene, or process improvement
- Do NOT ask about duplicates, consolidation, recurrence patterns, or cleanup. Related entities are shown as context only — never as the subject of the gap.
- Do NOT emit gaps that are observations about the system, database, or organizational structure.
- ALLOWED: Questions that reference related entities to inform dependencies ("Could Task A depend on Task B?"), stakeholders ("Is Alice responsible for this?"), or context ("Does this connect to the Q2 project?").

## New Entity (%s)
%s
%s
## Gap Categories

1. **missing_contact**: Contact information (phone, email, address, social profiles)
2. **missing_location**: Physical location or venue details
3. **missing_context**: Background context, situational information, or explanation
4. **missing_dependency**: Related items, blockers, or dependencies needed to proceed
5. **missing_detail**: Specific details (dimensions, quantities, specifications)
6. **missing_deadline**: Timeline, deadline, or time-sensitive information
7. **missing_stakeholder**: Key people, roles, or decision-makers involved
8. **missing_outcome**: Goals, success criteria, or desired outcomes

## Options Guidance

For each gap, decide whether the answer space is enumerable (discrete choices) or open-ended:

- **Enumerable questions** (discrete answer set): Provide 2-4 representative options. Set options to the option strings and options_confidence to your confidence (0.0-1.0) that these are the right choices.
  - Example: "Is the project timeline...?" → options: ["flexible", "fixed deadline", "unknown"] → options_confidence: 0.9
  - Example: "Where is the meeting?" → options: ["building A", "building B", "remote", "unknown"] → options_confidence: 0.8

- **Open-ended questions** (unbounded answer): Set options to empty array [] and options_confidence to 0.
  - Example: "What is the project budget?" → options: [] → options_confidence: 0
  - Example: "Who are all the stakeholders?" → options: [] → options_confidence: 0

Return ONLY valid JSON with no other text. Use this structure:
{
  "gaps": [
    {
      "category": "<missing_contact|missing_location|missing_context|missing_dependency|missing_detail|missing_deadline|missing_stakeholder|missing_outcome>",
      "question": "<specific question to ask the user>",
      "reasoning": "<why this information would be useful>",
      "confidence": <0.0-1.0>,
      "options": ["option1", "option2", "option3"],
      "options_confidence": <0.0-1.0>,
      "related_ids": ["<entity_id>", ...]
    }
  ]
}

Only include gaps with confidence >= 0.5. Return an empty gaps array if no significant gaps are found.`, entityType, entityType, entityContent, relatedBlock)
}
