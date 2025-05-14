package http

import (
	"net/http"
)

// ResponseWriter wraps http.ResponseWriter to capture status code and bytes written
type ResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// WriteHeader captures the status code and writes it to the underlying ResponseWriter
func (w *ResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write captures the number of bytes written and writes to the underlying ResponseWriter
func (w *ResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// Status returns the HTTP status code of the response
func (w *ResponseWriter) Status() int {
	return w.status
}

// BytesWritten returns the number of bytes written in the response
func (w *ResponseWriter) BytesWritten() int64 {
	return w.bytesWritten
}

// Unwrap returns the original http.ResponseWriter
func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
