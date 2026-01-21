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

func UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folder string, cfg *config.AppConfig) (string, error) {

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

	client := &http.Client{Timeout: 10 * time.Second}
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
