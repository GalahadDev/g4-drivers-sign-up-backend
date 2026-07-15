package storage

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"g4-services/api/config"

	"github.com/google/uuid"
)

const storageBucket = "driver-documents"

// storageClient is shared across all upload calls to reuse TCP+TLS connections.
var storageClient = &http.Client{Timeout: 60 * time.Second}

// DeleteFile removes a previously uploaded object given its public URL.
// Used to roll back orphaned uploads when a later step (e.g. the DB insert) fails (audit A1).
func DeleteFile(publicURL string, cfg *config.AppConfig) error {
	marker := "/storage/v1/object/public/" + storageBucket + "/"
	_, objectPath, found := strings.Cut(publicURL, marker)
	if !found {
		return fmt.Errorf("storage: unrecognized public URL: %s", publicURL)
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", cfg.SupabaseURL, storageBucket, objectPath)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceRoleKey)

	resp, err := storageClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("storage delete error (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// CleanupUploads best-effort deletes a set of uploaded URLs, logging failures.
func CleanupUploads(urls []string, cfg *config.AppConfig) {
	for _, u := range urls {
		if u == "" {
			continue
		}
		if err := DeleteFile(u, cfg); err != nil {
			slog.Error("Failed to roll back orphaned upload", "url", u, "error", err)
		}
	}
}

// detectContentType reads the first 512 bytes to detect the real MIME type,
// resets the read pointer, and returns an error if the type is not allowed.
func detectContentType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer[:n])

	allowedTypes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"application/pdf": true,
	}
	if !allowedTypes[contentType] {
		return "", fmt.Errorf("invalid file type: %s. Only JPG, PNG, WEBP and PDF are allowed", contentType)
	}

	return contentType, nil
}

func UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folder string, cfg *config.AppConfig) (string, error) {
	detectedType, err := detectContentType(file)
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(fileHeader.Filename)
	uniqueName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

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
	req.Header.Set("Content-Type", detectedType)

	resp, err := storageClient.Do(req)
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
