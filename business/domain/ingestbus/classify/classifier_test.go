package classify

import (
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name           string
		clause         string
		expectedType   ItemType
		minConfidence  float64
		maxConfidence  float64
	}{
		// Clear tasks (obligation verbs)
		{
			name:          "finish the report",
			clause:        "Need to finish the report",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "call dentist",
			clause:        "Call my dentist",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "buy groceries",
			clause:        "Buy groceries",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "schedule meeting",
			clause:        "Schedule a meeting",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "don't forget invoice",
			clause:        "Don't forget to send the invoice",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "remember pick up milk",
			clause:        "Remember to pick up milk",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "review proposal",
			clause:        "Review the proposal",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "email team",
			clause:        "Email the team",
			expectedType:  TaskType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},

		// Clear events (temporal, no obligation)
		{
			name:          "Tuesday at 2pm",
			clause:        "Tuesday at 2pm",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "tomorrow at 10am",
			clause:        "Tomorrow at 10am",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "next Monday",
			clause:        "Next Monday",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "meeting Friday 3-4pm",
			clause:        "Meeting Friday 3-4pm",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "birthday April 15",
			clause:        "Birthday on April 15",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "today at 3",
			clause:        "Today at 3pm",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},
		{
			name:          "tomorrow morning",
			clause:        "Tomorrow morning",
			expectedType:  EventType,
			minConfidence: 0.85,
			maxConfidence: 1.0,
		},

		// Clear notes (pure reference)
		{
			name:          "phone number",
			clause:        "555-123-4567",
			expectedType:  NoteType,
			minConfidence: 0.90,
			maxConfidence: 1.0,
		},
		{
			name:          "email address",
			clause:        "john@example.com",
			expectedType:  NoteType,
			minConfidence: 0.90,
			maxConfidence: 1.0,
		},
		{
			name:          "street address",
			clause:        "123 Main Street Springfield",
			expectedType:  NoteType,
			minConfidence: 0.90,
			maxConfidence: 1.0,
		},
		{
			name:          "fact: person is role",
			clause:        "Sarah is a software engineer",
			expectedType:  NoteType,
			minConfidence: 0.80,
			maxConfidence: 1.0,
		},
		{
			name:          "fact: best place",
			clause:        "The best coffee place is Blue Bottle",
			expectedType:  NoteType,
			minConfidence: 0.80,
			maxConfidence: 1.0,
		},
		{
			name:          "phone with parens",
			clause:        "(555) 123-4567",
			expectedType:  NoteType,
			minConfidence: 0.90,
			maxConfidence: 1.0,
		},

		// Ambiguous cases (low confidence)
		{
			name:          "task with temporal",
			clause:        "Need to be there Tuesday at 2pm",
			expectedType:  TaskType,
			minConfidence: 0.5,
			maxConfidence: 0.7,
		},
		{
			name:          "call with time",
			clause:        "Call me tomorrow at 3",
			expectedType:  TaskType,
			minConfidence: 0.5,
			maxConfidence: 0.7,
		},
		{
			name:          "meeting at place",
			clause:        "Meeting is at the coffee shop",
			expectedType:  NoteType,
			minConfidence: 0.4,
			maxConfidence: 0.7,
		},
		{
			name:          "remember dentist Thursday",
			clause:        "Remember to go to the dentist on Thursday",
			expectedType:  TaskType,
			minConfidence: 0.5,
			maxConfidence: 0.7,
		},

		// Edge cases
		{
			name:          "empty string",
			clause:        "",
			expectedType:  NoteType,
			minConfidence: 0.0,
			maxConfidence: 0.1,
		},
		{
			name:          "whitespace only",
			clause:        "   ",
			expectedType:  NoteType,
			minConfidence: 0.0,
			maxConfidence: 0.1,
		},
		{
			name:          "generic text",
			clause:        "something random",
			expectedType:  NoteType,
			minConfidence: 0.4,
			maxConfidence: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Classify(tt.clause)

			if result.Type != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, result.Type)
			}

			if result.Confidence < tt.minConfidence || result.Confidence > tt.maxConfidence {
				t.Errorf("expected confidence between %f and %f, got %f",
					tt.minConfidence, tt.maxConfidence, result.Confidence)
			}
		})
	}
}

func TestClassifyTaskVerbs(t *testing.T) {
	taskVerbs := []string{
		"Need to finish",
		"Have to complete",
		"Should check",
		"Must do",
		"Want to read",
		"Remember to call",
		"Don't forget to send",
		"Make sure to update",
		"Create a new file",
		"Write the report",
	}

	for _, clause := range taskVerbs {
		result := Classify(clause)
		if result.Type != TaskType {
			t.Errorf("clause '%s' should be classified as task, got %s", clause, result.Type)
		}
		if result.Confidence < 0.8 {
			t.Errorf("clause '%s' should have high confidence, got %f", clause, result.Confidence)
		}
	}
}

func TestClassifyEventPatterns(t *testing.T) {
	eventPatterns := []string{
		"Monday at 2pm",
		"Friday afternoon",
		"Tomorrow evening",
		"Next Tuesday",
		"At 3:30",
		"In 2 days",
		"March 15",
	}

	for _, clause := range eventPatterns {
		result := Classify(clause)
		if result.Type != EventType {
			t.Errorf("clause '%s' should be classified as event, got %s", clause, result.Type)
		}
		if result.Confidence < 0.8 {
			t.Errorf("clause '%s' should have high confidence, got %f", clause, result.Confidence)
		}
	}
}

func TestClassifyNotePatterns(t *testing.T) {
	notePatterns := []string{
		"555-1234",
		"test@example.com",
		"42 Oak Street",
		"John is a developer",
		"The best pizza is from Tony's",
	}

	for _, clause := range notePatterns {
		result := Classify(clause)
		if result.Type != NoteType {
			t.Errorf("clause '%s' should be classified as note, got %s", clause, result.Type)
		}
		if result.Confidence < 0.8 {
			t.Errorf("clause '%s' should have high confidence, got %f", clause, result.Confidence)
		}
	}
}
