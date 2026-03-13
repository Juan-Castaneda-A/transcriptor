package models

import "time"

// User represents a registered user.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	AvatarURL    string    `json:"avatar_url"`
	PlanType     string    `json:"plan_type"`     // free, pro, enterprise
	MinutesUsed  float64   `json:"minutes_used"`  // total minutes transcribed
	MinutesLimit float64   `json:"minutes_limit"` // monthly limit based on plan
	CreatedAt    time.Time `json:"created_at"`
}

// MediaFile represents an uploaded audio/video file.
type MediaFile struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	StorageURL string    `json:"storage_url"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`   // bytes
	MimeType   string    `json:"mime_type"`   // audio/mpeg, audio/wav, etc.
	Duration   float64   `json:"duration"`    // seconds
	Status     string    `json:"status"`      // pending, processing, completed, error
	ErrorMsg   string    `json:"error_msg"`   // populated when status=error
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Status constants for MediaFile.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusError      = "error"
)

// Transcription represents the full transcription result for a media file.
type Transcription struct {
	ID          string    `json:"id"`
	MediaFileID string    `json:"media_file_id"`
	FullText    string    `json:"full_text"`
	Language    string    `json:"language"`    // detected language code (es, en, etc.)
	WordCount   int       `json:"word_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Segment represents a timed chunk of transcribed speech.
// This enables the synchronized editor feature.
type Segment struct {
	ID              string  `json:"id"`
	TranscriptionID string  `json:"transcription_id"`
	StartTime       float64 `json:"start_time"`    // seconds
	EndTime         float64 `json:"end_time"`      // seconds
	SpeakerLabel    string  `json:"speaker_label"` // "Speaker 1", "Speaker 2", etc.
	Content         string  `json:"content"`       // the transcribed text for this segment
}

// QueueMessage is the payload sent to Redis for the worker to process.
type QueueMessage struct {
	FileID     string `json:"file_id"`
	StorageURL string `json:"storage_url"`
	UserID     string `json:"user_id"`
	FileName   string `json:"file_name"`
}

// UploadResponse is returned when a file upload is initiated.
type UploadResponse struct {
	FileID    string `json:"file_id"`
	UploadURL string `json:"upload_url"` // Supabase signed URL
	Message   string `json:"message"`
}

// TranscriptionResponse is the full response for a transcription request.
type TranscriptionResponse struct {
	Transcription *Transcription `json:"transcription"`
	Segments      []Segment      `json:"segments"`
	MediaFile     *MediaFile     `json:"media_file"`
}

// ProjectListResponse is returned for the dashboard.
type ProjectListResponse struct {
	Files      []MediaFile `json:"files"`
	TotalCount int         `json:"total_count"`
}

// ExportResponse holds the exported transcription content.
type ExportResponse struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
	Format   string `json:"format"` // txt
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
