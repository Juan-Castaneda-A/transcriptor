package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juan-Castaneda-A/transcriptor/internal/models"
	"github.com/Juan-Castaneda-A/transcriptor/internal/queue"
	"github.com/Juan-Castaneda-A/transcriptor/internal/storage"
)

// allowedMimeTypes defines which audio formats are accepted.
var allowedMimeTypes = map[string]bool{
	"audio/mpeg": true, // MP3
	"audio/wav":  true, // WAV
	"audio/mp4":  true, // M4A
	"audio/ogg":  true, // OGG
	"audio/flac": true, // FLAC
	"audio/webm": true, // WebM audio
}

// maxFileSize is 100MB for MVP.
const maxFileSize = 100 * 1024 * 1024

// Handler holds dependencies for API handlers.
type Handler struct {
	storage *storage.Client
	queue   *queue.Client
}

// NewHandler creates a new API handler with dependencies.
func NewHandler(storageClient *storage.Client, queueClient *queue.Client) *Handler {
	return &Handler{
		storage: storageClient,
		queue:   queueClient,
	}
}

// HealthCheck returns a simple health status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "0.1.0-alpha",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// UploadFile handles the file upload initiation.
// POST /api/upload
// Body: { "file_name": "interview.mp3", "file_size": 5242880, "mime_type": "audio/mpeg" }
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)

	var req struct {
		FileName string `json:"file_name"`
		FileSize int64  `json:"file_size"`
		MimeType string `json:"mime_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Could not parse request body",
		})
		return
	}

	// Validate file type
	if !allowedMimeTypes[req.MimeType] {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_file_type",
			Message: fmt.Sprintf("File type '%s' is not supported. Accepted: MP3, WAV, M4A, OGG, FLAC", req.MimeType),
		})
		return
	}

	// Validate file size
	if req.FileSize > maxFileSize {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error:   "file_too_large",
			Message: "Maximum file size is 100MB",
		})
		return
	}

	// Generate unique file path in storage
	timestamp := time.Now().UnixMilli()
	ext := filepath.Ext(req.FileName)
	storagePath := fmt.Sprintf("%s/%d%s", userID, timestamp, ext)

	// Create signed upload URL
	signedURL, err := h.storage.CreateSignedUploadURL("media", storagePath)
	if err != nil {
		log.Printf("Failed to create signed URL: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "storage_error",
			Message: "Failed to prepare upload. Please try again.",
		})
		return
	}

	// Create media file record in DB
	fileID := fmt.Sprintf("mf_%d", timestamp)
	mediaFile := map[string]interface{}{
		"id":          fileID,
		"user_id":     userID,
		"storage_url": storagePath,
		"file_name":   req.FileName,
		"file_size":   req.FileSize,
		"mime_type":   req.MimeType,
		"status":      models.StatusPending,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	}

	if _, err := h.storage.InsertRecord("media_files", mediaFile); err != nil {
		log.Printf("Failed to insert media file record: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to register file. Please try again.",
		})
		return
	}

	log.Printf("📁 File registered: id=%s name=%s user=%s", fileID, req.FileName, userID)

	writeJSON(w, http.StatusOK, models.UploadResponse{
		FileID:    fileID,
		UploadURL: signedURL,
		Message:   "Upload URL generated. Upload the file, then call /api/upload/confirm.",
	})
}

// ConfirmUpload is called after the frontend uploads the file to Supabase Storage.
// POST /api/upload/confirm
// Body: { "file_id": "mf_123456" }
func (h *Handler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)

	var req struct {
		FileID string `json:"file_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Could not parse request body",
		})
		return
	}

	// Get the file record to get storage URL
	records, err := h.storage.QueryRecords("media_files",
		fmt.Sprintf("id=eq.%s&user_id=eq.%s&select=*", req.FileID, userID), "")
	if err != nil {
		log.Printf("Failed to query media file: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to find file record.",
		})
		return
	}

	var files []models.MediaFile
	if err := json.Unmarshal(records, &files); err != nil || len(files) == 0 {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "File not found.",
		})
		return
	}

	file := files[0]

	// Enqueue the transcription job
	msg := models.QueueMessage{
		FileID:     file.ID,
		StorageURL: file.StorageURL,
		UserID:     userID,
		FileName:   file.FileName,
	}

	if err := h.queue.Enqueue(r.Context(), msg); err != nil {
		log.Printf("Failed to enqueue job: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "queue_error",
			Message: "Failed to start transcription. Please try again.",
		})
		return
	}

	// Update status to processing
	h.storage.UpdateRecord("media_files",
		fmt.Sprintf("id=eq.%s", file.ID),
		map[string]interface{}{
			"status":     models.StatusProcessing,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})

	log.Printf("🚀 Transcription enqueued: file=%s", file.ID)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Transcription started! You'll be notified when it's ready.",
		"status":  models.StatusProcessing,
	})
}

// GetProjects returns the user's media files for the dashboard.
// GET /api/projects
func (h *Handler) GetProjects(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	records, err := h.storage.QueryRecords("media_files",
		fmt.Sprintf("user_id=eq.%s&order=created_at.desc&select=*", userID), token)
	if err != nil {
		log.Printf("Failed to query projects: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load projects.",
		})
		return
	}

	var files []models.MediaFile
	json.Unmarshal(records, &files)
	if files == nil {
		files = []models.MediaFile{}
	}

	writeJSON(w, http.StatusOK, models.ProjectListResponse{
		Files:      files,
		TotalCount: len(files),
	})
}

// GetTranscription returns the full transcription with segments.
// GET /api/transcriptions/{id}
func (h *Handler) GetTranscription(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	// Use Go 1.22 path value
	fileID := r.PathValue("id")
	if fileID == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing_id",
			Message: "ID is required",
		})
		return
	}

	// Check if this is an export request
	isExport := strings.HasSuffix(r.URL.Path, "/export")

	// Get the media file
	fileRecords, err := h.storage.QueryRecords("media_files",
		fmt.Sprintf("id=eq.%s&user_id=eq.%s&select=*", fileID, userID), token)
	if err != nil {
		log.Printf("❌ Error querying media_files: %v", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load file.",
		})
		return
	}

	var files []models.MediaFile
	json.Unmarshal(fileRecords, &files)
	if len(files) == 0 {
		log.Printf("⚠️ File not found or RLS restricted: id=%s user=%s", fileID, userID)
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{
			Error:   "not_found",
			Message: "File not found.",
		})
		return
	}

	// Get the transcription
	transRecords, err := h.storage.QueryRecords("transcriptions",
		fmt.Sprintf("media_file_id=eq.%s&select=*", fileID), token)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to load transcription.",
		})
		return
	}

	var transcriptions []models.Transcription
	json.Unmarshal(transRecords, &transcriptions)
	if len(transcriptions) == 0 {
		log.Printf("⏳ Transcription not ready for file: %s", fileID)
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{
			Error:   "not_ready",
			Message: "Transcription is not ready yet.",
		})
		return
	}

	transcription := transcriptions[0]

	// If export, return the text directly
	if isExport {
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "txt"
		}

		baseName := strings.TrimSuffix(files[0].FileName, filepath.Ext(files[0].FileName))

		writeJSON(w, http.StatusOK, models.ExportResponse{
			FileName: fmt.Sprintf("%s.%s", baseName, format),
			Content:  transcription.FullText,
			Format:   format,
		})
		return
	}

	// Get segments
	segRecords, err := h.storage.QueryRecords("segments",
		fmt.Sprintf("transcription_id=eq.%s&order=start_time.asc&select=*", transcription.ID), token)
	if err != nil {
		log.Printf("Failed to query segments: %v", err)
	}

	var segments []models.Segment
	json.Unmarshal(segRecords, &segments)
	if segments == nil {
		segments = []models.Segment{}
	}

	writeJSON(w, http.StatusOK, models.TranscriptionResponse{
		Transcription: &transcription,
		Segments:      segments,
		MediaFile:     &files[0],
	})
}
