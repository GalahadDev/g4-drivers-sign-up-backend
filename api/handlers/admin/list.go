package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	CreatedAt      string `json:"created_at"`
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
// @Param        search query    string  false  "Search by name or email"
// @Param        category query  string  false  "Filter by category"
// @Success      200   {object}  UserListResponse
// @Failure      403   {string}  string "Forbidden"
// @Failure      500   {string}  string "DB Error"
// @Router       /admin/users [get]
// @Security     BearerAuth
func ListUsers(w http.ResponseWriter, r *http.Request) {

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

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

	// 1. Build Dynamic Dynamic WHERE clause
	whereSQL := "WHERE p.role != 'admin'"
	args := []interface{}{}
	argCounter := 1

	// Filter by Category
	if category != "" && category != "all" {
		whereSQL += " AND da.driver_category = $" + strconv.Itoa(argCounter)
		args = append(args, category)
		argCounter++
	}

	// Filter by Search (Email or Name)
	if search != "" {
		searchParam := "%" + search + "%"
		whereSQL += " AND (p.email ILIKE $" + strconv.Itoa(argCounter) + " OR da.full_name ILIKE $" + strconv.Itoa(argCounter) + ")"
		args = append(args, searchParam)
		argCounter++
	}

	// 2. Get Data with CTE for referral counts + window function for total
	offset := (page - 1) * limit
	args = append(args, limit, offset)

	dataSQL := `
		WITH referral_counts AS (
			SELECT referred_by_code, COUNT(*) AS total
			FROM profiles
			WHERE referred_by_code IS NOT NULL
			GROUP BY referred_by_code
		)
		SELECT
			p.id, p.email, p.role,
			COALESCE(da.full_name, 'N/A'),
			COALESCE(da.status, 'pending'),
			COALESCE(da.driver_category, 'none'),
			COALESCE(rc.total, 0) AS referred_count,
			p.created_at,
			COUNT(*) OVER() AS total_count
		FROM profiles p
		LEFT JOIN driver_applications da ON p.id = da.user_id
		LEFT JOIN referral_counts rc ON rc.referred_by_code = p.referral_code
		` + whereSQL + `
		ORDER BY p.created_at DESC
		LIMIT $` + strconv.Itoa(argCounter) + ` OFFSET $` + strconv.Itoa(argCounter+1)

	var rows pgx.Rows
	rows, err = database.Pool.Query(r.Context(), dataSQL, args...)
	if err != nil {
		slog.Error("Admin list users error", "error", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var totalItems int
	users := []AdminUserListItem{}
	for rows.Next() {
		var u AdminUserListItem
		var createdAtTime time.Time
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.DriverStatus, &u.DriverCategory, &u.ReferredCount, &createdAtTime, &totalItems); err == nil {
			u.CreatedAt = createdAtTime.Format(time.RFC3339)
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
	if totalItems == 0 && resp.Meta.TotalPages == 0 {
		resp.Meta.TotalPages = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
