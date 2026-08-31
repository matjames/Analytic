package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func uploadPostMediaHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid media upload or file exceeds 16 MB")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	sniffBuffer := make([]byte, 512)
	n, err := io.ReadFull(file, sniffBuffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		writeError(w, http.StatusBadRequest, "failed to inspect upload")
		return
	}
	contentType := http.DetectContentType(sniffBuffer[:n])
	if !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(contentType, "video/") {
		writeError(w, http.StatusBadRequest, "feed media must be an image or video")
		return
	}
	if !isAllowedAttachmentType(contentType) {
		writeError(w, http.StatusBadRequest, "media type is not allowed")
		return
	}

	extension := strings.ToLower(filepath.Ext(header.Filename))
	safeName := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	destination, err := os.Create(filepath.Join(ensureUploadDir(), safeName))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create media upload")
		return
	}
	if _, err = io.Copy(destination, io.MultiReader(bytes.NewReader(sniffBuffer[:n]), file)); err != nil {
		destination.Close()
		writeError(w, http.StatusInternalServerError, "failed to save media upload")
		return
	}
	if err = destination.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to finalize media upload")
		return
	}
	writeJSON(w, map[string]string{
		"url":      "/uploads/" + safeName,
		"fileName": header.Filename,
		"mimeType": contentType,
	})
}
