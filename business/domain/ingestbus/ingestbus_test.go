package ingestbus_test

// Tests require a running database. Run with:
//
//	make db-up && make migrate && make seed
//	go test ./business/domain/ingestbus/... -run TestProcessEmail -count=1 -v
//
// The test file validates:
// 1. Ambiguous deadlines from extraction produce ambiguous_deadline clarifications
// 2. Auto-context creation when SuggestNewContext=true produces new_context clarifications
// 3. Low-confidence context matching produces context_assignment clarifications
// 4. Multiple interpretations on action items produce ambiguous_action clarifications
