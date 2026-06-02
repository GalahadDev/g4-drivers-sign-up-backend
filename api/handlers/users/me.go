package users

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"g4-services/api/database"
	"g4-services/api/domains"
	"g4-services/api/middleware"

	"github.com/google/uuid"
)

type UserProfileResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	ReferralCode string    `json:"referral_code"`
	ReferredBy   *string   `json:"referred_by,omitempty"`
	AvatarURL    *string   `json:"avatar_url"`

	Application *DriverApplicationFull `json:"application"`
}

type DriverApplicationFull struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	FullName        string    `json:"full_name"`
	Address         string    `json:"address"`
	PhoneNumber     string    `json:"phone_number"`
	EmergencyNumber string    `json:"emergency_number"`
	DeviceType      string    `json:"device_type"`
	DriverCategory  string    `json:"driver_category"`

	VehicleType       string `json:"vehicle_type"`
	PassengerCapacity int    `json:"passenger_capacity"`
	VehicleCategory   string `json:"vehicle_category"`

	DriverLicenseURL     string   `json:"driver_license_url"`
	TLCLicenseURL        string   `json:"tlc_license_url"`
	CarRegistrationURL   string   `json:"car_registration_url"`
	VehicleInspectionURL string   `json:"vehicle_inspection_url"`
	TLCDiamondURL        string   `json:"tlc_diamond_url"`
	InsuranceFilesURLs   []string `json:"insurance_files_urls"`
	ProfilePhotoURL      string   `json:"profile_photo_url"`
	VehiclePhotosURLs    []string `json:"vehicle_photos_urls"`

	AdditionalInfo map[string]interface{} `json:"additional_info"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
}

// GetMe godoc
// @Summary      Get My Profile
// @Description  Get current user profile and application status
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200  {object}  UserProfileResponse
// @Failure      401  {string}  string "Unauthorized"
// @Failure      404  {string}  string "User not found"
// @Router       /user/me [get]
// @Security     BearerAuth
func GetMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextKeyUserID).(string)

	response := UserProfileResponse{}
	sql := `
		SELECT 
			-- Tabla Profiles
			p.id, p.email, p.role, p.referral_code, p.referred_by_code, p.avatar_url,
			
			-- Tabla Driver Applications (21 columnas)
			da.id, da.user_id, da.full_name, da.address, da.phone_number, da.emergency_number,
			da.device_type, da.driver_category, da.vehicle_type, da.passenger_capacity,
			da.driver_license_url, da.tlc_license_url, da.car_registration_url,
			da.vehicle_inspection_url, da.tlc_diamond_url, da.insurance_files_urls,
			da.profile_photo_url, da.vehicle_photos_urls, da.additional_info,
			da.status, da.created_at

		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		WHERE p.id = $1`

	var (
		pRefBy             *string
		pAvatar            *string
		appID              *uuid.UUID
		appUserID          *uuid.UUID
		appFullName        *string
		appAddress         *string
		appPhone           *string
		appEmergency       *string
		appDevice          *string
		appCategory        *string
		appVehicleType     *string
		appCapacity        *int
		appLicenseURL      *string
		appTLCURL          *string
		appCarRegURL       *string
		appInspectURL      *string
		appDiamondURL      *string
		appInsuranceURLs   []string
		appProfilePhotoURL *string
		appVehiclePhotos   []string
		appAdditionalBytes []byte
		appStatus          *string
		appCreatedAt       *time.Time
	)

	err := database.Pool.QueryRow(r.Context(), sql, userID).Scan(
		&response.ID, &response.Email, &response.Role, &response.ReferralCode, &pRefBy, &pAvatar,
		&appID, &appUserID, &appFullName, &appAddress, &appPhone, &appEmergency,
		&appDevice, &appCategory, &appVehicleType, &appCapacity,
		&appLicenseURL, &appTLCURL, &appCarRegURL,
		&appInspectURL, &appDiamondURL, &appInsuranceURLs,
		&appProfilePhotoURL, &appVehiclePhotos, &appAdditionalBytes,
		&appStatus, &appCreatedAt,
	)

	if err != nil {
		slog.Error("Error fetching /me", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response.ReferredBy = pRefBy

	response.AvatarURL = pAvatar

	if appID != nil {
		appData := &DriverApplicationFull{
			ID:                   *appID,
			UserID:               *appUserID,
			FullName:             *appFullName,
			Address:              *appAddress,
			PhoneNumber:          *appPhone,
			EmergencyNumber:      *appEmergency,
			DeviceType:           *appDevice,
			DriverCategory:       *appCategory,
			VehicleType:          *appVehicleType,
			PassengerCapacity:    *appCapacity,
			VehicleCategory:      *appCategory,
			DriverLicenseURL:     domains.GetString(appLicenseURL),
			TLCLicenseURL:        domains.GetString(appTLCURL),
			CarRegistrationURL:   domains.GetString(appCarRegURL),
			VehicleInspectionURL: domains.GetString(appInspectURL),
			TLCDiamondURL:        domains.GetString(appDiamondURL),
			ProfilePhotoURL:      domains.GetString(appProfilePhotoURL),

			InsuranceFilesURLs: appInsuranceURLs,
			VehiclePhotosURLs:  appVehiclePhotos,

			Status:    *appStatus,
			CreatedAt: *appCreatedAt,
		}

		if len(appAdditionalBytes) > 0 {
			_ = json.Unmarshal(appAdditionalBytes, &appData.AdditionalInfo)
		} else {
			appData.AdditionalInfo = make(map[string]interface{})
		}

		response.Application = appData
	} else {
		response.Application = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

