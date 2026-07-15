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

// DocValidationRequest is the JSON body for POST /drivers/validate-document.
type DocValidationRequest struct {
	DocType       string `json:"docType"`
	File          string `json:"file"` // base64, may include "data:<mime>;base64," prefix
	MimeType      string `json:"mimeType"`
	ExpectedName  string `json:"expectedName"`
	ExpectedPlate string `json:"expectedPlate"`
}

// DocValidationResponse is the JSON response for POST /drivers/validate-document.
// HTTP 200 is returned even when Valid=false — only system errors use 5xx.
type DocValidationResponse struct {
	Valid          bool   `json:"valid"`
	ExtractedPlate string `json:"extractedPlate"`
	ErrorCode      string `json:"errorCode"`
	ErrorMessage   string `json:"errorMessage"`
}

// ValidateDocument validates an uploaded driver document using Google Cloud Vision.
func ValidateDocument(svc *vision.VisionService) http.HandlerFunc {
	validDocTypes := map[string]bool{
		"driverLicense": true, "tlcLicense": true, "carRegistration": true,
		"vehicleInspection": true, "tlcDiamond": true, "insuranceFiles": true,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 15<<20) // 15 MB

		var req DocValidationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				http.Error(w, "File too large (max 15MB)", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if !validDocTypes[req.DocType] {
			http.Error(w, "Invalid docType", http.StatusBadRequest)
			return
		}
		if req.File == "" {
			http.Error(w, "file is required", http.StatusBadRequest)
			return
		}

		// Strip data URL prefix if present ("data:image/jpeg;base64,...")
		encoded := req.File
		if idx := strings.Index(encoded, ","); idx != -1 {
			encoded = encoded[idx+1:]
		}

		content, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			http.Error(w, "Invalid file encoding", http.StatusBadRequest)
			return
		}

		result, err := svc.ValidateDocument(r.Context(), vision.DocumentValidationRequest{
			DocType:       req.DocType,
			Content:       content,
			MimeType:      req.MimeType,
			ExpectedName:  req.ExpectedName,
			ExpectedPlate: req.ExpectedPlate,
		})
		if err != nil {
			slog.Error("Document validation service error", "docType", req.DocType, "error", err)
			http.Error(w, "Document validation service unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DocValidationResponse{
			Valid:          result.Valid,
			ExtractedPlate: result.ExtractedPlate,
			ErrorCode:      result.ErrorCode,
			ErrorMessage:   result.ErrorMessage,
		})
	}
}
