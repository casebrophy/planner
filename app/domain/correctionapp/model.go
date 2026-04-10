package correctionapp

import "encoding/json"

type CorrectionRequest struct {
	ItemID   string `json:"item_id"`
	ItemType string `json:"item_type"`
	NewType  string `json:"new_type"`
}

type CorrectionResult struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (r CorrectionResult) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
