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
