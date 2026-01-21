package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"g4-services/api/config"
	"g4-services/api/database"
	"g4-services/api/services/email"
)

type UpdateProfileRequest struct {
	Phone     string `json:"phone_number"`
	Address   string `json:"address"`
	AvatarURL string `json:"avatar_url"`
}

// UpdateMyProfile godoc
// @Summary      Update My Profile
// @Description  Update user phone, address or avatar
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body UpdateProfileRequest true "Update Data"
// @Success      200  {string}  string "{"status":"updated"}"
// @Failure      400  {string}  string "Invalid Body"
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
