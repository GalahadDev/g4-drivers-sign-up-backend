package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"g4-services/api/database"

	"github.com/jackc/pgx/v5"
)

type AdminUserListItem struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	DriverStatus   string `json:"driver_status"`
	DriverCategory string `json:"driver_category"`
	ReferredCount  int    `json:"referred_count"`
}

type UserListResponse struct {
	Data []AdminUserListItem `json:"data"`
	Meta PaginationMeta      `json:"meta"`
}

type PaginationMeta struct {
	TotalItems   int `json:"total_items"`
	TotalPages   int `json:"total_pages"`
	CurrentPage  int `json:"current_page"`
	ItemsPerPage int `json:"items_per_page"`
}

// ListUsers godoc
// @Summary      List Users (Admin)
// @Description  Get paginated list of all users and their application status
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        page  query     int     false  "Page number"
// @Param        limit query     int     false  "Items per page"
// @Success      200   {object}  UserListResponse
// @Failure      403   {string}  string "Forbidden"
// @Failure      500   {string}  string "DB Error"
// @Router       /admin/users [get]
// @Security     BearerAuth
func ListUsers(w http.ResponseWriter, r *http.Request) {

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var totalItems int
	err = database.Pool.QueryRow(r.Context(), "SELECT COUNT(*) FROM profiles").Scan(&totalItems)
	if err != nil {
		slog.Error("Admin list users count error", "error", err)
		totalItems = 0
	}

	offset := (page - 1) * limit
	sql := `
		SELECT 
			p.id, p.email, p.role,
			COALESCE(da.full_name, 'N/A'),
			COALESCE(da.status, 'no_app'),
			COALESCE(da.driver_category, 'none'),
			(SELECT COUNT(*) FROM profiles WHERE referred_by_code = p.referral_code) as total_referrals
		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`

	var rows pgx.Rows
	rows, err = database.Pool.Query(r.Context(), sql, limit, offset)
	if err != nil {
		slog.Error("Admin list users error", "error", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []AdminUserListItem{}
	for rows.Next() {
		var u AdminUserListItem
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.DriverStatus, &u.DriverCategory, &u.ReferredCount); err == nil {
			users = append(users, u)
		}
	}

	resp := UserListResponse{
		Data: users,
		Meta: PaginationMeta{
			TotalItems:   totalItems,
			TotalPages:   (totalItems + limit - 1) / limit,
			CurrentPage:  page,
			ItemsPerPage: limit,
		},
	}
	if totalItems == 0 {
		resp.Meta.TotalPages = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
