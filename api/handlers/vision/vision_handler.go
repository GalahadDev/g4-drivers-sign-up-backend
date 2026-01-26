package vision

import (
	"encoding/base64"
	"encoding/json"
	"g4-services/api/services/vision"
	"log/slog"
	"net/http"
	"strings"
)

type ValidationRequest struct {
	ImageBase64 string `json:"image"`
}

type ValidationResponse struct {
	IsFormal bool     `json:"is_formal"`
	Labels   []string `json:"labels"`
}

func ValidatePhoto(svc *vision.VisionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2MB

		var req ValidationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				http.Error(w, "Image too large (Max 2MB)", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.ImageBase64 == "" {
			http.Error(w, "Image is required", http.StatusBadRequest)
			return
		}

		encodedData := req.ImageBase64
		if idx := strings.Index(encodedData, ","); idx != -1 {
			encodedData = encodedData[idx+1:]
		}

		data, err := base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			slog.Error("Failed to decode base64 image", "error", err)
			http.Error(w, "Invalid image encoding", http.StatusBadRequest)
			return
		}

		isFormal, labels, err := svc.ValidateFormalWear(r.Context(), data)
		if err != nil {
			slog.Error("Vision API analysis failed", "error", err)
			http.Error(w, "Failed to analyze image", http.StatusInternalServerError)
			return
		}

		resp := ValidationResponse{
			IsFormal: isFormal,
			Labels:   labels,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
