package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
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

	// Ensure reminders table exists in the active database
	_ = db.EnsureRemindersTable()

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
			_ = db.EnsureRemindersTable()
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
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid due date format. Expected YYYY-MM-DD."}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminder(title, dueDate, domain.ReminderPriority(priorityStr)); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var assignedToUserID *int64
	if assignedToStr != "" {
		if uid, err := strconv.ParseInt(assignedToStr, 10, 64); err == nil && uid > 0 {
			assignedToUserID = &uid
		}
	}

	var newID int64
	_ = db.EnsureRemindersTable()
	err = db.DB.QueryRow(`
		INSERT INTO reminders (title, details, due_date, priority, category, status, assigned_to_user_id)
		VALUES (?, ?, ?, ?, ?, 'Pending', ?)
		RETURNING id
	`, title, details, dueDate, priorityStr, category, assignedToUserID).Scan(&newID)
	if err != nil {
		log.Printf("ERROR: Failed to create reminder: %v", err)
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to create reminder."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_REMINDER", "Reminder", fmt.Sprintf("%d", newID), fmt.Sprintf("Scheduled reminder '%s' for %s", title, dueDateStr))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Reminder created successfully!"}}`)
	returnUrl := r.Header.Get("HX-Current-URL")
	if returnUrl == "" {
		returnUrl = r.Header.Get("Referer")
	}
	if returnUrl == "" {
		returnUrl = "/calendar"
	}
	target := r.Header.Get("HX-Target")
	if target == "#main-content" || target == "main-content" || r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
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
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid due date format. Expected YYYY-MM-DD."}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminder(title, dueDate, domain.ReminderPriority(priorityStr)); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := domain.ValidateReminderStatus(domain.ReminderStatus(status)); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var assignedToUserID *int64
	if assignedToStr != "" {
		if uid, err := strconv.ParseInt(assignedToStr, 10, 64); err == nil && uid > 0 {
			assignedToUserID = &uid
		}
	}

	_, err = db.DB.Exec(`
		UPDATE reminders
		SET title = ?, details = ?, due_date = ?, priority = ?, category = ?, status = ?, assigned_to_user_id = ?
		WHERE id = ?
	`, title, details, dueDate, priorityStr, category, status, assignedToUserID, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update reminder."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "UPDATE_REMINDER", "Reminder", idStr, fmt.Sprintf("Updated reminder '%s' (due %s)", title, dueDateStr))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Reminder updated successfully!"}}`)
	returnUrl := r.Header.Get("HX-Current-URL")
	if returnUrl == "" {
		returnUrl = r.Header.Get("Referer")
	}
	if returnUrl == "" {
		returnUrl = "/calendar"
	}
	target := r.Header.Get("HX-Target")
	if target == "#main-content" || target == "main-content" || r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
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

	var curStatus string
	err = db.DB.Get(&curStatus, "SELECT status FROM reminders WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Reminder not found", http.StatusNotFound)
		return
	}

	newStatus := "Completed"
	if curStatus == "Completed" {
		newStatus = "Pending"
	}

	_, _ = db.DB.Exec("UPDATE reminders SET status = ? WHERE id = ?", newStatus, id)
	app.LogActivity(r, "TOGGLE_REMINDER", "Reminder", idStr, fmt.Sprintf("Marked reminder ID %s as %s", idStr, newStatus))

	returnUrl := r.Header.Get("HX-Current-URL")
	if returnUrl == "" {
		returnUrl = r.Header.Get("Referer")
	}
	if returnUrl == "" {
		returnUrl = "/calendar"
	}

	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "success", "message": "Reminder marked as %s"}}`, newStatus))
	target := r.Header.Get("HX-Target")
	if target == "#main-content" || target == "main-content" || r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
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

	_, _ = db.DB.Exec("DELETE FROM reminders WHERE id = ?", id)
	app.LogActivity(r, "DELETE_REMINDER", "Reminder", idStr, "Removed reminder ID "+idStr)

	returnUrl := r.Header.Get("HX-Current-URL")
	if returnUrl == "" {
		returnUrl = r.Header.Get("Referer")
	}
	if returnUrl == "" {
		returnUrl = "/calendar"
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Reminder deleted"}}`)
	target := r.Header.Get("HX-Target")
	if target == "#main-content" || target == "main-content" || r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
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
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid date format. Expected YYYY-MM-DD."}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var title string
	err = db.DB.Get(&title, "SELECT title FROM reminders WHERE id = ?", id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Reminder not found."}}`)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, err = db.DB.Exec("UPDATE reminders SET due_date = ? WHERE id = ?", dueDate, id)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to reschedule reminder."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "RESCHEDULE_REMINDER", "Reminder", idStr, fmt.Sprintf("Rescheduled reminder '%s' to %s", title, dueDateStr))

	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "success", "message": "Rescheduled '%s' to %s!"}}`, title, dueDate.Format("Jan 02, 2006")))

	returnUrl := r.Header.Get("HX-Current-URL")
	if returnUrl == "" {
		returnUrl = r.Header.Get("Referer")
	}
	if returnUrl == "" {
		returnUrl = "/calendar"
	}

	target := r.Header.Get("HX-Target")
	if target == "#main-content" || target == "main-content" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}
