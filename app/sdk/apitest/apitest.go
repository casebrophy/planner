package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casebrophy/planner/app/domain/activitylogapp"
	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/domain/correctionapp"
	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/domain/dailyplanapp"
	"github.com/casebrophy/planner/app/domain/emailapp"
	"github.com/casebrophy/planner/app/domain/entitylinkapp"
	"github.com/casebrophy/planner/app/domain/eventapp"
	"github.com/casebrophy/planner/app/domain/noteapp"
	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/domain/rawinputapp"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/domain/voiceingestapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

// Test contains state for running API tests.
type Test struct {
	DB  *dbtest.Database
	mux http.Handler
}

// New creates a new Test with a live Docker Postgres database and
// a full HTTP handler stack using the TestAPIKey for authentication.
func New(t *testing.T, testName string) *Test {
	t.Helper()

	db := dbtest.New(t, testName)

	cfg := mux.Config{
		Log:    db.Log,
		DB:     db.DB,
		APIKey: TestAPIKey,
	}

	handler := mux.WebAPI(cfg,
		taskapp.Routes{},
		contextapp.Routes{},
		tagapp.Routes{},
		clarificationapp.Routes{},
		emailapp.Routes{},
		rawinputapp.Routes{},
		threadapp.Routes{},
		observationapp.Routes{},
		voiceingestapp.Routes{},
		eventapp.Routes{},
		dailyplanapp.Routes{},
		entitylinkapp.Routes{},
		noteapp.Routes{},
		activitylogapp.Routes{},
		correctionapp.Routes{},
	)

	return &Test{
		DB:  db,
		mux: handler,
	}
}

// Run performs the actual test logic based on the table data.
func (at *Test) Run(t *testing.T, table []Table, testName string) {
	t.Helper()

	for _, tt := range table {
		f := func(t *testing.T) {
			t.Helper()

			r := httptest.NewRequest(tt.Method, tt.URL, nil)
			w := httptest.NewRecorder()

			if tt.Input != nil {
				d, err := json.Marshal(tt.Input)
				if err != nil {
					t.Fatalf("Should be able to marshal the model : %s", err)
				}
				r = httptest.NewRequest(tt.Method, tt.URL, bytes.NewBuffer(d))
			}

			r.Header.Set("X-API-Key", tt.Token)
			at.mux.ServeHTTP(w, r)

			if w.Code != tt.StatusCode {
				t.Fatalf("%s: Should receive a status code of %d for the response : %d", tt.Name, tt.StatusCode, w.Code)
			}

			if tt.StatusCode == http.StatusNoContent {
				return
			}

			if tt.GotResp == nil {
				return
			}

			if err := json.Unmarshal(w.Body.Bytes(), tt.GotResp); err != nil {
				t.Fatalf("Should be able to unmarshal the response : %s", err)
			}

			diff := tt.CmpFunc(tt.GotResp, tt.ExpResp)
			if diff != "" {
				t.Log("DIFF")
				t.Logf("%s", diff)
				t.Log("GOT")
				t.Logf("%#v", tt.GotResp)
				t.Log("EXP")
				t.Logf("%#v", tt.ExpResp)
				t.Fatalf("Should get the expected response")
			}
		}

		t.Run(testName+"-"+tt.Name, f)
	}
}
