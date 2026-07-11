package runtime

import (
	"context"
	"encoding/json"
	"net/http"
)

// ResponseWriter serializes a controller's return value to the HTTP response
// (§22). The framework depends only on this interface so the encoding is
// pluggable.
type ResponseWriter interface {
	Write(ctx context.Context, w http.ResponseWriter, status int, value any) error
}

// JSONResponseWriter encodes values as JSON. A nil value or a 204 status writes
// only the status line and no body.
type JSONResponseWriter struct{}

// Write implements ResponseWriter.
func (JSONResponseWriter) Write(_ context.Context, w http.ResponseWriter, status int, value any) error {
	if value == nil || status == http.StatusNoContent {
		w.WriteHeader(status)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}
