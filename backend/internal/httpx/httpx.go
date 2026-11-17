// Package httpx holds the small HTTP helpers every handler uses:
// writing JSON, the shared error format, and a panic recoverer.
//
// Every error response has the same shape:
//
//	{ "error": { "code": "invalid_parameter", "message": "sort must be one of …" } }
//
// `code` is stable and meant for the frontend to switch on;
// `message` is human-readable and may change.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Error codes used across the API.
const (
	CodeInvalidParameter = "invalid_parameter" // 400: a query/body value is wrong
	CodeNotFound         = "not_found"         // 404: no such route or resource
	CodeMethodNotAllowed = "method_not_allowed"
	CodeInternal         = "internal" // 500: our fault; details only go to the log
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

// Error writes the shared error format.
func Error(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	JSON(w, status, body)
}

// BadRequest is the 400 shortcut used for validation errors.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, CodeInvalidParameter, message)
}

// Internal logs the real error and sends the client a generic 500,
// so database details never leak into responses.
func Internal(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("internal error on %s %s: %v", r.Method, r.URL.Path, err)
	Error(w, http.StatusInternalServerError, CodeInternal, "Something went wrong. Please try again.")
}

// NotFound and MethodNotAllowed replace chi's plain-text defaults.
func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, http.StatusNotFound, CodeNotFound, "No route for "+r.Method+" "+r.URL.Path)
}

func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	Error(w, http.StatusMethodNotAllowed, CodeMethodNotAllowed, r.Method+" is not allowed on "+r.URL.Path)
}

// Recoverer turns a panic in any handler into a JSON 500 instead of
// killing the connection. (chi has one too, but it answers in plain text.)
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler { // client went away; nothing to answer
					panic(rec)
				}
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				Error(w, http.StatusInternalServerError, CodeInternal, "Something went wrong. Please try again.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
