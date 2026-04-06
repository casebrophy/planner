package voiceingestapp

import (
	"encoding/json"
)

type ingestRequest struct {
	Text string `json:"text"`
}

type ingestResponse struct {
	RawInputID string `json:"rawInputId"`
}

func (r ingestResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
