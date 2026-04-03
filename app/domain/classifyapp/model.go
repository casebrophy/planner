package classifyapp

import "encoding/json"

// ClassifyResult is the response from the classify endpoint.
type ClassifyResult struct {
	Classified            int `json:"classified"`
	ClarificationsCreated int `json:"clarificationsCreated"`
}

// Encode implements web.Encoder.
func (r ClassifyResult) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
