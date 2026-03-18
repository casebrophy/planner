package web

import (
	"encoding/json"
	"io"
	"net/http"
)

func Decode(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.Unmarshal(body, v)
}

func Param(r *http.Request, key string) string {
	return r.PathValue(key)
}
