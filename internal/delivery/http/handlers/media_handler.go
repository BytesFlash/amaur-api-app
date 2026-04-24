package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"amaur/api/internal/delivery/http/response"

	"github.com/google/uuid"
)

const (
	maxUploadSize    = 10 << 20 // 10 MB
	uploadsFormField = "file"
)

var allowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/avif": ".avif",
}

type MediaHandler struct {
	uploadDir   string
	publicBase  string
}

func NewMediaHandler(uploadDir, publicBase string) *MediaHandler {
	return &MediaHandler{
		uploadDir:  uploadDir,
		publicBase: strings.TrimRight(publicBase, "/"),
	}
}

type uploadResponse struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
	MIMEType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

// Upload accepts a multipart/form-data request with a "file" field,
// validates the MIME type, saves the file to disk and returns the public URL.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		response.BadRequest(w, "FILE_TOO_LARGE", fmt.Sprintf("El archivo excede el límite de %d MB", maxUploadSize>>20))
		return
	}

	file, header, err := r.FormFile(uploadsFormField)
	if err != nil {
		response.BadRequest(w, "MISSING_FILE", "Se requiere un archivo en el campo 'file'")
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect real MIME type (ignores Content-Type header).
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		response.InternalError(w)
		return
	}
	mimeType := http.DetectContentType(buf[:n])

	ext, allowed := allowedMIMETypes[mimeType]
	if !allowed {
		response.BadRequest(w, "INVALID_FILE_TYPE", "Solo se permiten imágenes (JPEG, PNG, WebP, GIF, AVIF)")
		return
	}

	// Ensure uploads dir exists.
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		response.InternalError(w)
		return
	}

	// Build a collision-safe file name.
	fileName := fmt.Sprintf("%s-%d%s", uuid.New().String(), time.Now().UnixMilli(), ext)
	dest := filepath.Join(h.uploadDir, fileName)

	// Seek back to start before writing (we already read 512 bytes for detection).
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.InternalError(w)
		return
	}

	out, err := os.Create(dest)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		response.InternalError(w)
		return
	}

	_ = header // original filename intentionally not used to avoid path traversal

	response.Created(w, uploadResponse{
		URL:      fmt.Sprintf("%s/uploads/%s", h.publicBase, fileName),
		FileName: fileName,
		MIMEType: mimeType,
		Size:     written,
	})
}
