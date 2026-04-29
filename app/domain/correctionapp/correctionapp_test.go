package correctionapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/dbtest"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

const testAPIKey = "test-api-key"

func newHandler(db *dbtest.Database) *app {
	return &app{
		db:            db.DB,
		taskBus:       db.BusDomain.Task,
		noteBus:       db.BusDomain.Note,
		eventBus:      db.BusDomain.Event,
		correctionBus: db.BusDomain.ClassificationCorrection,
	}
}

func postCorrection(t *testing.T, hdl *app, ctx context.Context, body CorrectionRequest) ([]byte, error) {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/corrections", strings.NewReader(string(jsonBody)))
	r.Header.Set("X-API-Key", testAPIKey)
	result := hdl.correct(ctx, r)
	if result == nil {
		return nil, fmt.Errorf("nil result from handler")
	}
	data, _, encErr := result.Encode()
	if encErr != nil {
		return nil, fmt.Errorf("encode result: %w", encErr)
	}
	// Check for error response shape by probing the numeric "code" field
	// (errs.Error always has code; CorrectionResult does not).
	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err == nil {
		if codeRaw, hasCode := probe["code"]; hasCode {
			if _, ok := codeRaw.(float64); ok {
				return data, fmt.Errorf("error response: %s", string(data))
			}
		}
	}
	return data, nil
}

func decodeResult(t *testing.T, data []byte) CorrectionResult {
	t.Helper()
	var res CorrectionResult
	if err := json.Unmarshal(data, &res); err != nil {
		t.Fatalf("unmarshal result: %v (raw: %s)", err, string(data))
	}
	return res
}

func assertCorrectionRecorded(t *testing.T, ctx context.Context, db *dbtest.Database, predicted, actual string) {
	t.Helper()
	source := "correction_applied"
	corrs, err := db.BusDomain.ClassificationCorrection.Query(ctx, classificationcorrectionbus.QueryFilter{
		Source:        &source,
		PredictedType: &predicted,
		ActualType:    &actual,
	}, classificationcorrectionbus.DefaultOrderBy, page.New(1, 10))
	if err != nil {
		t.Fatalf("query corrections: %v", err)
	}
	if len(corrs) == 0 {
		t.Fatalf("expected a recorded correction (predicted=%s actual=%s), got none", predicted, actual)
	}
}

func TestCorrect_TaskToNote(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_TaskToNote")
	ctx := context.Background()

	// Create raw input and tag for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Buy groceries: milk, eggs",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	tag, err := db.BusDomain.Tag.Create(ctx, tagbus.NewTag{
		Name: "test-tag",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "Buy groceries",
		Description: "Milk, eggs",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		RawInputID:  &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source task: %v", err)
	}

	// Add tag to task
	_, err = db.DB.ExecContext(ctx, "INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)", task.ID, tag.ID)
	if err != nil {
		t.Fatalf("add tag to task: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   task.ID.String(),
		ItemType: "task",
		NewType:  "note",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "note" {
		t.Fatalf("expected type=note, got %q", res.Type)
	}

	newID, err := uuid.Parse(res.ID)
	if err != nil {
		t.Fatalf("parse new id: %v", err)
	}
	note, err := db.BusDomain.Note.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new note: %v", err)
	}
	if !strings.Contains(note.Content, "Buy groceries") {
		t.Errorf("expected note content to contain title, got %q", note.Content)
	}
	if note.Source != "correction" {
		t.Errorf("expected source=correction, got %q", note.Source)
	}

	// Assert lineage preservation
	if note.RawInputID == nil || *note.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source task, got %v", note.RawInputID)
	}
	if note.CreatedAt.Sub(task.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source task, got %v vs %v", note.CreatedAt, task.CreatedAt)
	}

	// Assert tag was copied
	var tagCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM note_tags WHERE note_id = $1 AND tag_id = $2", note.ID, tag.ID).Scan(&tagCount)
	if err != nil {
		t.Fatalf("query note_tags: %v", err)
	}
	if tagCount != 1 {
		t.Errorf("expected 1 tag in note_tags, got %d", tagCount)
	}

	if _, err := db.BusDomain.Task.QueryByID(ctx, task.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source task to be deleted (ErrDBNotFound), got err=%v", err)
	}

	assertCorrectionRecorded(t, ctx, db, "task", "note")
}

func TestCorrect_TaskToEvent(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_TaskToEvent")
	ctx := context.Background()

	// Create raw input for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Dentist visit: annual checkup",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:       "Dentist visit",
		Description: "Annual checkup",
		Status:      taskstatus.Open,
		Priority:    taskpriority.Medium,
		RawInputID:  &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source task: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   task.ID.String(),
		ItemType: "task",
		NewType:  "event",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "event" {
		t.Fatalf("expected type=event, got %q", res.Type)
	}

	newID, _ := uuid.Parse(res.ID)
	event, err := db.BusDomain.Event.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new event: %v", err)
	}
	if event.Title != "Dentist visit" {
		t.Errorf("expected title=Dentist visit, got %q", event.Title)
	}
	if event.Description != "Annual checkup" {
		t.Errorf("expected description preserved, got %q", event.Description)
	}
	if event.AllDay {
		t.Errorf("expected all_day=false")
	}

	// Assert lineage preservation
	if event.RawInputID == nil || *event.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source task, got %v", event.RawInputID)
	}
	if event.CreatedAt.Sub(task.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source task, got %v vs %v", event.CreatedAt, task.CreatedAt)
	}

	if _, err := db.BusDomain.Task.QueryByID(ctx, task.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source task deleted, got err=%v", err)
	}
	assertCorrectionRecorded(t, ctx, db, "task", "event")
}

func TestCorrect_NoteToTask(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_NoteToTask")
	ctx := context.Background()

	// Create raw input and tag for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Email,
		RawContent: "Remember to renew passport",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	tag, err := db.BusDomain.Tag.Create(ctx, tagbus.NewTag{
		Name: "admin",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	note, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content:    "Remember to renew passport",
		Source:     "manual",
		RawInputID: &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source note: %v", err)
	}

	// Add tag to note
	_, err = db.DB.ExecContext(ctx, "INSERT INTO note_tags (note_id, tag_id) VALUES ($1, $2)", note.ID, tag.ID)
	if err != nil {
		t.Fatalf("add tag to note: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   note.ID.String(),
		ItemType: "note",
		NewType:  "task",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "task" {
		t.Fatalf("expected type=task, got %q", res.Type)
	}

	newID, _ := uuid.Parse(res.ID)
	task, err := db.BusDomain.Task.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new task: %v", err)
	}
	if task.Title != "Remember to renew passport" {
		t.Errorf("expected title=truncated content, got %q", task.Title)
	}
	if task.Description != "" {
		t.Errorf("expected empty description, got %q", task.Description)
	}
	if task.Status != taskstatus.Open {
		t.Errorf("expected status=open, got %v", task.Status)
	}

	// Assert lineage preservation
	if task.RawInputID == nil || *task.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source note, got %v", task.RawInputID)
	}
	if task.CreatedAt.Sub(note.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source note, got %v vs %v", task.CreatedAt, note.CreatedAt)
	}

	// Assert tag was copied
	var tagCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_tags WHERE task_id = $1 AND tag_id = $2", task.ID, tag.ID).Scan(&tagCount)
	if err != nil {
		t.Fatalf("query task_tags: %v", err)
	}
	if tagCount != 1 {
		t.Errorf("expected 1 tag in task_tags, got %d", tagCount)
	}

	if _, err := db.BusDomain.Note.QueryByID(ctx, note.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source note deleted, got err=%v", err)
	}
	assertCorrectionRecorded(t, ctx, db, "note", "task")
}

func TestCorrect_NoteToEvent(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_NoteToEvent")
	ctx := context.Background()

	// Create raw input for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Team standup",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	note, err := db.BusDomain.Note.Create(ctx, notebus.NewNote{
		Content:    "Team standup",
		Source:     "manual",
		RawInputID: &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source note: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   note.ID.String(),
		ItemType: "note",
		NewType:  "event",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "event" {
		t.Fatalf("expected type=event, got %q", res.Type)
	}

	newID, _ := uuid.Parse(res.ID)
	event, err := db.BusDomain.Event.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new event: %v", err)
	}
	if event.Title != "Team standup" {
		t.Errorf("expected title=Team standup, got %q", event.Title)
	}
	if event.AllDay {
		t.Errorf("expected all_day=false")
	}

	// Assert lineage preservation
	if event.RawInputID == nil || *event.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source note, got %v", event.RawInputID)
	}
	if event.CreatedAt.Sub(note.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source note, got %v vs %v", event.CreatedAt, note.CreatedAt)
	}

	// Assert no tags were created for event (no event_tags table)
	var noteTagCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM note_tags WHERE note_id = $1", note.ID).Scan(&noteTagCount)
	if err != nil {
		t.Fatalf("query note_tags: %v", err)
	}
	if noteTagCount != 0 {
		t.Errorf("expected 0 tags in note_tags (note still exists), got %d", noteTagCount)
	}

	if _, err := db.BusDomain.Note.QueryByID(ctx, note.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source note deleted, got err=%v", err)
	}
	assertCorrectionRecorded(t, ctx, db, "note", "event")
}

func TestCorrect_EventToTask(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_EventToTask")
	ctx := context.Background()

	// Create raw input for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Pick up dry cleaning before 5pm",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	event, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:       "Pick up dry cleaning",
		Description: "Before 5pm",
		StartsAt:    time.Now().Add(1 * time.Hour),
		EndsAt:      time.Now().Add(2 * time.Hour),
		RawInputID:  &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source event: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   event.ID.String(),
		ItemType: "event",
		NewType:  "task",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "task" {
		t.Fatalf("expected type=task, got %q", res.Type)
	}

	newID, _ := uuid.Parse(res.ID)
	task, err := db.BusDomain.Task.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new task: %v", err)
	}
	if task.Title != "Pick up dry cleaning" {
		t.Errorf("expected title preserved, got %q", task.Title)
	}
	if task.Description != "Before 5pm" {
		t.Errorf("expected description preserved, got %q", task.Description)
	}

	// Assert lineage preservation
	if task.RawInputID == nil || *task.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source event, got %v", task.RawInputID)
	}
	if task.CreatedAt.Sub(event.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source event, got %v vs %v", task.CreatedAt, event.CreatedAt)
	}

	if _, err := db.BusDomain.Event.QueryByID(ctx, event.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source event deleted, got err=%v", err)
	}
	assertCorrectionRecorded(t, ctx, db, "event", "task")
}

func TestCorrect_EventToNote(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_EventToNote")
	ctx := context.Background()

	// Create raw input for lineage preservation testing
	ri, err := db.BusDomain.RawInput.Create(ctx, rawinputbus.NewRawInput{
		SourceType: rawinputsource.Voice,
		RawContent: "Coffee with Alex to discuss roadmap",
	})
	if err != nil {
		t.Fatalf("create raw input: %v", err)
	}

	event, err := db.BusDomain.Event.Create(ctx, eventbus.NewEvent{
		Title:       "Coffee with Alex",
		Description: "Discuss roadmap",
		StartsAt:    time.Now().Add(1 * time.Hour),
		EndsAt:      time.Now().Add(2 * time.Hour),
		RawInputID:  &ri.ID,
	})
	if err != nil {
		t.Fatalf("create source event: %v", err)
	}

	hdl := newHandler(db)
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   event.ID.String(),
		ItemType: "event",
		NewType:  "note",
	})
	if err != nil {
		t.Fatalf("correct: %v", err)
	}
	res := decodeResult(t, data)
	if res.Type != "note" {
		t.Fatalf("expected type=note, got %q", res.Type)
	}

	newID, _ := uuid.Parse(res.ID)
	note, err := db.BusDomain.Note.QueryByID(ctx, newID)
	if err != nil {
		t.Fatalf("query new note: %v", err)
	}
	if !strings.Contains(note.Content, "Coffee with Alex") {
		t.Errorf("expected content to contain title, got %q", note.Content)
	}
	if !strings.Contains(note.Content, "Discuss roadmap") {
		t.Errorf("expected content to contain description, got %q", note.Content)
	}

	// Assert lineage preservation
	if note.RawInputID == nil || *note.RawInputID != ri.ID {
		t.Errorf("expected RawInputID to be preserved from source event, got %v", note.RawInputID)
	}
	if note.CreatedAt.Sub(event.CreatedAt).Abs() > time.Millisecond {
		t.Errorf("expected CreatedAt to be preserved from source event, got %v vs %v", note.CreatedAt, event.CreatedAt)
	}

	if _, err := db.BusDomain.Event.QueryByID(ctx, event.ID); !errors.Is(err, sqldb.ErrDBNotFound) {
		t.Errorf("expected source event deleted, got err=%v", err)
	}
	assertCorrectionRecorded(t, ctx, db, "event", "note")
}

func TestCorrect_InvalidTypes(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_InvalidTypes")
	ctx := context.Background()
	hdl := newHandler(db)
	validID := uuid.New().String()

	cases := []struct {
		name     string
		body     CorrectionRequest
		wantCode int // InvalidArgument = 3
	}{
		{
			name:     "bad item_type",
			body:     CorrectionRequest{ItemID: validID, ItemType: "widget", NewType: "task"},
			wantCode: 3,
		},
		{
			name:     "bad new_type",
			body:     CorrectionRequest{ItemID: validID, ItemType: "task", NewType: "widget"},
			wantCode: 3,
		},
		{
			name:     "item_type equals new_type",
			body:     CorrectionRequest{ItemID: validID, ItemType: "task", NewType: "task"},
			wantCode: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := postCorrection(t, hdl, ctx, tc.body)
			if err == nil {
				t.Fatalf("expected error response, got success: %s", string(data))
			}
			var errMap map[string]any
			if jerr := json.Unmarshal(data, &errMap); jerr != nil {
				t.Fatalf("unmarshal error response: %v", jerr)
			}
			code, ok := errMap["code"].(float64)
			if !ok {
				t.Fatalf("expected numeric code field, got %v", errMap["code"])
			}
			if int(code) != tc.wantCode {
				t.Errorf("expected code=%d (InvalidArgument), got %v", tc.wantCode, code)
			}
		})
	}
}

func TestCorrect_NotFound(t *testing.T) {
	db := dbtest.New(t, "TestCorrect_NotFound")
	ctx := context.Background()

	hdl := newHandler(db)
	missingID := uuid.New()
	data, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   missingID.String(),
		ItemType: "task",
		NewType:  "note",
	})
	if err == nil {
		t.Fatalf("expected error response for missing source, got success: %s", string(data))
	}

	var errMap map[string]any
	if jerr := json.Unmarshal(data, &errMap); jerr != nil {
		t.Fatalf("unmarshal error response: %v", jerr)
	}
	// errs.Error serializes Code as numeric. NotFound = 5.
	if code, ok := errMap["code"].(float64); !ok || int(code) != 5 {
		t.Errorf("expected code=5 (NotFound), got %v", errMap["code"])
	}
}

func TestCorrect_RecordsCorrection(t *testing.T) {
	// Specific regression test for the silent-swallow bug: ensure correction
	// rows are persisted (previously failures were ignored).
	db := dbtest.New(t, "TestCorrect_RecordsCorrection")
	ctx := context.Background()

	task, err := db.BusDomain.Task.Create(ctx, taskbus.NewTask{
		Title:    "Some misclassified item",
		Status:   taskstatus.Open,
		Priority: taskpriority.Medium,
	})
	if err != nil {
		t.Fatalf("create source task: %v", err)
	}

	hdl := newHandler(db)
	if _, err := postCorrection(t, hdl, ctx, CorrectionRequest{
		ItemID:   task.ID.String(),
		ItemType: "task",
		NewType:  "note",
	}); err != nil {
		t.Fatalf("correct: %v", err)
	}

	source := "correction_applied"
	corrs, err := db.BusDomain.ClassificationCorrection.QueryBySource(ctx, source, page.New(1, 10))
	if err != nil {
		t.Fatalf("query by source: %v", err)
	}
	found := false
	for _, c := range corrs {
		if c.PredictedType == "task" && c.ActualType == "note" && c.ClauseText == "Some misclassified item" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected correction row for task->note with clause_text matching source title; got %d corrections", len(corrs))
	}
}
