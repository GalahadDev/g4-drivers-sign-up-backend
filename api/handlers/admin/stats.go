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
			COUNT(*) FILTER (WHERE p.role != 'admin')                                               AS total_users,
			COUNT(*) FILTER (WHERE da.driver_category = 'comfort' AND p.role != 'admin')           AS total_comfort,
			COUNT(*) FILTER (WHERE da.driver_category = 'luxury'  AND p.role != 'admin')           AS total_luxury,
			COUNT(*) FILTER (WHERE p.created_at > NOW() - INTERVAL '7 days' AND p.role != 'admin') AS new_users,
			COUNT(*) FILTER (WHERE p.referred_by_code IS NOT NULL)                                  AS total_referrals
		FROM profiles p
		LEFT JOIN driver_applications da ON da.user_id = p.id`

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
