// Package httpx is the HTTP toolkit every endpoint uses: JSON, the `200 null` refusal,
// the error shape, request decoding and middleware (BACKEND_PLAN.md §4.2, §16).
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

// ErrorHeader carries the refusal code on a `200 null` answer.
const ErrorHeader = "X-Waraqah-Error"

// JSON writes v as a 200 application/json answer.
func JSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, `{"error":{"code":"internal","message":"encoding failed"}}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// Null writes the JSON value null (a lookup miss).
func Null(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("null"))
}

// Refuse answers 200 null with X-Waraqah-Error: code, and logs the refusal.
func Refuse(w http.ResponseWriter, r *http.Request, code string) {
	w.Header().Set(ErrorHeader, code)
	logx.From(r.Context()).Info("refusal", "refusal", code)
	Null(w)
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// Error writes the error shape with the given status. Internal details stay in the logs.
func Error(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	id := w.Header().Get("X-Request-ID")
	body, _ := json.Marshal(errorBody{errorDetail{Code: code, Message: msg, RequestID: id}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// Decode reads a JSON body into v. On failure it writes a 400 (or 413) and returns false.
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		Error(w, r, http.StatusBadRequest, CodeBadRequest, "request body is empty")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			Error(w, r, http.StatusRequestEntityTooLarge, CodeTooLarge, "request body is too large")
			return false
		}
		Error(w, r, http.StatusBadRequest, CodeBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// Query returns the first value of a query parameter ("" when absent).
func Query(r *http.Request, key string) string { return r.URL.Query().Get(key) }

// QueryBool reads a boolean query parameter ("true" or "1").
func QueryBool(r *http.Request, key string) bool {
	v := r.URL.Query().Get(key)
	return v == "true" || v == "1"
}

// QueryInt reads an integer query parameter, returning def when absent or invalid.
func QueryInt(r *http.Request, key string, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return n
}
