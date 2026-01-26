package admin

import (
	"encoding/json"
	"net/http"

	"g4-services/api/database"
)

type AdminStats struct {
	TotalUsers       int `json:"total_users"`
	TotalComfort     int `json:"total_comfort"`
	TotalLuxury      int `json:"total_luxury"`
	NewUsersLastWeek int `json:"new_users_last_week"`
	TotalReferrals   int `json:"total_referrals"`
}

// GetGlobalStats godoc
// @Summary      Get Admin Stats
// @Description  Get global platform statistics (users, applications, referrals)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200   {object}  AdminStats
// @Failure      500   {string}  string "Stats Error"
// @Router       /admin/stats [get]
// @Security     BearerAuth
func GetGlobalStats(w http.ResponseWriter, r *http.Request) {
	stats := AdminStats{}

	sql := `
		SELECT
			(SELECT COUNT(*) FROM profiles WHERE role != 'admin') as total_users,
			(SELECT COUNT(*) 
			 FROM driver_applications da 
			 JOIN profiles p ON da.user_id = p.id 
			 WHERE da.driver_category = 'comfort' AND p.role != 'admin') as comfort,
			(SELECT COUNT(*) 
			 FROM driver_applications da 
			 JOIN profiles p ON da.user_id = p.id 
			 WHERE da.driver_category = 'luxury' AND p.role != 'admin') as luxury,
			(SELECT COUNT(*) FROM profiles WHERE created_at > NOW() - INTERVAL '7 days' AND role != 'admin') as new_users,
			(SELECT COUNT(*) FROM profiles WHERE referred_by_code IS NOT NULL) as total_referrals
	`

	err := database.Pool.QueryRow(r.Context(), sql).Scan(
		&stats.TotalUsers,
		&stats.TotalComfort,
		&stats.TotalLuxury,
		&stats.NewUsersLastWeek,
		&stats.TotalReferrals,
	)

	if err != nil {
		http.Error(w, "Stats Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
