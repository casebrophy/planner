package voiceingestapp

import (
	"encoding/json"
)

type ingestRequest struct {
	Text string `json:"text"`
}

type ingestResponse struct {
	TaskIDs []string `json:"taskIds"`
}

func (ir ingestResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(ir)
	return data, "application/json", err
}
