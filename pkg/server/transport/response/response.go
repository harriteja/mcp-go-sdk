package response

import (
	"encoding/json"
	"net/http"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
)

// WriteError writes an error response with proper headers
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := &types.Error{
		Code:    code,
		Message: message,
	}
	_ = json.NewEncoder(w).Encode(struct {
		Error *types.Error `json:"error"`
	}{
		Error: err,
	})
}

// WriteJSON writes a JSON response with proper headers
func WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(struct {
		Result interface{} `json:"result"`
	}{
		Result: data,
	})
}
