package http

import (
	"encoding/json"
	"net/http"
	"reflect"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(coalesceNilSlice(body))
}

// coalesceNilSlice ensures a nil slice serializes as `[]` instead of `null`,
// so list endpoints always return a JSON array. Without this, an empty
// result (e.g. no prospects yet) marshals to `null`, and a frontend doing
// `list.map(...)` crashes — which, without an error boundary, white-screens
// the whole React app.
func coalesceNilSlice(body any) any {
	if body == nil {
		return body
	}
	v := reflect.ValueOf(body)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return body
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
