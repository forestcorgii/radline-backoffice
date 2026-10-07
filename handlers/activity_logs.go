package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"radline/db"
	"radline/models"
)

// getClientIP extracts caller IP address
func getClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	xrip := r.Header.Get("X-Real-IP")
	if xrip != "" {
		return strings.TrimSpace(xrip)
	}
	return r.RemoteAddr
}

// LogActivity records an action performed by the current authenticated user
func (app *App) LogActivity(r *http.Request, action, entityType, entityID, details string) {
	var userID int64
	username := "System/Anonymous"
	if r != nil {
		ctxUser := GetCurrentUser(r)
		if ctxUser != nil {
			userID = ctxUser.User.ID
			username = ctxUser.User.Username
		}
	}
	app.LogActivityWithUser(userID, username, r, action, entityType, entityID, details)
}

// LogActivityWithUser records an action with explicit user credentials
func (app *App) LogActivityWithUser(userID int64, username string, r *http.Request, action, entityType, entityID, details string) {
	if db.DB == nil {
		return
	}
	ip := getClientIP(r)

	_, err := db.DB.Exec(`
		INSERT INTO activity_logs (user_id, username, action, entity_type, entity_id, details, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, username, action, entityType, entityID, details, ip)
	if err != nil {
		log.Printf("[AUDIT LOG ERROR] Failed to record activity log: %v", err)
	}
}

// ActivityLogsHandler displays audit log trail with search and filter
func (app *App) ActivityLogsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 25
	offset := (page - 1) * pageSize

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	actionFilter := strings.TrimSpace(r.URL.Query().Get("action"))
	entityFilter := strings.TrimSpace(r.URL.Query().Get("entity"))

	query := `
		SELECT id, user_id, username, action, entity_type, entity_id, details, ip_address, created_at
		FROM activity_logs
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM activity_logs WHERE 1=1`
	var args []interface{}

	if search != "" {
		clause := " AND (username LIKE ? OR details LIKE ? OR entity_id LIKE ?)"
		query += clause
		countQuery += clause
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}

	if actionFilter != "" {
		clause := " AND action = ?"
		query += clause
		countQuery += clause
		args = append(args, actionFilter)
	}

	if entityFilter != "" {
		clause := " AND entity_type = ?"
		query += clause
		countQuery += clause
		args = append(args, entityFilter)
	}

	var totalRecords int
	_ = db.DB.Get(&totalRecords, countQuery, args...)

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	var logs []models.ActivityLog
	err := db.DB.Select(&logs, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	data := struct {
		Logs         []models.ActivityLog
		Search       string
		ActionFilter string
		EntityFilter string
		Page         int
		TotalPages   int
		TotalRecords int
	}{
		Logs:         logs,
		Search:       search,
		ActionFilter: actionFilter,
		EntityFilter: entityFilter,
		Page:         page,
		TotalPages:   totalPages,
		TotalRecords: totalRecords,
	}

	if r.Header.Get("HX-Request") == "true" && r.URL.Query().Get("fragment") == "true" {
		t, ok := app.Templates["activity_logs_results.html"]
		if !ok {
			http.Error(w, "Fragment template not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = t.Execute(w, data)
		return
	}

	app.RenderPage(w, r, "activity_logs.html", data)
}
