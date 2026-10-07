package handlers

import (
	"fmt"
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

	prevMonthStr := firstOfMonth.AddDate(0, -1, 0).Format("2006-01")
	nextMonthStr := firstOfMonth.AddDate(0, 1, 0).Format("2006-01")
	currentMonthStr := firstOfMonth.Format("2006-01")
	displayMonthName := firstOfMonth.Format("January 2006")

	// Fetch all reminders for this month
	var monthlyReminders []models.Reminder
	err := db.DB.Select(&monthlyReminders, `
		SELECT id, title, details, due_date, priority, category, status, created_at
		FROM reminders
		WHERE due_date >= ? AND due_date <= ?
		ORDER BY due_date ASC, priority DESC
	`, firstOfMonth.Format("2006-01-02 00:00:00"), lastOfMonth.Format("2006-01-02 23:59:59"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
	_ = db.DB.Select(&agenda, `
		SELECT id, title, details, due_date, priority, category, status, created_at
		FROM reminders
		WHERE status = 'Pending'
		ORDER BY due_date ASC
		LIMIT 15
	`)

	data := struct {
		Weeks            []CalendarWeek
		Agenda           []models.Reminder
		CurrentMonthStr  string
		DisplayMonthName string
		PrevMonthStr     string
		NextMonthStr     string
		TodayDateStr     string
	}{
		Weeks:            weeks,
		Agenda:           agenda,
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

	var newID int64
	err = db.DB.QueryRow(`
		INSERT INTO reminders (title, details, due_date, priority, category, status)
		VALUES (?, ?, ?, ?, ?, 'Pending')
		RETURNING id
	`, title, details, dueDate, priorityStr, category).Scan(&newID)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to create reminder."}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_REMINDER", "Reminder", fmt.Sprintf("%d", newID), fmt.Sprintf("Scheduled reminder '%s' for %s", title, dueDateStr))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Reminder created successfully!"}}`)
	returnUrl := r.Header.Get("Referer")
	if returnUrl == "" {
		returnUrl = "/reminders"
	}
	w.Header().Set("HX-Redirect", returnUrl)
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
		returnUrl = "/reminders"
	}

	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "success", "message": "Reminder marked as %s"}}`, newStatus))
	if r.Header.Get("HX-Target") == "#main-content" {
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
		returnUrl = "/reminders"
	}

	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Reminder deleted"}}`)
	if r.Header.Get("HX-Target") == "#main-content" {
		w.Header().Set("HX-Location", fmt.Sprintf(`{"path": "%s", "target": "#main-content"}`, returnUrl))
	} else {
		w.Header().Set("HX-Redirect", returnUrl)
	}
	w.WriteHeader(http.StatusOK)
}
