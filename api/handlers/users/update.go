package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"g4-services/api/config"
	"g4-services/api/database"
	"g4-services/api/services/email"
)

type UpdateProfileRequest struct {
	Phone        string `json:"phone_number"`
	Address      string `json:"address"`
	AvatarURL    string `json:"avatar_url"`
	ReferralCode string `json:"referral_code"`
}

// UpdateMyProfile godoc
// @Summary      Update My Profile
// @Description  Update user phone, address, avatar or set referral code
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body UpdateProfileRequest true "Update Data"
// @Success      200  {string}  string "{"status":"updated"}"
// @Failure      400  {string}  string "Invalid Body"
// @Failure      409  {string}  string "Conflict (Already referred)"
// @Failure      500  {string}  string "Server Error"
// @Router       /user/profile [put]
// @Security     BearerAuth
func UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Body", http.StatusBadRequest)
		return
	}

	tx, err := database.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if req.Phone != "" || req.Address != "" {
		sqlDriver := `
			UPDATE driver_applications 
			SET phone_number = COALESCE(NULLIF($1, ''), phone_number),
			    address = COALESCE(NULLIF($2, ''), address),
			    updated_at = NOW() 
			WHERE user_id = $3`

		if _, err := tx.Exec(r.Context(), sqlDriver, req.Phone, req.Address, userID); err != nil {
			slog.Error("Failed to update driver info", "error", err)
			http.Error(w, "Update Driver Failed", http.StatusInternalServerError)
			return
		}
	}

	if req.AvatarURL != "" {
		sqlProfile := `UPDATE profiles SET avatar_url = $1, updated_at = NOW() WHERE id = $2`
		if _, err := tx.Exec(r.Context(), sqlProfile, req.AvatarURL, userID); err != nil {
			slog.Error("Failed to update avatar", "error", err)
			http.Error(w, "Update Avatar Failed", http.StatusInternalServerError)
			return
		}
	}

	if req.ReferralCode != "" {
		code := strings.TrimSpace(req.ReferralCode)

		// Check if user already has a referrer
		var existingRef *string
		err := tx.QueryRow(r.Context(), "SELECT referred_by_code FROM profiles WHERE id = $1", userID).Scan(&existingRef)
		if err != nil {
			slog.Error("Failed to check existing referrer", "error", err)
			http.Error(w, "Database Error", http.StatusInternalServerError)
			return
		}

		if existingRef != nil {
			http.Error(w, "User already has a referrer", http.StatusConflict) // 409 Conflict
			return
		}

		var referrerID string
		err = tx.QueryRow(r.Context(), "SELECT id FROM profiles WHERE referral_code = $1", code).Scan(&referrerID)
		if err != nil {
			http.Error(w, "Invalid referral code", http.StatusBadRequest)
			return
		}

		if referrerID == userID {
			http.Error(w, "Cannot refer yourself", http.StatusBadRequest)
			return
		}

		if _, err := tx.Exec(r.Context(), "UPDATE profiles SET referred_by_code = $1 WHERE id = $2", code, userID); err != nil {
			slog.Error("Failed to set referred_by_code", "error", err)
			http.Error(w, "Failed to apply referral", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Commit Failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"updated"}`))

	go func() {
		cfg, _ := config.Load()
		var emailTo, fullName string
		if err := database.Pool.QueryRow(context.Background(),
			"SELECT p.email, COALESCE(da.full_name, 'User') FROM profiles p LEFT JOIN driver_applications da ON p.id = da.user_id WHERE p.id = $1",
			userID).Scan(&emailTo, &fullName); err == nil {

			subject := "Security Alert: Profile Updated"
			body := fmt.Sprintf(`<p>Hello %s,</p><p>Your G4 Car Service profile information was recently updated.</p>`, fullName)
			email.SendEmail([]string{emailTo}, subject, body, cfg)
		}
	}()
}
