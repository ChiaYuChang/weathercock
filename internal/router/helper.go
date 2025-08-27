package router

import (
	"encoding/json"
	"net/http"

	ec "github.com/ChiaYuChang/weathercock/pkgs/errors"
	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/rs/zerolog"
)

func fireErrResp(w http.ResponseWriter, r *http.Request, logger zerolog.Logger,
	header map[string]string, msg string, err error) {

	e, ok := err.(*ec.Error)
	e = utils.IfElse(ok, e, ec.ErrInternalServerError.Clone())

	event := logger.Error().
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int("http_status_code", e.HttpStatusCode).
		Int("internal_status_code", e.InternalStatusCode).
		Strs("details", e.Details).
		Err(e.Unwrap())
	for k, v := range header {
		event = event.Str(k, v)
		w.Header().Set(k, v)
	}
	event.Msg(msg)
	w.WriteHeader(e.HttpStatusCode)
	e.MarshalAndWriteTo(w)
}

func fireOkResp(w http.ResponseWriter, r *http.Request, logger zerolog.Logger,
	header map[string]string, data json.RawMessage) {

	event := logger.Error().
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Str("remote_addr", r.RemoteAddr).
		Str("user_agent", r.UserAgent()).
		Int("http_status_code", http.StatusOK).
		Int("internal_status_code", ec.ECSuccess).
		Int("data_length", len(data))
	for k, v := range header {
		event = event.Str(k, v)
		w.Header().Set(k, v)
	}
	event.Msg("success")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
