package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go-backend/configs"
	"go-backend/models"

	"github.com/gin-gonic/gin"
)

const (
	uploadDir       = "uploads/public-documents"
	maxFileSize     = 50 * 1024 * 1024 // 50MB
	allowedMimeType = "application/pdf"
)

var validCategories = []string{"SOP", "Manual", "Training", "Other"}

func init() {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create upload directory: %v\n", err)
	}
}

func isValidCategory(category string) bool {
	for _, valid := range validCategories {
		if category == valid {
			return true
		}
	}
	return false
}

// ListDocuments - GET /documents - List all public documents (public endpoint)
func ListDocuments(c *gin.Context) {
	category := c.Query("category")

	var rows *sql.Rows
	var err error

	if category != "" {
		rows, err = configs.DB.Query(
			`SELECT pd.*, u.username as uploaded_by_name 
			 FROM public_documents pd 
			 LEFT JOIN users u ON pd.uploaded_by = u.id 
			 WHERE pd.category = $1 
			 ORDER BY pd."createdAt" DESC`,
			category,
		)
	} else {
		rows, err = configs.DB.Query(
			`SELECT pd.*, u.username as uploaded_by_name 
			 FROM public_documents pd 
			 LEFT JOIN users u ON pd.uploaded_by = u.id 
			 ORDER BY pd."createdAt" DESC`,
		)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var documents []models.Document
	for rows.Next() {
		var doc models.Document
		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.Description, &doc.Category,
			&doc.Filename, &doc.OriginalFilename, &doc.FilePath,
			&doc.FileSize, &doc.MimeType, &doc.UploadedBy,
			&doc.CreatedAt, &doc.UpdatedAt, &doc.UploadedByName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, documents)
}

// GetDocument - GET /documents/:id - Get a single document
func GetDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var doc models.Document
	err = configs.DB.QueryRow(
		`SELECT pd.*, u.username as uploaded_by_name 
		 FROM public_documents pd 
		 LEFT JOIN users u ON pd.uploaded_by = u.id 
		 WHERE pd.id = $1`,
		id,
	).Scan(
		&doc.ID, &doc.Title, &doc.Description, &doc.Category,
		&doc.Filename, &doc.OriginalFilename, &doc.FilePath,
		&doc.FileSize, &doc.MimeType, &doc.UploadedBy,
		&doc.CreatedAt, &doc.UpdatedAt, &doc.UploadedByName,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// UploadDocument - POST /documents - Upload a new document (requires auth)
func UploadDocument(c *gin.Context) {
	userID := c.GetInt64("user_id")

	// Get the file from form
	file, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PDF file is required"})
		return
	}

	// Validate file type
	if file.Header.Get("Content-Type") != allowedMimeType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	// Validate file size
	if file.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds 50MB limit"})
		return
	}

	// Get form fields
	title := c.PostForm("title")
	category := c.PostForm("category")
	description := c.PostForm("description")

	if title == "" || category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and category are required"})
		return
	}

	if !isValidCategory(category) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category. Must be one of: SOP, Manual, Training, Other"})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	uniqueFilename := fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), os.Getpid(), ext)
	filePath := filepath.Join(uploadDir, uniqueFilename)

	// Save file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Insert into database
	var doc models.Document
	fileSize := int64(file.Size)
	mimeType := file.Header.Get("Content-Type")

	var desc *string
	if description != "" {
		desc = &description
	}

	err = configs.DB.QueryRow(
		`INSERT INTO public_documents 
		 (title, description, category, filename, original_filename, file_path, file_size, mime_type, uploaded_by, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		 RETURNING id, title, description, category, filename, original_filename, file_path, file_size, mime_type, uploaded_by, "createdAt", "updatedAt"`,
		title, desc, category, uniqueFilename, file.Filename, filePath, fileSize, mimeType, userID,
	).Scan(
		&doc.ID, &doc.Title, &doc.Description, &doc.Category,
		&doc.Filename, &doc.OriginalFilename, &doc.FilePath,
		&doc.FileSize, &doc.MimeType, &doc.UploadedBy,
		&doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fetch with uploaded_by_name
	var username sql.NullString
	err = configs.DB.QueryRow(
		`SELECT u.username 
		 FROM users u 
		 WHERE u.id = $1`,
		userID,
	).Scan(&username)

	if err == nil && username.Valid {
		doc.UploadedByName = &username.String
	}

	c.JSON(http.StatusCreated, doc)
}

// UpdateDocument - PUT /documents/:id - Update document metadata (requires auth)
func UpdateDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Category    *string `json:"category"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if document exists
	var existingID int64
	err = configs.DB.QueryRow("SELECT id FROM public_documents WHERE id = $1", id).Scan(&existingID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Validate category if provided
	if req.Category != nil && !isValidCategory(*req.Category) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category. Must be one of: SOP, Manual, Training, Other"})
		return
	}

	// Update document
	var doc models.Document
	err = configs.DB.QueryRow(
		`UPDATE public_documents 
		 SET title = COALESCE($1, title),
		     description = COALESCE($2, description),
		     category = COALESCE($3, category),
		     "updatedAt" = NOW()
		 WHERE id = $4
		 RETURNING id, title, description, category, filename, original_filename, file_path, file_size, mime_type, uploaded_by, "createdAt", "updatedAt"`,
		req.Title, req.Description, req.Category, id,
	).Scan(
		&doc.ID, &doc.Title, &doc.Description, &doc.Category,
		&doc.Filename, &doc.OriginalFilename, &doc.FilePath,
		&doc.FileSize, &doc.MimeType, &doc.UploadedBy,
		&doc.CreatedAt, &doc.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fetch uploaded_by_name
	var username sql.NullString
	err = configs.DB.QueryRow(
		`SELECT u.username 
		 FROM users u 
		 WHERE u.id = $1`,
		doc.UploadedBy,
	).Scan(&username)

	if err == nil && username.Valid {
		doc.UploadedByName = &username.String
	}

	c.JSON(http.StatusOK, doc)
}

// DeleteDocument - DELETE /documents/:id - Delete a document (requires auth)
func DeleteDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Get file path before deleting
	var filePath string
	err = configs.DB.QueryRow("SELECT file_path FROM public_documents WHERE id = $1", id).Scan(&filePath)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete from database
	result, err := configs.DB.Exec("DELETE FROM public_documents WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Delete file from filesystem
	if _, err := os.Stat(filePath); err == nil {
		os.Remove(filePath)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DownloadDocument - GET /documents/:id/download - Download a document (public endpoint)
func DownloadDocument(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var doc models.Document
	err = configs.DB.QueryRow(
		`SELECT id, original_filename, file_path, mime_type 
		 FROM public_documents 
		 WHERE id = $1`,
		id,
	).Scan(&doc.ID, &doc.OriginalFilename, &doc.FilePath, &doc.MimeType)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if file exists
	if _, err := os.Stat(doc.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found on server"})
		return
	}

	// Open file
	file, err := os.Open(doc.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Get file info for Content-Length
	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file info"})
		return
	}

	// Set headers
	mimeType := "application/pdf"
	if doc.MimeType != nil {
		mimeType = *doc.MimeType
	}
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, doc.OriginalFilename))
	c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	// Stream file
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		// Error writing response, but headers already sent
		return
	}
}
