package classifyapp

import "encoding/json"

// ClassifyAccepted is returned immediately when classification is enqueued in the background.
type ClassifyAccepted struct {
	Message       string `json:"message"`
	UnlinkedCount int    `json:"unlinkedCount"`
}

// Encode implements web.Encoder.
func (r ClassifyAccepted) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
