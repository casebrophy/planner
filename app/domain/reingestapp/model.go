package reingestapp

import "encoding/json"

// ReingestResponse is returned after successfully enqueuing a reingest.
type ReingestResponse struct {
	RawInputID   string `json:"rawInputId"`
	SkipClassify bool   `json:"skipClassify"`
	Enqueued     bool   `json:"enqueued"`
}

// Encode implements web.Encoder.
func (r ReingestResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}

// BulkReingestRequest is the request body for bulk reingest.
type BulkReingestRequest struct {
	EntityType string `json:"entityType"`
	ContextID  string `json:"contextId,omitempty"`
	DateRange  *struct {
		Start string `json:"start,omitempty"`
		End   string `json:"end,omitempty"`
	} `json:"dateRange,omitempty"`
}

// BulkReingestResponse is returned after queuing bulk reingest items.
type BulkReingestResponse struct {
	Queued int `json:"queued"`
}

// Encode implements web.Encoder.
func (r BulkReingestResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
