package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"g4-services/api/database"

	"github.com/google/uuid"
)

type AdminUserDetailResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	ReferralCode string    `json:"referral_code"`
	Referrer     *string   `json:"referred_by,omitempty"`
	AvatarURL    *string   `json:"avatar_url"`

	Application *DriverApplicationDetail `json:"application"`
}

type DriverApplicationDetail struct {
	ID                   uuid.UUID `json:"id"`
	UserID               uuid.UUID `json:"user_id"`
	FullName             string    `json:"full_name"`
	Address              string    `json:"address"`
	PhoneNumber          string    `json:"phone_number"`
	EmergencyNumber      string    `json:"emergency_number"`
	DeviceType           string    `json:"device_type"`
	DriverCategory       string    `json:"driver_category"`
	VehicleType          string    `json:"vehicle_type"`
	PassengerCapacity    int       `json:"passenger_capacity"`
	DriverLicenseURL     string    `json:"driver_license_url"`
	TLCLicenseURL        string    `json:"tlc_license_url"`
	CarRegistrationURL   string    `json:"car_registration_url"`
	VehicleInspectionURL string    `json:"vehicle_inspection_url"`
	TLCDiamondURL        string    `json:"tlc_diamond_url"`
	InsuranceFilesURLs   []string  `json:"insurance_files_urls"`
	ProfilePhotoURL      string    `json:"profile_photo_url"`
	VehiclePhotosURLs    []string  `json:"vehicle_photos_urls"`

	AdditionalInfo map[string]interface{} `json:"additional_info"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
}

// GetUserDetail godoc
// @Summary      Get User Detail (Admin)
// @Description  Get full details of a specific user including profile and application data
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        id    query     string  true  "User UUID"
// @Success      200   {object}  AdminUserDetailResponse
// @Failure      400   {string}  string "Missing id parameter"
// @Failure      404   {string}  string "User not found"
// @Router       /admin/user [get]
// @Security     BearerAuth
func GetUserDetail(w http.ResponseWriter, r *http.Request) {
	targetUserID := r.URL.Query().Get("id")
	if targetUserID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	response := AdminUserDetailResponse{}

	sql := `
		SELECT 
			p.id, p.email, p.role, p.referral_code, p.referred_by_code, p.avatar_url,
			da.id, da.full_name, da.address, da.phone_number, da.emergency_number,
			da.device_type, da.driver_category, da.vehicle_type, da.passenger_capacity,
			da.driver_license_url, da.tlc_license_url, da.car_registration_url,
			da.vehicle_inspection_url, da.tlc_diamond_url, da.insurance_files_urls,
			da.profile_photo_url, da.vehicle_photos_urls, da.additional_info,
			da.status, da.created_at
		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		WHERE p.id = $1`

	var (
		pRefBy  *string
		pAvatar *string

		appID           *uuid.UUID
		appFullName     *string
		appAddress      *string
		appPhone        *string
		appEmergency    *string
		appDevice       *string
		appCategory     *string
		appVehicle      *string
		appCapacity     *int
		appLicense      *string
		appTLC          *string
		appCarReg       *string
		appInspect      *string
		appDiamond      *string
		appInsurances   []string
		appProfilePhoto *string
		appVehPhotos    []string
		appAddInfo      []byte
		appStatus       *string
		appCreated      *time.Time
	)

	err := database.Pool.QueryRow(r.Context(), sql, targetUserID).Scan(
		&response.ID, &response.Email, &response.Role, &response.ReferralCode, &pRefBy, &pAvatar,
		&appID, &appFullName, &appAddress, &appPhone, &appEmergency,
		&appDevice, &appCategory, &appVehicle, &appCapacity,
		&appLicense, &appTLC, &appCarReg,
		&appInspect, &appDiamond, &appInsurances,
		&appProfilePhoto, &appVehPhotos, &appAddInfo,
		&appStatus, &appCreated,
	)

	if err != nil {
		slog.Error("Admin GetUserDetail failed", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response.Referrer = pRefBy
	response.AvatarURL = pAvatar

	if appID != nil {
		app := &DriverApplicationDetail{
			ID:                *appID,
			UserID:            response.ID,
			FullName:          getString(appFullName),
			Address:           getString(appAddress),
			PhoneNumber:       getString(appPhone),
			EmergencyNumber:   getString(appEmergency),
			DeviceType:        getString(appDevice),
			DriverCategory:    getString(appCategory),
			VehicleType:       getString(appVehicle),
			PassengerCapacity: getInt(appCapacity),

			DriverLicenseURL:     getString(appLicense),
			TLCLicenseURL:        getString(appTLC),
			CarRegistrationURL:   getString(appCarReg),
			VehicleInspectionURL: getString(appInspect),
			TLCDiamondURL:        getString(appDiamond),
			ProfilePhotoURL:      getString(appProfilePhoto),

			InsuranceFilesURLs: appInsurances,
			VehiclePhotosURLs:  appVehPhotos,

			Status:    *appStatus,
			CreatedAt: *appCreated,
		}

		if len(appAddInfo) > 0 {
			_ = json.Unmarshal(appAddInfo, &app.AdditionalInfo)
		} else {
			app.AdditionalInfo = make(map[string]interface{})
		}
		response.Application = app
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func getInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
