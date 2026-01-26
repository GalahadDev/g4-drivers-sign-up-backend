package storage

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"g4-services/api/config"

	"github.com/google/uuid"
)

func ValidateFileContentType(file multipart.File) error {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	contentType := http.DetectContentType(buffer[:n])

	allowedTypes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"application/pdf": true,
	}

	if !allowedTypes[contentType] {
		return fmt.Errorf("invalid file type: %s. Only JPG, PNG, WEBP and PDF are allowed", contentType)
	}

	return nil
}

func UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folder string, cfg *config.AppConfig) (string, error) {

	if err := ValidateFileContentType(file); err != nil {
		return "", err
	}

	ext := filepath.Ext(fileHeader.Filename)
	uniqueName := fmt.Sprintf("%s/%s_%s", folder, uuid.New().String(), ext) // ej: licenses/uuid.jpg

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/storage/v1/object/driver-documents/%s", cfg.SupabaseURL, uniqueName)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+cfg.ServiceRoleKey)
	req.Header.Set("Content-Type", fileHeader.Header.Get("Content-Type"))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase storage error: %s", string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/driver-documents/%s", cfg.SupabaseURL, uniqueName)
	return publicURL, nil
}
