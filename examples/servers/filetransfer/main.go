package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server/middleware"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

const maxUploadSize = 10 << 20 // 10 MB

// FileInfo represents file metadata
type FileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// FileServer represents a file transfer server
type FileServer struct {
	logger     types.Logger
	uploadDir  string
	files      map[string]FileInfo
	maxSize    int64
	allowTypes []string
}

// NewFileServer creates a new file transfer server
func NewFileServer(logger types.Logger, uploadDir string, maxSize int64, allowTypes []string) *FileServer {
	// Create upload directory if it doesn't exist
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create upload directory: %v", err))
		}
	}

	return &FileServer{
		logger:     logger,
		uploadDir:  uploadDir,
		files:      make(map[string]FileInfo),
		maxSize:    maxSize,
		allowTypes: allowTypes,
	}
}

// ListFiles returns a list of uploaded files
func (s *FileServer) ListFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.logger.Info(ctx, "file", "list", "Listing files")

	// Read files from directory
	entries, err := os.ReadDir(s.uploadDir)
	if err != nil {
		s.logger.Error(ctx, "file", "list", fmt.Sprintf("Failed to read directory: %v", err))
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			s.logger.Error(ctx, "file", "list", fmt.Sprintf("Failed to get file info: %v", err))
			continue
		}

		contentType := "application/octet-stream"
		if idx := strings.LastIndex(info.Name(), "."); idx >= 0 {
			ext := strings.ToLower(info.Name()[idx+1:])
			switch ext {
			case "txt":
				contentType = "text/plain"
			case "pdf":
				contentType = "application/pdf"
			case "json":
				contentType = "application/json"
			case "png":
				contentType = "image/png"
			case "jpg", "jpeg":
				contentType = "image/jpeg"
			}
		}

		files = append(files, FileInfo{
			Name:        info.Name(),
			Size:        info.Size(),
			ContentType: contentType,
			UploadedAt:  info.ModTime(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		s.logger.Error(ctx, "file", "list", fmt.Sprintf("Failed to encode response: %v", err))
	}
}

// UploadFile handles file uploads
func (s *FileServer) UploadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info(ctx, "file", "upload", "Received file upload request")

	// Parse multipart form
	if err := r.ParseMultipartForm(s.maxSize); err != nil {
		s.logger.Error(ctx, "file", "upload", fmt.Sprintf("Failed to parse form: %v", err))
		http.Error(w, "Request too large", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		s.logger.Error(ctx, "file", "upload", fmt.Sprintf("Failed to get file: %v", err))
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > s.maxSize {
		s.logger.Warn(ctx, "file", "upload", fmt.Sprintf("File too large: %d bytes", header.Size))
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// Check content type if allowed types are specified
	if len(s.allowTypes) > 0 {
		contentType := header.Header.Get("Content-Type")
		allowed := false
		for _, allowType := range s.allowTypes {
			if strings.HasPrefix(contentType, allowType) {
				allowed = true
				break
			}
		}
		if !allowed {
			s.logger.Warn(ctx, "file", "upload", fmt.Sprintf("Invalid content type: %s", contentType))
			http.Error(w, "Invalid content type", http.StatusBadRequest)
			return
		}
	}

	// Create output file
	filename := filepath.Join(s.uploadDir, header.Filename)
	dst, err := os.Create(filename)
	if err != nil {
		s.logger.Error(ctx, "file", "upload", fmt.Sprintf("Failed to create file: %v", err))
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, file); err != nil {
		s.logger.Error(ctx, "file", "upload", fmt.Sprintf("Failed to write file: %v", err))
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	s.logger.Info(ctx, "file", "upload", fmt.Sprintf("Uploaded file: %s (%d bytes)", header.Filename, header.Size))

	// Create response
	fileInfo := FileInfo{
		Name:        header.Filename,
		Size:        header.Size,
		ContentType: header.Header.Get("Content-Type"),
		UploadedAt:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(fileInfo); err != nil {
		s.logger.Error(ctx, "file", "upload", fmt.Sprintf("Failed to encode response: %v", err))
	}
}

// DownloadFile handles file downloads
func (s *FileServer) DownloadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filename from URL path
	filename := filepath.Base(r.URL.Path)
	if filename == "" || filename == "." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	s.logger.Info(ctx, "file", "download", fmt.Sprintf("Download request for file: %s", filename))

	// Prevent path traversal
	fullPath := filepath.Join(s.uploadDir, filename)
	if !strings.HasPrefix(fullPath, s.uploadDir) {
		s.logger.Warn(ctx, "file", "download", "Path traversal attempt detected")
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		s.logger.Warn(ctx, "file", "download", fmt.Sprintf("File not found: %s", filename))
		http.Error(w, "File not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.Error(ctx, "file", "download", fmt.Sprintf("Failed to stat file: %v", err))
		http.Error(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		s.logger.Error(ctx, "file", "download", fmt.Sprintf("Failed to open file: %v", err))
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set content type based on extension
	contentType := "application/octet-stream"
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		ext := strings.ToLower(filename[idx+1:])
		switch ext {
		case "txt":
			contentType = "text/plain"
		case "pdf":
			contentType = "application/pdf"
		case "json":
			contentType = "application/json"
		case "png":
			contentType = "image/png"
		case "jpg", "jpeg":
			contentType = "image/jpeg"
		}
	}

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	// Stream file
	if _, err := io.Copy(w, file); err != nil {
		s.logger.Error(ctx, "file", "download", fmt.Sprintf("Failed to stream file: %v", err))
		return
	}

	s.logger.Info(ctx, "file", "download", fmt.Sprintf("Downloaded file: %s (%d bytes)", filename, info.Size()))
}

// DeleteFile handles file deletion
func (s *FileServer) DeleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filename from URL path
	filename := filepath.Base(r.URL.Path)
	if filename == "" || filename == "." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	s.logger.Info(ctx, "file", "delete", fmt.Sprintf("Delete request for file: %s", filename))

	// Prevent path traversal
	fullPath := filepath.Join(s.uploadDir, filename)
	if !strings.HasPrefix(fullPath, s.uploadDir) {
		s.logger.Warn(ctx, "file", "delete", "Path traversal attempt detected")
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Check if file exists
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		s.logger.Warn(ctx, "file", "delete", fmt.Sprintf("File not found: %s", filename))
		http.Error(w, "File not found", http.StatusNotFound)
		return
	} else if err != nil {
		s.logger.Error(ctx, "file", "delete", fmt.Sprintf("Failed to stat file: %v", err))
		http.Error(w, "Failed to access file", http.StatusInternalServerError)
		return
	}

	// Delete file
	if err := os.Remove(fullPath); err != nil {
		s.logger.Error(ctx, "file", "delete", fmt.Sprintf("Failed to delete file: %v", err))
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	s.logger.Info(ctx, "file", "delete", fmt.Sprintf("Deleted file: %s", filename))

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Server address")
	uploadDir := flag.String("upload-dir", "./uploads", "Directory for uploaded files")
	maxSize := flag.Int64("max-size", maxUploadSize, "Maximum upload size in bytes")
	flag.Parse()

	// Initialize logger
	stdLogger := logger.New("file-transfer")
	ctx := context.Background()

	// Create allowed content types
	allowTypes := []string{
		"text/",
		"image/",
		"application/json",
		"application/pdf",
	}

	// Create file server
	fileServer := NewFileServer(stdLogger, *uploadDir, *maxSize, allowTypes)

	// Create metrics registry
	registry := prometheus.NewRegistry()

	// Create server mux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/files", fileServer.ListFiles)
	mux.HandleFunc("/upload", fileServer.UploadFile)
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fileServer.DownloadFile(w, r)
		case http.MethodDelete:
			fileServer.DeleteFile(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Add metrics middleware
	handler := middleware.MetricsMiddleware(middleware.MetricsConfig{
		Registry:     registry,
		Subsystem:    "filetransfer",
		ExcludePaths: []string{"/metrics"},
	})(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
	go func() {
		stdLogger.Info(ctx, "file", "server", fmt.Sprintf("Starting file transfer server on %s", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stdLogger.Error(ctx, "file", "server", fmt.Sprintf("Failed to start server: %v", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		stdLogger.Error(ctx, "file", "server", fmt.Sprintf("Failed to stop server gracefully: %v", err))
	}
}
