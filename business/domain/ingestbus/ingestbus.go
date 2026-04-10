package ingestbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/classify"
	"github.com/casebrophy/planner/business/domain/ingestbus/cleanup"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sanitize"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/taskenergy"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

// IngestResult holds the IDs of created tasks, events, and notes from ingestion.
type IngestResult struct {
	TaskIDs  []uuid.UUID
	EventIDs []uuid.UUID
	NoteIDs  []uuid.UUID
}

// PipelineResult tracks per-step pipeline outcomes for observability.
type PipelineResult struct {
	Sanitize     *StepResult `json:"sanitize,omitempty"`
	Extraction   *StepResult `json:"extraction,omitempty"`
	ContextMatch *StepResult `json:"contextMatch,omitempty"`
	Tasks        *StepResult `json:"tasks,omitempty"`
	Events       *StepResult `json:"events,omitempty"`
	Notes        *StepResult `json:"notes,omitempty"`
}

// StepResult records the outcome of a single pipeline step.
type StepResult struct {
	Status string         `json:"status"` // "completed", "failed", "skipped"
	Detail map[string]any `json:"detail,omitempty"`
}

// Business orchestrates the email ingestion pipeline.
type Business struct {
	log              *logger.Logger
	rawInputBus      *rawinputbus.Business
	emailBus         *emailbus.Business
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	eventBus         *eventbus.Business
	extractor        extractor.Extractor
	noteBus          *notebus.Business
	tagBus           *tagbus.Business
}

// NewBusiness creates a new ingestion pipeline orchestrator.
func NewBusiness(
	log *logger.Logger,
	rawInputBus *rawinputbus.Business,
	emailBus *emailbus.Business,
	taskBus *taskbus.Business,
	contextBus *contextbus.Business,
	clarificationBus *clarificationbus.Business,
	eventBus *eventbus.Business,
	ext extractor.Extractor,
	noteBus *notebus.Business,
	tagBus *tagbus.Business,
) *Business {
	return &Business{
		log:              log,
		rawInputBus:      rawInputBus,
		emailBus:         emailBus,
		taskBus:          taskBus,
		contextBus:       contextBus,
		clarificationBus: clarificationBus,
		eventBus:         eventBus,
		extractor:        ext,
		noteBus:          noteBus,
		tagBus:           tagBus,
	}
}

// ProcessEmail runs the 10-step ingestion pipeline for an email.
func (b *Business) ProcessEmail(ctx context.Context, rawContent string) error {
	// Step 1: Store raw_input
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Email,
		RawContent: rawContent,
	})
	if err != nil {
		return fmt.Errorf("store raw input: %w", err)
	}

	if err := b.processRawInput(ctx, ri, rawContent); err != nil {
		// Step 10 (failure): Mark raw_input failed
		errMsg := err.Error()
		if _, fErr := b.rawInputBus.MarkFailed(ctx, ri, errMsg); fErr != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to mark raw_input failed", "error", fErr)
		}
		return err
	}

	return nil
}

// Reprocess re-runs the pipeline for an existing raw_input.
func (b *Business) Reprocess(ctx context.Context, rawInputID uuid.UUID) error {
	ri, err := b.rawInputBus.QueryByID(ctx, rawInputID)
	if err != nil {
		return fmt.Errorf("query raw input: %w", err)
	}

	ri, err = b.rawInputBus.MarkProcessing(ctx, ri)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	if err := b.processRawInput(ctx, ri, ri.RawContent); err != nil {
		errMsg := err.Error()
		if _, fErr := b.rawInputBus.MarkFailed(ctx, ri, errMsg); fErr != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to mark raw_input failed", "error", fErr)
		}
		return err
	}

	return nil
}

func (b *Business) processRawInput(ctx context.Context, ri rawinputbus.RawInput, rawContent string) error {
	// Mark as processing
	ri, err := b.rawInputBus.MarkProcessing(ctx, ri)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	// Step 2: Parse email
	parsed, err := parseEmail(rawContent)
	if err != nil {
		return fmt.Errorf("parse email: %w", err)
	}

	// Step 3: Dedup check
	if parsed.MessageID != "" {
		_, err := b.emailBus.QueryByMessageID(ctx, parsed.MessageID)
		if err == nil {
			// Already exists, skip
			if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
				return fmt.Errorf("mark processed (dedup): %w", err)
			}
			return nil
		}
		if !errors.Is(err, sqldb.ErrDBNotFound) {
			return fmt.Errorf("dedup check: %w", err)
		}
	}

	// Step 4: Store email record
	var msgID *string
	if parsed.MessageID != "" {
		msgID = &parsed.MessageID
	}
	var fromName *string
	if parsed.FromName != "" {
		fromName = &parsed.FromName
	}
	var bodyHTML *string
	if parsed.BodyHTML != "" {
		bodyHTML = &parsed.BodyHTML
	}

	email, err := b.emailBus.Create(ctx, emailbus.NewEmail{
		RawInputID:  ri.ID,
		MessageID:   msgID,
		FromAddress: parsed.FromAddress,
		FromName:    fromName,
		ToAddress:   parsed.ToAddress,
		Subject:     parsed.Subject,
		BodyText:    parsed.BodyText,
		BodyHTML:    bodyHTML,
		ReceivedAt:  time.Now(),
	})
	if err != nil {
		return fmt.Errorf("store email: %w", err)
	}

	// Step 5: Fetch active contexts
	activeStatus := contextbus.Active
	contexts, err := b.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return fmt.Errorf("fetch contexts: %w", err)
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{
			ID:    c.ID.String(),
			Title: c.Title,
		}
	}
	busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
	for i, r := range ctxRefs {
		busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
	}

	// Step 5b: Sanitize before sending to external API
	subjectResult := sanitize.Sanitize(parsed.Subject)
	bodyResult := sanitize.Sanitize(parsed.BodyText)
	if len(subjectResult.Findings) > 0 || len(bodyResult.Findings) > 0 {
		b.log.Info(ctx, "ingest", "msg", "PII redacted before extraction",
			"raw_input_id", ri.ID,
			"subject_findings", subjectResult.Findings,
			"body_findings", bodyResult.Findings,
		)
	}

	// Step 6: AI extraction
	extraction, err := b.extractor.ExtractEmail(ctx, subjectResult.Text, bodyResult.Text, parsed.FromAddress, ctxRefs)
	if err != nil {
		b.log.Error(ctx, "ingest", "msg", "ai extraction failed, continuing without", "error", err)
		// Don't fail the pipeline on extraction error; just skip AI features
		if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
			return fmt.Errorf("mark processed: %w", err)
		}
		return nil
	}

	// Step 7: Context matching
	var matchedContextID *uuid.UUID
	if extraction.SuggestedContextID != nil && *extraction.SuggestedContextID != "" {
		id, err := uuid.Parse(*extraction.SuggestedContextID)
		if err == nil {
			// Verify context exists
			if _, err := b.contextBus.QueryByID(ctx, id); err == nil {
				matchedContextID = &id
			}
		}
	}

	// Fallback: keyword fuzzy match
	if matchedContextID == nil && len(extraction.SuggestedContextKeywords) > 0 {
		matchedContextID = b.matchContextByKeywords(contexts, extraction.SuggestedContextKeywords)
	}

	// Auto-create context if AI suggests it and no match found
	if matchedContextID == nil && extraction.SuggestNewContext && extraction.SuggestedContextTitle != "" {
		newCtx, err := b.contextBus.Create(ctx, contextbus.NewContext{
			Title:       extraction.SuggestedContextTitle,
			Description: fmt.Sprintf("Auto-created from email: %s", parsed.Subject),
		})
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to auto-create context", "error", err)
		} else {
			id := newCtx.ID
			matchedContextID = &id

			// Generate new_context confirmation clarification
			optionsJSON, _ := json.Marshal(clarificationbus.NewContextOptions{
				ContextID: newCtx.ID.String(),
				Title:     newCtx.Title,
			})
			guess, _ := json.Marshal(map[string]string{
				"title": newCtx.Title,
			})
			guessRaw := json.RawMessage(guess)
			reasoning := fmt.Sprintf("Auto-created context '%s' from email (subject: %s, from: %s). No existing context matched.", newCtx.Title, parsed.Subject, parsed.FromAddress)

			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.NewContext,
				SubjectType:   "context",
				SubjectID:     newCtx.ID,
				Question:      fmt.Sprintf("A new context '%s' was auto-created. Does this look right?", newCtx.Title),
				ClaudeGuess:   &guessRaw,
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create new context clarification", "error", err)
			}
		}
	}

	// Generate clarification for low-confidence context matches
	if matchedContextID != nil && extraction.ContextConfidence > 0 && extraction.ContextConfidence < 0.7 {
		optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
			SuggestedContext:  matchedContextID.String(),
			Confidence:        extraction.ContextConfidence,
			AvailableContexts: busCtxRefs,
		})
		guess, _ := json.Marshal(map[string]string{
			"context_id": matchedContextID.String(),
		})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched with %.0f%% confidence based on keywords: %s", extraction.ContextConfidence*100, strings.Join(extraction.SuggestedContextKeywords, ", "))

		if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   "email",
			SubjectID:     email.ID,
			Question:      fmt.Sprintf("Which context does this email belong to? (Subject: %s)", parsed.Subject),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optionsJSON),
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create context assignment clarification", "error", err)
		}
	}

	// Generate clarification for ambiguous action items
	for _, item := range extraction.ActionItems {
		if len(item.Interpretations) > 1 {
			optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousActionOptions{
				Interpretations: item.Interpretations,
			})
			guess, _ := json.Marshal(map[string]string{
				"title": item.Title,
			})
			guessRaw := json.RawMessage(guess)
			reasoning := fmt.Sprintf("Multiple interpretations found for action item: %s", item.Title)

			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.AmbiguousAction,
				SubjectType:   "email",
				SubjectID:     email.ID,
				Question:      fmt.Sprintf("What does this action item mean? '%s'", item.Title),
				ClaudeGuess:   &guessRaw,
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create ambiguous action clarification", "error", err)
			}
		}
	}

	// Generate clarification for ambiguous deadlines
	for _, dl := range extraction.Deadlines {
		if !dl.IsAmbiguous {
			continue
		}

		optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousDeadlineOptions{
			Description: dl.Description,
			RawDate:     dl.Date,
		})
		guess, _ := json.Marshal(map[string]string{
			"date": dl.Date,
		})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("Deadline '%s' has ambiguous date: %s", dl.Description, dl.Date)

		if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.AmbiguousDeadline,
			SubjectType:   "email",
			SubjectID:     email.ID,
			Question:      fmt.Sprintf("When is '%s' due? (extracted: %s)", dl.Description, dl.Date),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optionsJSON),
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create ambiguous deadline clarification", "error", err)
		}
	}

	// Update email with context if matched
	if matchedContextID != nil {
		email, err = b.emailBus.Update(ctx, email, emailbus.UpdateEmail{ContextID: matchedContextID})
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to update email context", "error", err)
		}
	}

	// Step 8: Create tasks from action items
	for _, item := range extraction.ActionItems {
		priority := taskpriority.Medium
		if item.Priority != "" {
			if p, err := taskpriority.Parse(item.Priority); err == nil {
				priority = p
			}
		}

		nt := taskbus.NewTask{
			Title:       item.Title,
			Description: item.Description,
			Status:      taskstatus.Open,
			Priority:    priority,
			Energy:      taskenergy.Medium,
			ContextID:   matchedContextID,
		}

		if _, err := b.taskBus.Create(ctx, nt); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create task from email", "error", err, "title", item.Title)
		}
	}

	// Step 10: Mark raw_input processed
	if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	return nil
}

// ProcessText runs the ingestion pipeline for raw text input (e.g. voice capture).
func (b *Business) ProcessText(ctx context.Context, rawContent string) (IngestResult, error) {
	// Step 1: Store raw_input
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: rawContent,
	})
	if err != nil {
		return IngestResult{}, fmt.Errorf("store raw input: %w", err)
	}

	result, err := b.processTextInput(ctx, ri, rawContent)
	if err != nil {
		// Mark raw_input failed
		errMsg := err.Error()
		if _, fErr := b.rawInputBus.MarkFailed(ctx, ri, errMsg); fErr != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to mark raw_input failed", "error", fErr)
		}
		return IngestResult{}, err
	}

	return result, nil
}

// EnqueueEmail stores a raw_input record for an email and returns its ID immediately.
// The background worker will process it asynchronously.
func (b *Business) EnqueueEmail(ctx context.Context, rawContent string) (uuid.UUID, error) {
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Email,
		RawContent: rawContent,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("enqueue email: %w", err)
	}
	return ri.ID, nil
}

// EnqueueText stores a raw_input record for a voice/text capture and returns its ID immediately.
// The background worker will process it asynchronously.
func (b *Business) EnqueueText(ctx context.Context, rawContent string) (uuid.UUID, error) {
	ri, err := b.rawInputBus.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: rawContent,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("enqueue text: %w", err)
	}
	return ri.ID, nil
}

// ProcessRawInputByID runs the full ingestion pipeline for an existing raw_input.
// Called by the background worker. On failure it returns the error WITHOUT calling
// MarkFailed — the caller (worker) decides whether to retry or mark terminal.
func (b *Business) ProcessRawInputByID(ctx context.Context, id uuid.UUID) error {
	ri, err := b.rawInputBus.QueryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("query raw input: %w", err)
	}

	ri, err = b.rawInputBus.MarkProcessing(ctx, ri)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	switch ri.SourceType {
	case rawinputsource.Email:
		return b.processRawInput(ctx, ri, ri.RawContent)
	case rawinputsource.Voice:
		_, err := b.processTextInput(ctx, ri, ri.RawContent)
		return err
	default:
		return fmt.Errorf("unknown source type: %s", ri.SourceType)
	}
}

func (b *Business) processTextInput(ctx context.Context, ri rawinputbus.RawInput, rawContent string) (IngestResult, error) {
	// Step 3: Fetch active contexts
	activeStatus := contextbus.Active
	contexts, err := b.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return IngestResult{}, fmt.Errorf("fetch contexts: %w", err)
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{
			ID:    c.ID.String(),
			Title: c.Title,
		}
	}
	busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
	for i, r := range ctxRefs {
		busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
	}

	// Step 4: Sanitize text before sending to external API
	sanitizeResult := sanitize.Sanitize(rawContent)
	if len(sanitizeResult.Findings) > 0 {
		b.log.Info(ctx, "ingest", "msg", "PII redacted before extraction",
			"raw_input_id", ri.ID,
			"findings", sanitizeResult.Findings,
		)
	}

	pr := PipelineResult{}
	pr.Sanitize = &StepResult{
		Status: "completed",
		Detail: map[string]any{"pii_findings": len(sanitizeResult.Findings)},
	}

	// Step 5: Cleanup — strip fillers and split into clauses
	stripped := cleanup.StripFillers(sanitizeResult.Text)
	clauses := cleanup.SplitClauses(stripped)
	if len(clauses) == 0 {
		clauses = []string{sanitizeResult.Text}
	}

	// Step 6: Per-clause classify + extract
	type clauseResult struct {
		clause     string
		cl         classify.Classification
		extraction extractor.TextExtraction
	}

	var clauseResults []clauseResult
	var allKeywords []string
	var bestContextID *string
	var bestContextConf float64
	var suggestNewContext bool
	var suggestedContextTitle string

	for _, clause := range clauses {
		cl := classify.Classify(clause)
		extraction, err := b.extractor.ExtractText(ctx, clause, ctxRefs, string(cl.Type))
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "ai extraction failed for clause, skipping",
				"error", err, "clause", clause)
			continue
		}
		clauseResults = append(clauseResults, clauseResult{
			clause:     clause,
			cl:         cl,
			extraction: extraction,
		})
		allKeywords = append(allKeywords, extraction.SuggestedContextKeywords...)
		if extraction.SuggestedContextID != nil && *extraction.SuggestedContextID != "" &&
			extraction.ContextConfidence > bestContextConf {
			bestContextID = extraction.SuggestedContextID
			bestContextConf = extraction.ContextConfidence
		}
		if extraction.SuggestNewContext && extraction.SuggestedContextTitle != "" {
			suggestNewContext = true
			suggestedContextTitle = extraction.SuggestedContextTitle
		}
	}

	if len(clauseResults) == 0 {
		b.log.Error(ctx, "ingest", "msg", "all clause extractions failed, continuing without")
		pr.Extraction = &StepResult{
			Status: "failed",
			Detail: map[string]any{"error": "all clause extractions failed"},
		}
		resultJSON, _ := json.Marshal(pr)
		raw := json.RawMessage(resultJSON)
		updatedRi, updateErr := b.rawInputBus.Update(ctx, ri, rawinputbus.UpdateRawInput{Result: &raw})
		if updateErr != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to save pipeline result", "error", updateErr)
		} else {
			ri = updatedRi
		}
		if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
			return IngestResult{}, fmt.Errorf("mark processed: %w", err)
		}
		return IngestResult{}, nil
	}

	var totalActionItems, totalEvents, totalNotes int
	for _, cr := range clauseResults {
		totalActionItems += len(cr.extraction.ActionItems)
		totalEvents += len(cr.extraction.Events)
		totalNotes += len(cr.extraction.Notes)
	}

	b.log.Info(ctx, "ingest", "msg", "text extraction result",
		"raw_input_id", ri.ID,
		"clauses", len(clauses),
		"action_items_count", totalActionItems,
		"events_count", totalEvents,
	)

	pr.Extraction = &StepResult{
		Status: "completed",
		Detail: map[string]any{
			"clauses":      len(clauses),
			"action_items": totalActionItems,
			"events":       totalEvents,
			"notes":        totalNotes,
		},
	}

	// Step 7: Context matching (using merged keywords and best context ID from all clauses)
	var matchedContextID *uuid.UUID
	if bestContextID != nil && *bestContextID != "" {
		id, err := uuid.Parse(*bestContextID)
		if err == nil {
			if _, err := b.contextBus.QueryByID(ctx, id); err == nil {
				matchedContextID = &id
			}
		}
	}

	// Fallback: keyword fuzzy match across all clause keywords
	if matchedContextID == nil && len(allKeywords) > 0 {
		matchedContextID = b.matchContextByKeywords(contexts, allKeywords)
	}

	// Auto-create context if any clause suggests it and no match found
	if matchedContextID == nil && suggestNewContext && suggestedContextTitle != "" {
		newCtx, err := b.contextBus.Create(ctx, contextbus.NewContext{
			Title:       suggestedContextTitle,
			Description: "Auto-created from voice input",
		})
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to auto-create context", "error", err)
		} else {
			id := newCtx.ID
			matchedContextID = &id

			optionsJSON, _ := json.Marshal(clarificationbus.NewContextOptions{
				ContextID: newCtx.ID.String(),
				Title:     newCtx.Title,
			})
			guess, _ := json.Marshal(map[string]string{
				"title": newCtx.Title,
			})
			guessRaw := json.RawMessage(guess)
			reasoning := fmt.Sprintf("Auto-created context '%s' from voice input. No existing context matched.", newCtx.Title)

			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.NewContext,
				SubjectType:   "context",
				SubjectID:     newCtx.ID,
				Question:      fmt.Sprintf("A new context '%s' was auto-created. Does this look right?", newCtx.Title),
				ClaudeGuess:   &guessRaw,
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create new context clarification", "error", err)
			}
		}
	}

	if matchedContextID != nil {
		pr.ContextMatch = &StepResult{
			Status: "completed",
			Detail: map[string]any{"context_id": matchedContextID.String()},
		}
	} else {
		pr.ContextMatch = &StepResult{Status: "skipped"}
	}

	// Generate clarification for low-confidence context matches
	if matchedContextID != nil && bestContextConf > 0 && bestContextConf < 0.7 {
		optionsJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
			SuggestedContext:  matchedContextID.String(),
			Confidence:        bestContextConf,
			AvailableContexts: busCtxRefs,
		})
		guess, _ := json.Marshal(map[string]string{
			"context_id": matchedContextID.String(),
		})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched with %.0f%% confidence based on keywords: %s", bestContextConf*100, strings.Join(allKeywords, ", "))

		if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   "raw_input",
			SubjectID:     ri.ID,
			Question:      fmt.Sprintf("Which context does this belong to? (%d clauses processed)", len(clauses)),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optionsJSON),
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create context assignment clarification", "error", err)
		}
	}

	// Step 8: Create tasks, events, notes per clause; collect IDs and all items for clarifications
	var createdTaskIDs []uuid.UUID
	var createdEventIDs []uuid.UUID
	var createdNoteIDs []uuid.UUID
	var allActionItems []extractor.ActionItem
	var allDeadlines []extractor.Deadline

	for _, cr := range clauseResults {
		unconfirmed := cr.cl.Confidence < 0.75

		// Create TypeAssignment clarification for low-confidence clauses
		if unconfirmed {
			optionsJSON, _ := json.Marshal(clarificationbus.TypeAssignmentOptions{
				ClauseText:    cr.clause,
				PredictedType: string(cr.cl.Type),
				Confidence:    cr.cl.Confidence,
				Options:       []string{"task", "note", "event"},
			})
			reasoning := fmt.Sprintf("Classified as %s with %.0f%% confidence — ambiguous signal", cr.cl.Type, cr.cl.Confidence*100)
			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.TypeAssignment,
				SubjectType:   "raw_input",
				SubjectID:     ri.ID,
				Question:      fmt.Sprintf("What type is this? '%s'", cr.clause),
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create type assignment clarification", "error", err)
			}
		}

		// Create tasks from this clause's action items
		for _, item := range cr.extraction.ActionItems {
			priority := taskpriority.Medium
			if item.Priority != "" {
				if p, err := taskpriority.Parse(item.Priority); err == nil {
					priority = p
				}
			}

			nt := taskbus.NewTask{
				Title:       item.Title,
				Description: item.Description,
				Status:      taskstatus.Open,
				Priority:    priority,
				Energy:      taskenergy.Medium,
				ContextID:   matchedContextID,
				Unconfirmed: unconfirmed,
			}

			task, err := b.taskBus.Create(ctx, nt)
			if err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create task from text", "error", err, "title", item.Title)
			} else {
				createdTaskIDs = append(createdTaskIDs, task.ID)
			}
		}

		// Create events from this clause's events
		for _, ev := range cr.extraction.Events {
			startsAt, err := time.Parse(time.RFC3339, ev.StartsAt)
			if err != nil {
				startsAt, err = time.Parse("2006-01-02", ev.StartsAt)
				if err != nil {
					b.log.Error(ctx, "ingest", "msg", "failed to parse event start time", "error", err, "starts_at", ev.StartsAt)
					continue
				}
			}

			endsAt := startsAt.Add(time.Hour)
			if ev.EndsAt != "" {
				if parsed, err := time.Parse(time.RFC3339, ev.EndsAt); err == nil {
					endsAt = parsed
				} else if parsed, err := time.Parse("2006-01-02", ev.EndsAt); err == nil {
					endsAt = parsed
				}
			}

			var location *string
			if ev.Location != "" {
				location = &ev.Location
			}

			event, err := b.eventBus.Create(ctx, eventbus.NewEvent{
				Title:       ev.Title,
				Description: ev.Description,
				Location:    location,
				StartsAt:    startsAt,
				EndsAt:      endsAt,
				AllDay:      ev.AllDay,
				RawInputID:  &ri.ID,
				ContextID:   matchedContextID,
				Unconfirmed: unconfirmed || ev.IsAmbiguous,
			})
			if err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create event", "error", err, "title", ev.Title)
				continue
			}
			createdEventIDs = append(createdEventIDs, event.ID)
		}

		// Create notes from this clause's notes
		for _, n := range cr.extraction.Notes {
			nn := notebus.NewNote{
				Content:     n.Content,
				Source:      "voice",
				RawInputID:  &ri.ID,
				ContextID:   matchedContextID,
				Unconfirmed: unconfirmed,
			}

			note, err := b.noteBus.Create(ctx, nn)
			if err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create note", "error", err)
				continue
			}
			createdNoteIDs = append(createdNoteIDs, note.ID)

			for _, tagName := range n.SuggestedTags {
				tags, _ := b.tagBus.Query(ctx, tagbus.QueryFilter{Name: &tagName}, tagbus.DefaultOrderBy, page.New(1, 1))
				var tagID uuid.UUID
				if len(tags) > 0 {
					tagID = tags[0].ID
				} else {
					newTag, err := b.tagBus.Create(ctx, tagbus.NewTag{Name: tagName})
					if err != nil {
						continue
					}
					tagID = newTag.ID
				}
				_ = b.tagBus.AddToNote(ctx, note.ID, tagID)
			}
		}

		allActionItems = append(allActionItems, cr.extraction.ActionItems...)
		allDeadlines = append(allDeadlines, cr.extraction.Deadlines...)
	}

	taskIDStrs := make([]string, len(createdTaskIDs))
	for i, id := range createdTaskIDs {
		taskIDStrs[i] = id.String()
	}
	pr.Tasks = &StepResult{
		Status: "completed",
		Detail: map[string]any{"count": len(createdTaskIDs), "ids": taskIDStrs},
	}

	eventIDStrs := make([]string, len(createdEventIDs))
	for i, id := range createdEventIDs {
		eventIDStrs[i] = id.String()
	}
	pr.Events = &StepResult{
		Status: "completed",
		Detail: map[string]any{"count": len(createdEventIDs), "ids": eventIDStrs},
	}

	noteIDStrs := make([]string, len(createdNoteIDs))
	for i, id := range createdNoteIDs {
		noteIDStrs[i] = id.String()
	}
	pr.Notes = &StepResult{
		Status: "completed",
		Detail: map[string]any{"count": len(createdNoteIDs), "ids": noteIDStrs},
	}

	// Step 9: Generate clarifications for ambiguous action items and deadlines
	for _, item := range allActionItems {
		if len(item.Interpretations) > 1 {
			optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousActionOptions{
				Interpretations: item.Interpretations,
			})
			guess, _ := json.Marshal(map[string]string{
				"title": item.Title,
			})
			guessRaw := json.RawMessage(guess)
			reasoning := fmt.Sprintf("Multiple interpretations found for action item: %s", item.Title)

			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.AmbiguousAction,
				SubjectType:   "raw_input",
				SubjectID:     ri.ID,
				Question:      fmt.Sprintf("What does this action item mean? '%s'", item.Title),
				ClaudeGuess:   &guessRaw,
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create ambiguous action clarification", "error", err)
			}
		}
	}

	for _, dl := range allDeadlines {
		if !dl.IsAmbiguous {
			continue
		}

		optionsJSON, _ := json.Marshal(clarificationbus.AmbiguousDeadlineOptions{
			Description: dl.Description,
			RawDate:     dl.Date,
		})
		guess, _ := json.Marshal(map[string]string{
			"date": dl.Date,
		})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("Deadline '%s' has ambiguous date: %s", dl.Description, dl.Date)

		if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.AmbiguousDeadline,
			SubjectType:   "raw_input",
			SubjectID:     ri.ID,
			Question:      fmt.Sprintf("When is '%s' due? (extracted: %s)", dl.Description, dl.Date),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optionsJSON),
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create ambiguous deadline clarification", "error", err)
		}
	}

	// Generate voice_reference clarifications for ambiguous references
	for _, cr := range clauseResults {
		for _, ref := range cr.extraction.AmbiguousReferences {
			optionsJSON, _ := json.Marshal(clarificationbus.VoiceReferenceOptions{
				OriginalText:  ref.OriginalText,
				ReferenceType: ref.ReferenceType,
				ClauseText:    cr.clause,
			})
			reasoning := fmt.Sprintf("Voice input contains ambiguous %s reference: %q", ref.ReferenceType, ref.OriginalText)

			if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
				Kind:          clarificationkind.VoiceReference,
				SubjectType:   "raw_input",
				SubjectID:     ri.ID,
				Question:      fmt.Sprintf("What does '%s' refer to?", ref.OriginalText),
				Reasoning:     &reasoning,
				AnswerOptions: json.RawMessage(optionsJSON),
			}); err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to create voice reference clarification", "error", err)
			}
		}
	}

	// Save pipeline result before marking processed
	resultJSON, _ := json.Marshal(pr)
	raw := json.RawMessage(resultJSON)
	updatedRi, updateErr := b.rawInputBus.Update(ctx, ri, rawinputbus.UpdateRawInput{Result: &raw})
	if updateErr != nil {
		b.log.Error(ctx, "ingest", "msg", "failed to save pipeline result", "error", updateErr)
	} else {
		ri = updatedRi
	}

	// Step 10: Mark raw_input processed
	if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
		return IngestResult{}, fmt.Errorf("mark processed: %w", err)
	}

	return IngestResult{
		TaskIDs:  createdTaskIDs,
		EventIDs: createdEventIDs,
		NoteIDs:  createdNoteIDs,
	}, nil
}

// matchContextByKeywords attempts to find a matching context by looking for keywords in context titles.
func (b *Business) matchContextByKeywords(contexts []contextbus.Context, keywords []string) *uuid.UUID {
	for _, c := range contexts {
		title := strings.ToLower(c.Title)
		for _, kw := range keywords {
			if strings.Contains(title, strings.ToLower(kw)) {
				id := c.ID
				return &id
			}
		}
	}
	return nil
}
