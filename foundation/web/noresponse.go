package web

import "net/http"

type NoResponse struct{}

func (NoResponse) Encode() ([]byte, string, error) {
	return nil, "", nil
}

func (NoResponse) HTTPStatus() int {
	return http.StatusNoContent
}
