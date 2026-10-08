package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"radline/db"
	"radline/models"
)

func parseRemindersTemplatesForTest() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"derefInt": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			d := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				k, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				d[k] = values[i+1]
			}
			return d, nil
		},
	}

	pages := []string{"reminders.html"}
	for _, page := range pages {
		t := template.New(page).Funcs(funcMap)
		files := []string{"../templates/base.html", "../templates/" + page}
		t = template.Must(t.ParseFiles(files...))
		templates[page] = t
	}

	return templates
}

func TestReminders_AddEditAndAssign(t *testing.T) {
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	app := &App{
		Templates: parseRemindersTemplatesForTest(),
	}

	// Ensure there is at least one active user to assign to
	var user models.User
	err = db.DB.Get(&user, "SELECT id, username, full_name FROM users WHERE is_active = TRUE LIMIT 1")
	if err != nil {
		t.Fatalf("Expected at least one active user: %v", err)
	}

	// 1. Create a reminder assigned to this user
	addForm := url.Values{}
	addForm.Set("title", "Test Assignment Reminder")
	addForm.Set("details", "Verify assigning to user works")
	addForm.Set("due_date", time.Now().Format("2006-01-02"))
	addForm.Set("priority", "High")
	addForm.Set("category", "Payment")
	addForm.Set("assigned_to_user_id", fmt.Sprintf("%d", user.ID))

	addReq := httptest.NewRequest(http.MethodPost, "/reminders/add", nil)
	addReq.PostForm = addForm
	addRec := httptest.NewRecorder()

	app.AddReminderHandler(addRec, addReq)

	if addRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for AddReminderHandler, got %d (Trigger: %s)", addRec.Code, addRec.Header().Get("HX-Trigger"))
	}

	// Find the created reminder
	var createdRem models.Reminder
	err = db.DB.Get(&createdRem, `
		SELECT 
			r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
			r.assigned_to_user_id,
			COALESCE(u.full_name, u.username, '') AS assigned_to_name
		FROM reminders r
		LEFT JOIN users u ON r.assigned_to_user_id = u.id
		WHERE r.title = 'Test Assignment Reminder'
		ORDER BY r.id DESC LIMIT 1
	`)
	if err != nil {
		t.Fatalf("Failed to query created reminder: %v", err)
	}
	if createdRem.AssignedToUserID == nil || *createdRem.AssignedToUserID != user.ID {
		t.Fatalf("Expected reminder to be assigned to user ID %d, got %v", user.ID, createdRem.AssignedToUserID)
	}
	if createdRem.AssignedToName == "" {
		t.Fatalf("Expected reminder assigned user name to be non-empty")
	}

	// Clean up created reminder at end
	defer func() {
		_, _ = db.DB.Exec("DELETE FROM reminders WHERE id = ?", createdRem.ID)
	}()

	// 2. Edit the reminder (change title, details, priority, and unassign/reassign)
	editForm := url.Values{}
	editForm.Set("title", "Updated Assignment Reminder")
	editForm.Set("details", "Updated notes after editing")
	editForm.Set("due_date", time.Now().AddDate(0, 0, 3).Format("2006-01-02"))
	editForm.Set("priority", "Medium")
	editForm.Set("category", "General")
	editForm.Set("status", "Completed")
	editForm.Set("assigned_to_user_id", "") // Unassign

	editReq := httptest.NewRequest(http.MethodPost, "/reminders/edit/"+fmt.Sprintf("%d", createdRem.ID), nil)
	editReq.SetPathValue("id", fmt.Sprintf("%d", createdRem.ID))
	editReq.PostForm = editForm
	editRec := httptest.NewRecorder()

	app.EditReminderHandler(editRec, editReq)

	if editRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for EditReminderHandler, got %d", editRec.Code)
	}

	// Verify update in DB
	var updatedRem models.Reminder
	err = db.DB.Get(&updatedRem, `
		SELECT 
			r.id, r.title, r.details, r.due_date, r.priority, r.category, r.status, r.created_at,
			r.assigned_to_user_id,
			COALESCE(u.full_name, u.username, '') AS assigned_to_name
		FROM reminders r
		LEFT JOIN users u ON r.assigned_to_user_id = u.id
		WHERE r.id = ?
	`, createdRem.ID)
	if err != nil {
		t.Fatalf("Failed to query updated reminder: %v", err)
	}
	if updatedRem.Title != "Updated Assignment Reminder" {
		t.Fatalf("Expected updated title, got %s", updatedRem.Title)
	}
	if updatedRem.Priority != "Medium" {
		t.Fatalf("Expected Medium priority, got %s", updatedRem.Priority)
	}
	if updatedRem.Status != "Completed" {
		t.Fatalf("Expected Completed status, got %s", updatedRem.Status)
	}
	if updatedRem.AssignedToUserID != nil {
		t.Fatalf("Expected assigned user to be nil after unassigning, got %v", updatedRem.AssignedToUserID)
	}

	// 3. Render Calendar view to ensure template rendering passes with assigned data
	calReq := httptest.NewRequest(http.MethodGet, "/calendar", nil)
	calRec := httptest.NewRecorder()
	app.RemindersHandler(calRec, calReq)

	if calRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for RemindersHandler, got %d", calRec.Code)
	}
}
