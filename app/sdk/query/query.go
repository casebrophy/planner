package query

import "encoding/json"

type Result[T any] struct {
	Items       []T `json:"items"`
	Total       int `json:"total"`
	Page        int `json:"page"`
	RowsPerPage int `json:"rowsPerPage"`
}

func NewResult[T any](items []T, total int, page int, rowsPerPage int) Result[T] {
	if items == nil {
		items = []T{}
	}
	return Result[T]{
		Items:       items,
		Total:       total,
		Page:        page,
		RowsPerPage: rowsPerPage,
	}
}

func (r Result[T]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}
