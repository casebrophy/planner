package threadbus

import (
	"context"
	"fmt"

	"github.com/casebrophy/planner/foundation/claudecli"
)

// threadExtractionSchema is the JSON schema for structured output validation.
const threadExtractionSchema = `{
  "type": "object",
  "properties": {
    "kind": {"type": "string"},
    "sentiment": {"type": ["string", "null"]},
    "blocking_party": {"type": ["string", "null"]},
    "timeline_delta_days": {"type": ["integer", "null"]},
    "requires_action": {"type": "boolean"},
    "confidence": {"type": "number"}
  },
  "required": ["kind", "requires_action", "confidence"]
}`

// ClaudeCodeExtractor implements Extractor using the Claude Code CLI.
type ClaudeCodeExtractor struct {
	client *claudecli.Client
}

// NewClaudeCodeExtractor creates a thread extractor backed by the Claude CLI.
func NewClaudeCodeExtractor(client *claudecli.Client) *ClaudeCodeExtractor {
	return &ClaudeCodeExtractor{client: client}
}

// ExtractThreadEntry classifies a thread entry using Claude CLI inference.
func (e *ClaudeCodeExtractor) ExtractThreadEntry(ctx context.Context, content string, subjectType string) (ExtractionResult, error) {
	prompt := fmt.Sprintf(`Classify this thread entry and extract structured data. Return ONLY valid JSON.

Thread entry content:
%s

Subject type: %s

Return JSON with this schema:
{
  "kind": "update|blocker|decision|question|resolution|note",
  "sentiment": "positive|neutral|negative|null",
  "blocking_party": "who is blocking or null",
  "timeline_delta_days": "integer days of timeline shift or null",
  "requires_action": true/false,
  "confidence": 0.0-1.0
}

Rules:
- kind should be one of: update, blocker, decision, question, resolution, note
- Set confidence to reflect how certain you are about the classification
- Only set blocking_party when kind is "blocker"
- timeline_delta_days is positive for delays, negative for ahead of schedule`, content, subjectType)

	var result ExtractionResult
	shouldEscalate := func() bool {
		return result.Confidence < 0.4
	}

	if err := e.client.RunJSON(ctx, prompt, threadExtractionSchema, &result, shouldEscalate); err != nil {
		return ExtractionResult{}, err
	}

	return result, nil
}
