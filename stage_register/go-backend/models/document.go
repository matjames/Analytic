package models

import "time"

type Document struct {
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	Category        string    `json:"category"`
	Filename        string    `json:"filename"`
	OriginalFilename string   `json:"original_filename" db:"original_filename"`
	FilePath        string    `json:"file_path" db:"file_path"`
	FileSize        *int64    `json:"file_size" db:"file_size"`
	MimeType        *string   `json:"mime_type" db:"mime_type"`
	UploadedBy      int64     `json:"uploaded_by" db:"uploaded_by"`
	UploadedByName  *string   `json:"uploaded_by_name" db:"uploaded_by_name"`
	CreatedAt       time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updatedAt"`
}

