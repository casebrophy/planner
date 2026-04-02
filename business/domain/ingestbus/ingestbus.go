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
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
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

// IngestResult holds the IDs of created tasks and events from ingestion.
type IngestResult struct {
	TaskIDs  []uuid.UUID
	EventIDs []uuid.UUID
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
			optionsJSON, _ := json.Marshal(map[string]any{
				"type":       "new_context",
				"context_id": newCtx.ID.String(),
				"title":      newCtx.Title,
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
		optionsJSON, _ := json.Marshal(map[string]any{
			"type":               "context_assignment",
			"suggested_context":  matchedContextID.String(),
			"confidence":         extraction.ContextConfidence,
			"available_contexts": ctxRefs,
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
			optionsJSON, _ := json.Marshal(map[string]any{
				"type":            "ambiguous_action",
				"interpretations": item.Interpretations,
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

		optionsJSON, _ := json.Marshal(map[string]any{
			"type":        "ambiguous_deadline",
			"description": dl.Description,
			"raw_date":    dl.Date,
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
			Status:      taskstatus.Todo,
			Priority:    priority,
			Energy:      taskenergy.Medium,
			ContextID:   matchedContextID,
		}

		if _, err := b.taskBus.Create(ctx, nt); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create task from email", "error", err, "title", item.Title)
		}
	}

	// Step 9: Create context event
	if matchedContextID != nil {
		metadata := map[string]any{
			"email_id":     email.ID.String(),
			"from_address": parsed.FromAddress,
			"subject":      parsed.Subject,
			"sentiment":    extraction.Sentiment,
		}
		metadataJSON, _ := json.Marshal(metadata)
		raw := json.RawMessage(metadataJSON)

		emailID := email.ID
		if _, err := b.contextBus.AddEvent(ctx, contextbus.NewEvent{
			ContextID: *matchedContextID,
			Kind:      "email",
			Content:   extraction.Summary,
			Metadata:  &raw,
			SourceID:  &emailID,
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create context event", "error", err)
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

func (b *Business) processTextInput(ctx context.Context, ri rawinputbus.RawInput, rawContent string) (IngestResult, error) {
	// Step 2: Mark as processing
	ri, err := b.rawInputBus.MarkProcessing(ctx, ri)
	if err != nil {
		return IngestResult{}, fmt.Errorf("mark processing: %w", err)
	}

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

	// Step 4: Sanitize text before sending to external API
	sanitizeResult := sanitize.Sanitize(rawContent)
	if len(sanitizeResult.Findings) > 0 {
		b.log.Info(ctx, "ingest", "msg", "PII redacted before extraction",
			"raw_input_id", ri.ID,
			"findings", sanitizeResult.Findings,
		)
	}

	// Step 5: AI extraction
	extraction, err := b.extractor.ExtractText(ctx, sanitizeResult.Text, ctxRefs)
	if err != nil {
		b.log.Error(ctx, "ingest", "msg", "ai extraction failed, continuing without", "error", err)
		// Don't fail the pipeline on extraction error; just skip AI features
		if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
			return IngestResult{}, fmt.Errorf("mark processed: %w", err)
		}
		return IngestResult{}, nil
	}

	// Step 6: Context matching
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
			Description: "Auto-created from voice input",
		})
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to auto-create context", "error", err)
		} else {
			id := newCtx.ID
			matchedContextID = &id

			// Generate new_context confirmation clarification
			optionsJSON, _ := json.Marshal(map[string]any{
				"type":       "new_context",
				"context_id": newCtx.ID.String(),
				"title":      newCtx.Title,
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

	// Generate clarification for low-confidence context matches
	if matchedContextID != nil && extraction.ContextConfidence > 0 && extraction.ContextConfidence < 0.7 {
		optionsJSON, _ := json.Marshal(map[string]any{
			"type":               "context_assignment",
			"suggested_context":  matchedContextID.String(),
			"confidence":         extraction.ContextConfidence,
			"available_contexts": ctxRefs,
		})
		guess, _ := json.Marshal(map[string]string{
			"context_id": matchedContextID.String(),
		})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched with %.0f%% confidence based on keywords: %s", extraction.ContextConfidence*100, strings.Join(extraction.SuggestedContextKeywords, ", "))

		if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   "raw_input",
			SubjectID:     ri.ID,
			Question:      fmt.Sprintf("Which context does this belong to?"),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optionsJSON),
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create context assignment clarification", "error", err)
		}
	}

	// Step 7: Create tasks from action items and collect IDs
	var createdTaskIDs []uuid.UUID
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
			Status:      taskstatus.Todo,
			Priority:    priority,
			Energy:      taskenergy.Medium,
			ContextID:   matchedContextID,
		}

		task, err := b.taskBus.Create(ctx, nt)
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create task from text", "error", err, "title", item.Title)
		} else {
			createdTaskIDs = append(createdTaskIDs, task.ID)
		}
	}

	// Create events
	var createdEventIDs []uuid.UUID
	for _, ev := range extraction.Events {
		startsAt, err := time.Parse(time.RFC3339, ev.StartsAt)
		if err != nil {
			// Try other common formats
			startsAt, err = time.Parse("2006-01-02", ev.StartsAt)
			if err != nil {
				b.log.Error(ctx, "ingest", "msg", "failed to parse event start time", "error", err, "starts_at", ev.StartsAt)
				continue
			}
		}

		endsAt := startsAt.Add(time.Hour) // default 1 hour
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
		})
		if err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create event", "error", err, "title", ev.Title)
			continue
		}
		createdEventIDs = append(createdEventIDs, event.ID)
	}

	// Step 8: Create context event
	if matchedContextID != nil {
		metadata := map[string]any{
			"raw_input_id": ri.ID.String(),
			"text":         rawContent,
		}
		metadataJSON, _ := json.Marshal(metadata)
		raw := json.RawMessage(metadataJSON)

		if _, err := b.contextBus.AddEvent(ctx, contextbus.NewEvent{
			ContextID: *matchedContextID,
			Kind:      "voice",
			Content:   extraction.Summary,
			Metadata:  &raw,
			SourceID:  &ri.ID,
		}); err != nil {
			b.log.Error(ctx, "ingest", "msg", "failed to create context event", "error", err)
		}
	}

	// Step 9: Generate clarifications for ambiguous action items
	for _, item := range extraction.ActionItems {
		if len(item.Interpretations) > 1 {
			optionsJSON, _ := json.Marshal(map[string]any{
				"type":            "ambiguous_action",
				"interpretations": item.Interpretations,
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

	// Generate clarifications for ambiguous deadlines
	for _, dl := range extraction.Deadlines {
		if !dl.IsAmbiguous {
			continue
		}

		optionsJSON, _ := json.Marshal(map[string]any{
			"type":        "ambiguous_deadline",
			"description": dl.Description,
			"raw_date":    dl.Date,
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

	// Step 10: Mark raw_input processed
	if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
		return IngestResult{}, fmt.Errorf("mark processed: %w", err)
	}

	return IngestResult{
		TaskIDs:  createdTaskIDs,
		EventIDs: createdEventIDs,
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
