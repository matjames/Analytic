package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var uploadDir = getEnv("ENTERPRISE_UPLOAD_DIR", "./uploads/enterprise")

func handleListFiles(c *gin.Context) {
	category := c.Query("category")
	project := c.Query("project_id")
	uploader := c.Query("uploaded_by")
	limit := parseIntDefault(c.Query("limit"), 50)
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)

	files := fetchFileRecords(limit)
	filtered := make([]FileRecord, 0, len(files))
	for _, f := range files {
		if !canAccessFileRecord(f, tenantID, platformRole) {
			continue
		}
		if category != "" && f.Category != category {
			continue
		}
		if project != "" && f.ProjectID != project {
			continue
		}
		if uploader != "" && f.UploadedBy != uploader {
			continue
		}
		filtered = append(filtered, f)
		if len(filtered) >= limit {
			break
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "files": filtered})
}

func fetchFileRecords(limit int) []FileRecord {
	if redisClient == nil {
		return []FileRecord{}
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:files", 0, int64(limit-1)).Result()
	files := make([]FileRecord, 0, len(raw))
	for _, item := range raw {
		var f FileRecord
		if err := json.Unmarshal([]byte(item), &f); err == nil {
			files = append(files, f)
		}
	}
	return files
}

func handleGetFile(c *gin.Context) {
	id := c.Param("id")
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)
	files := fetchFileRecords(500)
	for _, f := range files {
		if f.ID == id && canAccessFileRecord(f, tenantID, platformRole) {
			c.JSON(200, f)
			return
		}
	}
	c.JSON(404, gin.H{"error": "file not found"})
}

func handleUploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "file required", "detail": err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to read file"})
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	category := detectFileCategory(mimeType, header.Filename)

	id := fmt.Sprintf("file_%d", time.Now().UnixNano())
	filePath := fmt.Sprintf("%s/%s_%s", uploadDir, id, sanitizeFilename(header.Filename))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(500, gin.H{"error": "failed to create upload directory"})
		return
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		c.JSON(500, gin.H{"error": "failed to save file"})
		return
	}

	rec := FileRecord{
		ID:           id,
		TenantID:     getContextTenantID(c),
		Name:         header.Filename,
		OriginalName: header.Filename,
		Path:         filePath,
		Size:         int64(len(data)),
		MimeType:     mimeType,
		Category:     category,
		UploadedBy:   c.GetHeader("X-User-ID"),
		ProjectID:    c.PostForm("project_id"),
		Version:      1,
		Status:       "scanned",
		PreviewURL:   fmt.Sprintf("/api/files/%s/download", id),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if metaStr := c.PostForm("metadata"); metaStr != "" {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
			rec.Metadata = meta
		}
	}

	if redisClient != nil {
		record, _ := json.Marshal(rec)
		redisClient.RPush(context.Background(), "statgate:files", string(record))
	}

	publishEvent(DomainEvent{
		EventType:  "file.uploaded",
		Source:     "enterprise",
		ObjectType: "file",
		ObjectID:   id,
		Actor:      rec.UploadedBy,
		ProjectID:  rec.ProjectID,
		Payload:    map[string]interface{}{"name": rec.Name, "size": rec.Size, "category": rec.Category},
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	})

	recordAudit("file.upload", "enterprise", rec.UploadedBy, map[string]interface{}{
		"file_id": id, "name": rec.Name, "size": rec.Size, "category": rec.Category,
	})

	c.JSON(201, rec)
}

func handleDownloadFile(c *gin.Context) {
	id := c.Param("id")
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)
	files := fetchFileRecords(500)
	for _, f := range files {
		if f.ID == id && canAccessFileRecord(f, tenantID, platformRole) {
			c.FileAttachment(f.Path, f.Name)
			return
		}
	}
	c.JSON(404, gin.H{"error": "file not found"})
}

func handleUpdateFileMetadata(c *gin.Context) {
	id := c.Param("id")
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)
	files := fetchFileRecords(500)
	for _, f := range files {
		if f.ID == id && canAccessFileRecord(f, tenantID, platformRole) {
			var updates map[string]interface{}
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(400, gin.H{"error": "invalid metadata"})
				return
			}
			if name, ok := updates["name"].(string); ok {
				f.Name = name
			}
			if meta, ok := updates["metadata"].(map[string]interface{}); ok {
				f.Metadata = meta
			}
			if projectID, ok := updates["project_id"].(string); ok {
				f.ProjectID = projectID
			}
			f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			persistFileRecord(id, f)
			c.JSON(200, f)
			return
		}
	}
	c.JSON(404, gin.H{"error": "file not found"})
}

func handleDeleteFile(c *gin.Context) {
	id := c.Param("id")
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)
	if redisClient != nil {
		ctx := context.Background()
		raw, _ := redisClient.LRange(ctx, "statgate:files", 0, -1).Result()
		redisClient.Del(ctx, "statgate:files")
		for _, item := range raw {
			var f FileRecord
			if err := json.Unmarshal([]byte(item), &f); err == nil {
				if f.ID == id && canAccessFileRecord(f, tenantID, platformRole) {
					os.Remove(f.Path)
					continue
				}
				redisClient.RPush(ctx, "statgate:files", item)
			}
		}
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "deleted": true})
}

func handleFileVersions(c *gin.Context) {
	id := c.Param("id")
	tenantID := getContextTenantID(c)
	platformRole := getContextRole(c)
	files := fetchFileRecords(500)
	for _, f := range files {
		if f.ID == id && canAccessFileRecord(f, tenantID, platformRole) {
			versions := []map[string]interface{}{
				{"version": f.Version, "name": f.Name, "size": f.Size, "uploaded_by": f.UploadedBy, "created_at": f.CreatedAt},
			}
			c.JSON(200, gin.H{"file_id": id, "versions": versions})
			return
		}
	}
	c.JSON(404, gin.H{"error": "file not found"})
}

func persistFileRecord(id string, updated FileRecord) {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:files", 0, -1).Result()
	redisClient.Del(ctx, "statgate:files")
	for _, item := range raw {
		var f FileRecord
		if err := json.Unmarshal([]byte(item), &f); err == nil {
			if f.ID == id {
				f = updated
			}
			data, _ := json.Marshal(f)
			redisClient.RPush(ctx, "statgate:files", string(data))
		}
	}
}

func detectFileCategory(mimeType, filename string) string {
	mimeType = strings.ToLower(mimeType)
	name := strings.ToLower(filename)
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.Contains(mimeType, "pdf"), strings.Contains(mimeType, "word"), strings.Contains(mimeType, "text/"), strings.Contains(mimeType, "spreadsheet"), strings.Contains(mimeType, "presentation"):
		return "document"
	case strings.HasSuffix(name, ".csv"), strings.HasSuffix(name, ".tsv"), strings.HasSuffix(name, ".xlsx"), strings.HasSuffix(name, ".xls"), strings.HasSuffix(name, ".parquet"):
		return "dataset"
	case strings.HasSuffix(name, ".zip"), strings.HasSuffix(name, ".tar"), strings.HasSuffix(name, ".gz"), strings.HasSuffix(name, ".rar"), strings.HasSuffix(name, ".7z"):
		return "archive"
	default:
		return "document"
	}
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func canAccessFileRecord(f FileRecord, tenantID, platformRole string) bool {
	if isPlatformAccessRole(platformRole) {
		return true
	}
	return tenantID != "" && f.TenantID == tenantID
}

func isPlatformAccessRole(role string) bool {
	switch role {
	case "admin", "superadmin", "platform_admin":
		return true
	default:
		return false
	}
}
