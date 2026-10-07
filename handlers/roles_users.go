package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"radline/db"
	"radline/domain"
	"radline/models"
)

type RoleWithDetails struct {
	models.Role
	Permissions map[string]bool
}

// RolesHandler renders the roles and permissions management page
func (app *App) RolesHandler(w http.ResponseWriter, r *http.Request) {
	var roles []models.Role
	err := db.DB.Select(&roles, `
		SELECT r.id, r.name, r.description, r.created_at, COUNT(u.id) as user_count
		FROM roles r
		LEFT JOIN users u ON r.id = u.role_id
		GROUP BY r.id, r.name, r.description, r.created_at
		ORDER BY r.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var rolesWithDetails []RoleWithDetails
	for _, rle := range roles {
		var permKeys []string
		_ = db.DB.Select(&permKeys, "SELECT permission_key FROM role_permissions WHERE role_id = ?", rle.ID)
		permMap := make(map[string]bool)
		for _, k := range permKeys {
			permMap[k] = true
		}
		rolesWithDetails = append(rolesWithDetails, RoleWithDetails{
			Role:        rle,
			Permissions: permMap,
		})
	}

	data := struct {
		Roles       []RoleWithDetails
		Permissions []domain.PermissionDef
	}{
		Roles:       rolesWithDetails,
		Permissions: domain.GetSystemPermissions(),
	}

	app.RenderPage(w, r, "roles.html", data)
}

// AddRoleHandler creates a new role with assigned permissions
func (app *App) AddRoleHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	if err := domain.ValidateRole(name); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := db.DB.Exec("INSERT INTO roles (name, description) VALUES (?, ?)", name, desc)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to create role: name may already exist"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	roleID, _ := res.LastInsertId()

	selectedPerms := r.Form["permissions"]
	for _, p := range selectedPerms {
		_, _ = db.DB.Exec("INSERT OR IGNORE INTO role_permissions (role_id, permission_key) VALUES (?, ?)", roleID, p)
	}

	app.LogActivity(r, "CREATE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Created role %s with %d permissions", name, len(selectedPerms)))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Role created successfully"}}`)
	w.Header().Set("HX-Redirect", "/settings/roles")
	w.WriteHeader(http.StatusOK)
}

// UpdateRoleHandler updates an existing role and its permissions
func (app *App) UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	roleID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	var existing models.Role
	err = db.DB.Get(&existing, "SELECT id, name, description FROM roles WHERE id = ?", roleID)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("description"))
	if err := domain.ValidateRole(name); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Prevent renaming the Admin role
	if strings.EqualFold(existing.Name, "Admin") && !strings.EqualFold(name, "Admin") {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "The system Admin role cannot be renamed"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE roles SET name = ?, description = ? WHERE id = ?", name, desc, roleID)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update role"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Admin always holds all permissions
	if !strings.EqualFold(name, "Admin") {
		_, _ = db.DB.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID)
		selectedPerms := r.Form["permissions"]
		for _, p := range selectedPerms {
			_, _ = db.DB.Exec("INSERT OR IGNORE INTO role_permissions (role_id, permission_key) VALUES (?, ?)", roleID, p)
		}
	}

	app.LogActivity(r, "UPDATE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Updated role %s and permissions", name))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Role updated successfully"}}`)
	w.Header().Set("HX-Redirect", "/settings/roles")
	w.WriteHeader(http.StatusOK)
}

// DeleteRoleHandler removes a role if not Admin and has no assigned users
func (app *App) DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	roleID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	var role models.Role
	err = db.DB.Get(&role, "SELECT id, name FROM roles WHERE id = ?", roleID)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	if strings.EqualFold(role.Name, "Admin") {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Cannot delete the default Admin role"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var userCount int
	_ = db.DB.Get(&userCount, "SELECT COUNT(*) FROM users WHERE role_id = ?", roleID)
	if userCount > 0 {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "Cannot delete role: %d user(s) are currently assigned to it"}}`, userCount))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, _ = db.DB.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID)
	_, _ = db.DB.Exec("DELETE FROM roles WHERE id = ?", roleID)

	app.LogActivity(r, "DELETE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Deleted role %s", role.Name))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "Role deleted successfully"}}`)
	w.Header().Set("HX-Redirect", "/settings/roles")
	w.WriteHeader(http.StatusOK)
}

// UsersHandler lists all operators and supports user administration
func (app *App) UsersHandler(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	err := db.DB.Select(&users, `
		SELECT u.id, u.username, u.password_hash, u.full_name, u.role_id, r.name as role_name, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.id ASC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var roles []models.Role
	_ = db.DB.Select(&roles, "SELECT id, name, description FROM roles ORDER BY name ASC")

	data := struct {
		Users []models.User
		Roles []models.Role
	}{
		Users: users,
		Roles: roles,
	}

	app.RenderPage(w, r, "users.html", data)
}

// AddUserHandler creates a new user account
func (app *App) AddUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	password := r.FormValue("password")
	roleIDStr := r.FormValue("role_id")

	if err := domain.ValidateUsername(username); err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if fullName == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Full name is required"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := domain.HashPassword(password)
	if err != nil {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roleID, _ := strconv.ParseInt(roleIDStr, 10, 64)
	if roleID <= 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Please select a valid role"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res, err := db.DB.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES (?, ?, ?, ?, 1)
	`, username, hash, fullName, roleID)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Username already exists"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newUserID, _ := res.LastInsertId()
	app.LogActivity(r, "CREATE_USER", "User", fmt.Sprintf("%d", newUserID), fmt.Sprintf("Created user account %s (%s)", username, fullName))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "User account created successfully"}}`)
	w.Header().Set("HX-Redirect", "/settings/users")
	w.WriteHeader(http.StatusOK)
}

// UpdateUserHandler updates user information or resets password
func (app *App) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	roleIDStr := r.FormValue("role_id")
	isActive := r.FormValue("is_active") == "1" || r.FormValue("is_active") == "true"
	newPassword := r.FormValue("new_password")

	roleID, _ := strconv.ParseInt(roleIDStr, 10, 64)
	if roleID <= 0 {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Please select a valid role"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if newPassword != "" {
		hash, err := domain.HashPassword(newPassword)
		if err != nil {
			w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "error", "message": "%s"}}`, err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = db.DB.Exec("UPDATE users SET full_name = ?, role_id = ?, is_active = ?, password_hash = ? WHERE id = ?", fullName, roleID, isActive, hash, userID)
	} else {
		_, err = db.DB.Exec("UPDATE users SET full_name = ?, role_id = ?, is_active = ? WHERE id = ?", fullName, roleID, isActive, userID)
	}

	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to update user"}}`)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	app.LogActivity(r, "UPDATE_USER", "User", fmt.Sprintf("%d", userID), fmt.Sprintf("Updated user account ID %d (%s)", userID, fullName))
	w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "User updated successfully"}}`)
	w.Header().Set("HX-Redirect", "/settings/users")
	w.WriteHeader(http.StatusOK)
}
