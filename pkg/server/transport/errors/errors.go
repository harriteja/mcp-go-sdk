package errors

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// WriteError writes an error response with proper headers
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(types.NewError(code, message)); err != nil {
		log.Printf("Failed to encode error: %v", err)
	}
}

// WriteJSON writes a JSON response with proper headers
func WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(struct {
		Result interface{} `json:"result"`
	}{
		Result: data,
	}); err != nil {
		log.Printf("Failed to encode JSON: %v", err)
	}
}

// WriteErrorObject writes an error response with proper headers
func WriteErrorObject(w http.ResponseWriter, code int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(struct {
		Error interface{} `json:"error"`
	}{
		Error: obj,
	}); err != nil {
		log.Printf("Failed to encode error object: %v", err)
	}
}

// Common error responses
var (
	ErrUnauthorized          = types.NewError(http.StatusUnauthorized, "Unauthorized")
	ErrForbidden             = types.NewError(http.StatusForbidden, "Forbidden")
	ErrNotFound              = types.NewError(http.StatusNotFound, "Not found")
	ErrMethodNotAllowed      = types.NewError(http.StatusMethodNotAllowed, "Method not allowed")
	ErrTooManyRequests       = types.NewError(http.StatusTooManyRequests, "Too many requests")
	ErrRequestEntityTooLarge = types.NewError(http.StatusRequestEntityTooLarge, "Request too large")
	ErrInternalServer        = types.NewError(http.StatusInternalServerError, "Internal server error")
)

// Common headers
const (
	HeaderContentType              = "Content-Type"
	HeaderAuthorization            = "Authorization"
	HeaderSessionID                = "mcp-session-id"
	HeaderLastEventID              = "last-event-id"
	HeaderCacheControl             = "Cache-Control"
	HeaderConnection               = "Connection"
	HeaderAccessControlAllowOrigin = "Access-Control-Allow-Origin"
)

// Common content types
const (
	ContentTypeJSON      = "application/json"
	ContentTypeSSE       = "text/event-stream"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)
