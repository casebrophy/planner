package extractor

import "fmt"

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
