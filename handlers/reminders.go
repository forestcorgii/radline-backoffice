package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"rav/db"
	"rav/domain"
	"rav/models"
)

type CalendarDay struct {
	DayNumber int
	DateStr   string
	IsToday   bool
	Reminders []models.Reminder
}

type CalendarWeek struct {
	Days []CalendarDay
}

// sanitizeReminderReturnURL ensures a return URL from HX-Current-URL or Referer is a safe relative internal path.
func sanitizeReminderReturnURL(raw string, r *http.Request) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/reminders"
	}
	if strings.ContainsAny(trimmed, "\r\n\"\\") {
		return "/reminders"
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "/reminders"
	}

	if u.Scheme != "" || u.Host != "" {
		if r == nil || r.Host == "" || !strings.EqualFold(u.Host, r.Host) {
			return "/reminders"
		}
	}

	path := u.Path
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "/reminders"
	}

	if u.RawQuery != "" {
		return path + "?" + u.RawQuery
	}
	return path
}

func resolveReminderReturnURL(r *http.Request) string {
	raw := r.Header.Get("HX-Current-URL")
	if raw == "" {
		raw = r.Header.Get("Referer")
	}
	return sanitizeReminderReturnURL(raw, r)
}

func isReminderAdminOrManager(currUser *CurrentUserContext) bool {
	if currUser == nil {
		return true
	}
	return strings.EqualFold(currUser.User.RoleName, "Admin") || currUser.HasPermission(domain.PermSettingsEdit)
}

func canCreateReminder(currUser *CurrentUserContext) bool {
	if currUser == nil {
		return true
	}
	if isReminderAdminOrManager(currUser) {
		return true
	}
	return currUser.HasPermission(domain.PermSalesCreate) ||
		currUser.HasPermission(domain.PermInventoryReceive) ||
		currUser.HasPermission(domain.PermItemsEdit)
}

func canModifyReminder(currUser *CurrentUserContext, assignedToUserID *int64) bool {
	if currUser == nil {
		return true
	}
	if isReminderAdminOrManager(currUser) {
		return true
	}
	if !canCreateReminder(currUser) {
		return false
	}
	return assignedToUserID != nil && *assignedToUserID == currUser.User.ID
}

// RemindersHandler renders the interactive calendar view and reminder list
func (app *App) RemindersHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	monthParam := r.URL.Query().Get("month") // "YYYY-MM"
	viewDate := now
	if monthParam != "" {
		if t, err := time.Parse("2006-01", monthParam); err == nil {
			viewDate = t
		}
	}

	year, month, _ := viewDate.Date()
	firstOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, viewDate.Location())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	endOfMonth := time.Date(year, month, lastOfMonth.Day(), 23, 59, 59, 999999999, viewDate.Location())

	prevMonthStr := firstOfMonth.AddDate(0, -1, 0).Format("2006-01")
	nextMonthStr := firstOfMonth.AddDate(0, 1, 0).Format("2006-01")
	currentMonthStr := firstOfMonth.Format("2006-01")
	displayMonthName := firstOfMonth.Format("January 2006")

	// Fetch all reminders for this month using typed time.Time parameters for PostgreSQL TIMESTAMPTZ compatibility
	var monthlyReminders []models.Reminder
	if db.DB != nil {
		err := db.DB.Select(&monthlyReminders, `
			SELECT 
				r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
				r.assigned_to_user_id,
				COALESCE(u.full_name, u.username, '') AS assigned_to_name
			FROM reminders r
			LEFT JOIN users u ON r.assigned_to_user_id = u.id
			WHERE r.due_date >= ? AND r.due_date <= ?
			ORDER BY r.due_date ASC, r.priority DESC
		`, firstOfMonth, endOfMonth)
		if err != nil {
			log.Printf("[RemindersHandler] Warning: failed to fetch monthly reminders with time.Time: %v. Retrying with string fallback...", err)
			_ = db.DB.Select(&monthlyReminders, `
				SELECT 
					r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
					r.assigned_to_user_id,
					COALESCE(u.full_name, u.username, '') AS assigned_to_name
				FROM reminders r
				LEFT JOIN users u ON r.assigned_to_user_id = u.id
				WHERE r.due_date >= ? AND r.due_date <= ?
				ORDER BY r.due_date ASC, r.priority DESC
			`, firstOfMonth.Format("2006-01-02 00:00:00"), lastOfMonth.Format("2006-01-02 23:59:59"))
		}
	}
	if monthlyReminders == nil {
		monthlyReminders = []models.Reminder{}
	}

	// Group reminders by date string ("YYYY-MM-DD")
	remindersByDate := make(map[string][]models.Reminder)
	for _, rem := range monthlyReminders {
		dStr := rem.DueDate.Format("2006-01-02")
		remindersByDate[dStr] = append(remindersByDate[dStr], rem)
	}

	// Build calendar weeks
	startWeekday := int(firstOfMonth.Weekday()) // 0 = Sunday
	totalDays := lastOfMonth.Day()

	var weeks []CalendarWeek
	currentWeek := CalendarWeek{}

	// Leading blank days
	for i := 0; i < startWeekday; i++ {
		currentWeek.Days = append(currentWeek.Days, CalendarDay{DayNumber: 0})
	}

	todayStr := now.Format("2006-01-02")
	for day := 1; day <= totalDays; day++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		calDay := CalendarDay{
			DayNumber: day,
			DateStr:   dateStr,
			IsToday:   dateStr == todayStr,
			Reminders: remindersByDate[dateStr],
		}
		currentWeek.Days = append(currentWeek.Days, calDay)

		if len(currentWeek.Days) == 7 {
			weeks = append(weeks, currentWeek)
			currentWeek = CalendarWeek{}
		}
	}

	// Trailing blank days
	if len(currentWeek.Days) > 0 {
		for len(currentWeek.Days) < 7 {
			currentWeek.Days = append(currentWeek.Days, CalendarDay{DayNumber: 0})
		}
		weeks = append(weeks, currentWeek)
	}

	// Fetch upcoming pending agenda
	var agenda []models.Reminder
	if db.DB != nil {
		errAgenda := db.DB.Select(&agenda, `
			SELECT 
				r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
				r.assigned_to_user_id,
				COALESCE(u.full_name, u.username, '') AS assigned_to_name
			FROM reminders r
			LEFT JOIN users u ON r.assigned_to_user_id = u.id
			WHERE r.status = 'Pending'
			ORDER BY r.due_date ASC
			LIMIT 15
		`)
		if errAgenda != nil {
			log.Printf("[RemindersHandler] Warning: failed to fetch agenda: %v", errAgenda)
		}
	}
	if agenda == nil {
		agenda = []models.Reminder{}
	}

	// Fetch active users for assignment dropdown
	var users []models.User
	if db.DB != nil {
		_ = db.DB.Select(&users, `
			SELECT id, username, full_name
			FROM users
			WHERE is_active = TRUE
			ORDER BY full_name ASC, username ASC
		`)
	}

	data := struct {
		Weeks            []CalendarWeek
		Agenda           []models.Reminder
		Users            []models.User
		CurrentMonthStr  string
		DisplayMonthName string
		PrevMonthStr     string
		NextMonthStr     string
		TodayDateStr     string
	}{
		Weeks:            weeks,
		Agenda:           agenda,
		Users:            users,
		CurrentMonthStr:  currentMonthStr,
		DisplayMonthName: displayMonthName,
		PrevMonthStr:     prevMonthStr,
		NextMonthStr:     nextMonthStr,
		TodayDateStr:     todayStr,
	}

	app.RenderPage(w, r, "reminders.html", data)
}

// AddReminderHandler creates a new scheduled task
func (app *App) AddReminderHandler(w http.ResponseWriter, r *http.Request) {
	currUser := GetCurrentUser(r)
	if !canCreateReminder(currUser) {
		setHXTriggerToast(w, "error", "Access Denied: You do not have permission to create reminders.")
		http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	details := strings.TrimSpace(r.FormValue("details"))
	dueDateStr := strings.TrimSpace(r.FormValue("due_date"))
	priorityStr := strings.TrimSpace(r.FormValue("priority"))
	category := strings.TrimSpace(r.FormValue("category"))
	assignedToStr := strings.TrimSpace(r.FormValue("assigned_to_user_id"))

	if priorityStr == "" {
		priorityStr = "Medium"
	}
	if category == "" {
		category = "General"
	}

	dueDate, err := time.Parse("2006-01-02", dueDateStr)
	if err != nil {
		setHXTriggerToast(w, "error", "Invalid due date format. Expected YYYY-MM-DD.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminder(title, dueDate, domain.ReminderPriority(priorityStr)); err != nil {
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var assignedToUserID *int64
	if assignedToStr != "" {
		if uid, err := strconv.ParseInt(assignedToStr, 10, 64); err == nil && uid > 0 {
			assignedToUserID = &uid
		}
	}

	if currUser != nil && !isReminderAdminOrManager(currUser) {
		if assignedToUserID == nil {
			uid := currUser.User.ID
			assignedToUserID = &uid
		} else if *assignedToUserID != currUser.User.ID {
			setHXTriggerToast(w, "error", "Access Denied: You can only assign reminders to yourself.")
			http.Error(w, "403 Forbidden: Cannot assign reminder to another user", http.StatusForbidden)
			return
		}
	}

	var newID int64
	err = db.DB.QueryRow(`
		INSERT INTO reminders (title, details, due_date, priority, category, status, assigned_to_user_id)
		VALUES (?, ?, ?, ?, ?, 'Pending', ?)
		RETURNING id
	`, title, details, dueDate, priorityStr, category, assignedToUserID).Scan(&newID)
	if err != nil {
		log.Printf("ERROR: Failed to create reminder: %v", err)
		setHXTriggerToast(w, "error", "Failed to create reminder.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_REMINDER", "Reminder", fmt.Sprintf("%d", newID), fmt.Sprintf("Scheduled reminder '%s' for %s", title, dueDateStr))
	setHXTriggerToast(w, "success", "Reminder created successfully!")
	returnUrl := resolveReminderReturnURL(r)
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" || r.Header.Get("HX-Request") == "true" {
		setHXLocation(w, returnUrl, "#main-content")
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}

// EditReminderHandler updates an existing scheduled reminder
func (app *App) EditReminderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	var existingAssignedTo *int64
	err = db.DB.Get(&existingAssignedTo, "SELECT assigned_to_user_id FROM reminders WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Reminder not found", http.StatusNotFound)
		return
	}

	currUser := GetCurrentUser(r)
	if !canModifyReminder(currUser, existingAssignedTo) {
		setHXTriggerToast(w, "error", "Access Denied: You can only modify your own reminders.")
		http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	details := strings.TrimSpace(r.FormValue("details"))
	dueDateStr := strings.TrimSpace(r.FormValue("due_date"))
	priorityStr := strings.TrimSpace(r.FormValue("priority"))
	category := strings.TrimSpace(r.FormValue("category"))
	assignedToStr := strings.TrimSpace(r.FormValue("assigned_to_user_id"))
	status := strings.TrimSpace(r.FormValue("status"))

	if priorityStr == "" {
		priorityStr = "Medium"
	}
	if category == "" {
		category = "General"
	}
	if status == "" {
		status = "Pending"
	}

	dueDate, err := time.Parse("2006-01-02", dueDateStr)
	if err != nil {
		setHXTriggerToast(w, "error", "Invalid due date format. Expected YYYY-MM-DD.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminder(title, dueDate, domain.ReminderPriority(priorityStr)); err != nil {
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminderStatus(domain.ReminderStatus(status)); err != nil {
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var assignedToUserID *int64
	if assignedToStr != "" {
		if uid, err := strconv.ParseInt(assignedToStr, 10, 64); err == nil && uid > 0 {
			assignedToUserID = &uid
		}
	}

	if currUser != nil && !isReminderAdminOrManager(currUser) {
		if assignedToUserID == nil {
			uid := currUser.User.ID
			assignedToUserID = &uid
		} else if *assignedToUserID != currUser.User.ID {
			setHXTriggerToast(w, "error", "Access Denied: You cannot reassign reminders to another user.")
			http.Error(w, "403 Forbidden: Cannot reassign reminder", http.StatusForbidden)
			return
		}
	}

	_, err = db.DB.Exec(`
		UPDATE reminders
		SET title = ?, details = ?, due_date = ?, priority = ?, category = ?, status = ?, assigned_to_user_id = ?
		WHERE id = ?
	`, title, details, dueDate, priorityStr, category, status, assignedToUserID, id)
	if err != nil {
		log.Printf("EditReminderHandler update error: %v", err)
		setHXTriggerToast(w, "error", "Failed to update reminder.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "UPDATE_REMINDER", "Reminder", idStr, fmt.Sprintf("Updated reminder '%s' (due %s)", title, dueDateStr))
	setHXTriggerToast(w, "success", "Reminder updated successfully!")
	returnUrl := resolveReminderReturnURL(r)
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" || r.Header.Get("HX-Request") == "true" {
		setHXLocation(w, returnUrl, "#main-content")
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}

// ToggleReminderHandler flips status between Pending and Completed
func (app *App) ToggleReminderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("ToggleReminderHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to toggle reminder.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var row struct {
		Status           string `db:"status"`
		AssignedToUserID *int64 `db:"assigned_to_user_id"`
	}
	err = tx.Get(&row, "SELECT status, assigned_to_user_id FROM reminders WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Reminder not found", http.StatusNotFound)
		return
	}

	currUser := GetCurrentUser(r)
	if !canModifyReminder(currUser, row.AssignedToUserID) {
		setHXTriggerToast(w, "error", "Access Denied: You can only toggle your own reminders.")
		http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
		return
	}

	newStatus := "Completed"
	if row.Status == "Completed" {
		newStatus = "Pending"
	}

	if _, err := tx.Exec("UPDATE reminders SET status = ? WHERE id = ?", newStatus, id); err != nil {
		log.Printf("ToggleReminderHandler update error: %v", err)
		setHXTriggerToast(w, "error", "Failed to update reminder status.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ToggleReminderHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit reminder status.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "TOGGLE_REMINDER", "Reminder", idStr, fmt.Sprintf("Marked reminder ID %s as %s", idStr, newStatus))

	returnUrl := resolveReminderReturnURL(r)
	setHXTriggerToast(w, "success", fmt.Sprintf("Reminder marked as %s", newStatus))
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" || r.Header.Get("HX-Request") == "true" {
		setHXLocation(w, returnUrl, "#main-content")
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}

// DeleteReminderHandler deletes a reminder
func (app *App) DeleteReminderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	var existingAssignedTo *int64
	err = db.DB.Get(&existingAssignedTo, "SELECT assigned_to_user_id FROM reminders WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Reminder not found", http.StatusNotFound)
		return
	}

	currUser := GetCurrentUser(r)
	if !canModifyReminder(currUser, existingAssignedTo) {
		setHXTriggerToast(w, "error", "Access Denied: You can only delete your own reminders.")
		http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
		return
	}

	if _, err := db.DB.Exec("DELETE FROM reminders WHERE id = ?", id); err != nil {
		log.Printf("DeleteReminderHandler delete error: %v", err)
		setHXTriggerToast(w, "error", "Failed to delete reminder.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	app.LogActivity(r, "DELETE_REMINDER", "Reminder", idStr, "Removed reminder ID "+idStr)

	returnUrl := resolveReminderReturnURL(r)
	setHXTriggerToast(w, "success", "Reminder deleted")
	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" || r.Header.Get("HX-Request") == "true" {
		setHXLocation(w, returnUrl, "#main-content")
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}

// RescheduleReminderHandler updates only the due_date of a reminder (for calendar drag & drop)
func (app *App) RescheduleReminderHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	dueDateStr := strings.TrimSpace(r.FormValue("due_date"))
	dueDate, err := time.Parse("2006-01-02", dueDateStr)
	if err != nil {
		setHXTriggerToast(w, "error", "Invalid date format. Expected YYYY-MM-DD.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var row struct {
		Title            string `db:"title"`
		AssignedToUserID *int64 `db:"assigned_to_user_id"`
	}
	err = db.DB.Get(&row, "SELECT title, assigned_to_user_id FROM reminders WHERE id = ?", id)
	if err != nil {
		setHXTriggerToast(w, "error", "Reminder not found.")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	currUser := GetCurrentUser(r)
	if !canModifyReminder(currUser, row.AssignedToUserID) {
		setHXTriggerToast(w, "error", "Access Denied: You can only reschedule your own reminders.")
		http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
		return
	}

	_, err = db.DB.Exec("UPDATE reminders SET due_date = ? WHERE id = ?", dueDate, id)
	if err != nil {
		log.Printf("RescheduleReminderHandler update error: %v", err)
		setHXTriggerToast(w, "error", "Failed to reschedule reminder.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "RESCHEDULE_REMINDER", "Reminder", idStr, fmt.Sprintf("Rescheduled reminder '%s' to %s", row.Title, dueDateStr))

	setHXTriggerToast(w, "success", fmt.Sprintf("Rescheduled '%s' to %s!", row.Title, dueDate.Format("Jan 02, 2006")))

	returnUrl := resolveReminderReturnURL(r)

	target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#")
	if target == "main-content" {
		setHXLocation(w, returnUrl, "#main-content")
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}
