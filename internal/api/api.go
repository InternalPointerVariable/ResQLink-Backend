package api

import (
	"log/slog"
	"net/http"
)

type HTTPHandler func(w http.ResponseWriter, r *http.Request) Response

func (fn HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := fn(w, r)

	if res.Error != nil {
		slog.Error(res.Error.Error())
	}

	if err := res.Encode(w); err != nil {
		slog.Error(err.Error())
	}
}
