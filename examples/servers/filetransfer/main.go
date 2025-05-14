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

	"github.com/harriteja/mcp-go-sdk/pkg/server/middleware"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type FileTransferService struct {
	logger    *zap.Logger
	uploadDir string
}

func NewFileTransferService(logger *zap.Logger, uploadDir string) (*FileTransferService, error) {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return &FileTransferService{
		logger:    logger,
		uploadDir: uploadDir,
	}, nil
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error *types.Error `json:"error"`
	}{
		Error: types.NewError(code, message),
	})
}

func (s *FileTransferService) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse multipart form with 32MB max memory
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to get file")
		return
	}
	defer file.Close()

	// Create file with safe name
	filename := filepath.Join(s.uploadDir, header.Filename)
	out, err := os.Create(filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create file")
		return
	}
	defer out.Close()

	// Copy file with progress tracking
	written, err := io.Copy(out, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	s.logger.Info("File uploaded",
		zap.String("filename", header.Filename),
		zap.Int64("size", written),
	)

	w.WriteHeader(http.StatusOK)
}

func (s *FileTransferService) HandleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "Filename not provided")
		return
	}

	// Ensure filename is within upload directory
	filepath := filepath.Join(s.uploadDir, filename)
	if !strings.HasPrefix(filepath, s.uploadDir) {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "File not found")
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to open file")
		}
		return
	}
	defer file.Close()

	// Get file info for headers
	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get file info")
		return
	}

	// Set headers
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	// Stream file
	if _, err := io.Copy(w, file); err != nil {
		s.logger.Error("Failed to stream file",
			zap.String("filename", filename),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("File downloaded",
		zap.String("filename", filename),
		zap.Int64("size", info.Size()),
	)
}

func (s *FileTransferService) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	files, err := os.ReadDir(s.uploadDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list files")
		return
	}

	fileList := make([]map[string]interface{}, 0, len(files))
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		fileList = append(fileList, map[string]interface{}{
			"name":    file.Name(),
			"size":    info.Size(),
			"modTime": info.ModTime(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Result interface{} `json:"result"`
	}{
		Result: fileList,
	})
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Server address")
	uploadDir := flag.String("upload-dir", "./uploads", "Upload directory")
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create file transfer service
	service, err := NewFileTransferService(logger, *uploadDir)
	if err != nil {
		logger.Fatal("Failed to create service", zap.Error(err))
	}

	// Create metrics registry
	registry := prometheus.NewRegistry()

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", service.HandleUpload)
	mux.HandleFunc("/download", service.HandleDownload)
	mux.HandleFunc("/list", service.HandleList)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Add metrics middleware
	handler := middleware.MetricsMiddleware(middleware.MetricsConfig{
		Registry: registry,
	})(mux)

	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  5 * time.Minute, // Longer timeout for file uploads
		WriteTimeout: 5 * time.Minute, // Longer timeout for file downloads
	}

	// Start server
	go func() {
		logger.Info("Starting server", zap.String("addr", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}
}
