package dashboard

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"g4-services/api/database"
	"g4-services/api/domains"

	"github.com/jackc/pgx/v5"
)

type UserDashboard struct {
	Profile       ProfileInfo                `json:"profile"`
	ReferralStats ReferralStats              `json:"referral_stats"`
	ReferralList  []ReferralItem             `json:"referral_list"`
	Application   *domains.DriverApplication `json:"application,omitempty"`
}

type ProfileInfo struct {
	Email        string `json:"email"`
	ReferralCode string `json:"referral_code"`
	AvatarURL    string `json:"avatar_url"`
	Status       string `json:"status"`
	FullName     string `json:"full_name,omitempty"`
}

type ReferralStats struct {
	TotalReferred int `json:"total_referred"`
	TotalPages    int `json:"total_pages"`
	CurrentPage   int `json:"current_page"`
	ItemsPerPage  int `json:"items_per_page"`
}

type ReferralItem struct {
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
	JoinedAt  string `json:"joined_at"`
	Status    string `json:"status"`
}

// GetMyDashboard godoc
// @Summary      Get My Dashboard
// @Description  Get user dashboard including referral stats and referal list
// @Tags         dashboard
// @Accept       json
// @Produce      json
// @Param        page  query     int     false  "Page number"
// @Param        limit query     int     false  "Items per page"
// @Success      200   {object}  UserDashboard
// @Failure      500   {string}  string "Server Error"
// @Router       /user/dashboard [get]
// @Security     BearerAuth
func GetMyDashboard(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	response := UserDashboard{}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sqlProfile := `
		SELECT 
			p.email, p.referral_code, COALESCE(p.avatar_url, ''),
			COALESCE(da.status, 'not_started'), COALESCE(da.full_name, '')
		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		WHERE p.id = $1`

	err = database.Pool.QueryRow(r.Context(), sqlProfile, userID).Scan(
		&response.Profile.Email, &response.Profile.ReferralCode,
		&response.Profile.AvatarURL, &response.Profile.Status, &response.Profile.FullName,
	)
	if err != nil {
		slog.Error("Error getting dashboard profile", "error", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	// 1.5 Fetch Full Application if exists
	var app domains.DriverApplication
	var additionalInfoBytes []byte
	sqlApp := `
		SELECT id, user_id, full_name, address, phone_number, emergency_number,
		       device_type, driver_category, vehicle_type, passenger_capacity,
			   driver_license_url, tlc_license_url, car_registration_url,
			   vehicle_inspection_url, tlc_diamond_url, insurance_files_urls,
			   profile_photo_url, vehicle_photos_urls, additional_info,
			   status, created_at
		FROM driver_applications
		WHERE user_id = $1`

	err = database.Pool.QueryRow(r.Context(), sqlApp, userID).Scan(
		&app.ID, &app.UserID, &app.FullName, &app.Address, &app.PhoneNumber, &app.EmergencyNumber,
		&app.DeviceType, &app.DriverCategory, &app.VehicleType, &app.PassengerCapacity,
		&app.DriverLicenseURL, &app.TLCLicenseURL, &app.CarRegistrationURL,
		&app.VehicleInspectionURL, &app.TLCDiamondURL, &app.InsuranceFilesURLs,
		&app.ProfilePhotoURL, &app.VehiclePhotosURLs, &additionalInfoBytes,
		&app.Status, &app.CreatedAt,
	)

	if err == nil {
		if len(additionalInfoBytes) > 0 {
			_ = json.Unmarshal(additionalInfoBytes, &app.AdditionalInfo)
		}
		response.Application = &app
	} else if err != pgx.ErrNoRows {
		slog.Error("Error fetching application details", "error", err)
	}

	var totalReferred int
	err = database.Pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM profiles WHERE referred_by_code = $1", response.Profile.ReferralCode).Scan(&totalReferred)
	if err != nil {
		slog.Error("Error counting referrals", "error", err)
		totalReferred = 0
	}

	// Luego traemos la lista paginada
	offset := (page - 1) * limit
	sqlReferrals := `
		SELECT 
			p.email, 
			p.created_at, 
			COALESCE(da.status, 'registered'),
			COALESCE(da.full_name, 'Usuario'), 
			COALESCE(p.avatar_url, '')
		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		WHERE p.referred_by_code = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	var rows pgx.Rows
	rows, err = database.Pool.Query(r.Context(), sqlReferrals, response.Profile.ReferralCode, limit, offset)
	if err != nil {
		slog.Error("Error getting referrals", "error", err)
		response.ReferralList = []ReferralItem{}
	} else {
		defer rows.Close()
		response.ReferralList = []ReferralItem{}
		for rows.Next() {
			var item ReferralItem
			if err := rows.Scan(&item.Email, &item.JoinedAt, &item.Status, &item.FullName, &item.AvatarURL); err == nil {
				response.ReferralList = append(response.ReferralList, item)
			}
		}
	}

	response.ReferralStats.TotalReferred = totalReferred
	response.ReferralStats.CurrentPage = page
	response.ReferralStats.ItemsPerPage = limit
	if totalReferred > 0 {
		response.ReferralStats.TotalPages = (totalReferred + limit - 1) / limit
	} else {
		response.ReferralStats.TotalPages = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
