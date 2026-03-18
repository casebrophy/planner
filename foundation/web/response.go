package web

import (
	"context"
	"net/http"
)

type Encoder interface {
	Encode() (data []byte, contentType string, err error)
}

type HTTPStatuser interface {
	HTTPStatus() int
}

func Respond(ctx context.Context, w http.ResponseWriter, encoder Encoder) error {
	if encoder == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	data, ct, err := encoder.Encode()
	if err != nil {
		return err
	}

	statusCode := http.StatusOK
	if s, ok := encoder.(HTTPStatuser); ok {
		statusCode = s.HTTPStatus()
	}

	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	w.WriteHeader(statusCode)

	if data != nil {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}
