package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"rav/db"
	"rav/domain"
	"rav/models"
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
		log.Printf("RolesHandler select roles error: %v", err)
		http.Error(w, "Failed to load roles", http.StatusInternalServerError)
		return
	}

	// Single query for all role_permissions to avoid N+1 queries
	var allPerms []models.RolePermission
	if err := db.DB.Select(&allPerms, "SELECT role_id, permission_key FROM role_permissions"); err != nil {
		log.Printf("RolesHandler select role_permissions error: %v", err)
		http.Error(w, "Failed to load role permissions", http.StatusInternalServerError)
		return
	}

	permsByRole := make(map[int64]map[string]bool, len(roles))
	for _, rp := range allPerms {
		if permsByRole[rp.RoleID] == nil {
			permsByRole[rp.RoleID] = make(map[string]bool)
		}
		permsByRole[rp.RoleID][rp.PermissionKey] = true
	}

	var rolesWithDetails []RoleWithDetails
	for _, rle := range roles {
		permMap := permsByRole[rle.ID]
		if permMap == nil {
			permMap = make(map[string]bool)
		}
		if strings.EqualFold(rle.Name, "Admin") {
			for _, p := range domain.GetSystemPermissions() {
				permMap[p.Key] = true
			}
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
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	selectedPerms := r.Form["permissions"]
	for _, p := range selectedPerms {
		if !domain.IsValidPermission(p) {
			setHXTriggerToast(w, "error", fmt.Sprintf("Invalid permission key: %s", p))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("AddRoleHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to start transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var roleID int64
	err = tx.QueryRow("INSERT INTO roles (name, description) VALUES (?, ?) RETURNING id", name, desc).Scan(&roleID)
	if err != nil {
		setHXTriggerToast(w, "error", "Failed to create role: name may already exist")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, p := range selectedPerms {
		permKey := strings.TrimSpace(p)
		if _, err := tx.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", roleID, permKey); err != nil {
			log.Printf("AddRoleHandler insert permission error: %v", err)
			setHXTriggerToast(w, "error", "Failed to assign role permissions")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("AddRoleHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit role creation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "CREATE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Created role %s with %d permissions", name, len(selectedPerms)))
	setHXTriggerToast(w, "success", "Role created successfully")
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

	// Prevent renaming the Admin role or renaming another role to Admin
	if strings.EqualFold(existing.Name, "Admin") {
		if !strings.EqualFold(name, "Admin") {
			setHXTriggerToast(w, "error", "The system Admin role cannot be renamed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		if err := domain.ValidateRole(name); err != nil {
			setHXTriggerToast(w, "error", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	selectedPerms := r.Form["permissions"]
	for _, p := range selectedPerms {
		if !domain.IsValidPermission(p) {
			setHXTriggerToast(w, "error", fmt.Sprintf("Invalid permission key: %s", p))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("UpdateRoleHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to start transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE roles SET name = ?, description = ? WHERE id = ?", name, desc, roleID)
	if err != nil {
		setHXTriggerToast(w, "error", "Failed to update role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Admin always holds all permissions
	if !strings.EqualFold(name, "Admin") {
		if _, err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID); err != nil {
			log.Printf("UpdateRoleHandler delete permissions error: %v", err)
			setHXTriggerToast(w, "error", "Failed to update role permissions")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, p := range selectedPerms {
			permKey := strings.TrimSpace(p)
			if _, err := tx.Exec("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?) ON CONFLICT DO NOTHING", roleID, permKey); err != nil {
				log.Printf("UpdateRoleHandler insert permission error: %v", err)
				setHXTriggerToast(w, "error", "Failed to assign role permissions")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("UpdateRoleHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit role update")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "UPDATE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Updated role %s and permissions", name))
	setHXTriggerToast(w, "success", "Role updated successfully")
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
		setHXTriggerToast(w, "error", "Cannot delete the default Admin role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("DeleteRoleHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to start transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userCount int
	if err := tx.Get(&userCount, "SELECT COUNT(*) FROM users WHERE role_id = ?", roleID); err != nil {
		log.Printf("DeleteRoleHandler count users error: %v", err)
		setHXTriggerToast(w, "error", "Failed to verify role usage")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if userCount > 0 {
		setHXTriggerToast(w, "error", fmt.Sprintf("Cannot delete role: %d user(s) are currently assigned to it", userCount))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID); err != nil {
		log.Printf("DeleteRoleHandler delete permissions error: %v", err)
		setHXTriggerToast(w, "error", "Failed to delete role permissions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("DELETE FROM roles WHERE id = ?", roleID); err != nil {
		log.Printf("DeleteRoleHandler delete role error: %v", err)
		setHXTriggerToast(w, "error", "Failed to delete role")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("DeleteRoleHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit role deletion")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "DELETE_ROLE", "Role", fmt.Sprintf("%d", roleID), fmt.Sprintf("Deleted role %s", role.Name))
	setHXTriggerToast(w, "success", "Role deleted successfully")
	w.Header().Set("HX-Redirect", "/settings/roles")
	w.WriteHeader(http.StatusOK)
}

// UsersHandler lists all operators and supports user administration
func (app *App) UsersHandler(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	err := db.DB.Select(&users, `
		SELECT u.id, u.username, u.full_name, u.role_id, r.name as role_name, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.id ASC
	`)
	if err != nil {
		log.Printf("UsersHandler select users error: %v", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
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
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if fullName == "" {
		setHXTriggerToast(w, "error", "Full name is required")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := domain.HashPassword(password)
	if err != nil {
		setHXTriggerToast(w, "error", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roleID, _ := strconv.ParseInt(roleIDStr, 10, 64)
	if roleID <= 0 {
		setHXTriggerToast(w, "error", "Please select a valid role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var targetRole models.Role
	if err := db.DB.Get(&targetRole, "SELECT id, name FROM roles WHERE id = ?", roleID); err != nil {
		setHXTriggerToast(w, "error", "Please select a valid role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if currUser := GetCurrentUser(r); currUser != nil && !strings.EqualFold(currUser.User.RoleName, "Admin") {
		if strings.EqualFold(targetRole.Name, "Admin") {
			setHXTriggerToast(w, "error", "Only an Admin can assign the Admin role")
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	var newUserID int64
	err = db.DB.QueryRow(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES (?, ?, ?, ?, TRUE)
		RETURNING id
	`, username, hash, fullName, roleID).Scan(&newUserID)
	if err != nil {
		setHXTriggerToast(w, "error", "Username already exists")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	app.LogActivity(r, "CREATE_USER", "User", fmt.Sprintf("%d", newUserID), fmt.Sprintf("Created user account %s (%s)", username, fullName))
	setHXTriggerToast(w, "success", "User account created successfully")
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

	if fullName == "" {
		setHXTriggerToast(w, "error", "Full name is required")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roleID, _ := strconv.ParseInt(roleIDStr, 10, 64)
	if roleID <= 0 {
		setHXTriggerToast(w, "error", "Please select a valid role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var existingUser models.User
	err = db.DB.Get(&existingUser, `
		SELECT u.id, u.username, u.full_name, u.role_id, r.name as role_name, u.is_active
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = ?
	`, userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var targetRole models.Role
	if err := db.DB.Get(&targetRole, "SELECT id, name FROM roles WHERE id = ?", roleID); err != nil {
		setHXTriggerToast(w, "error", "Please select a valid role")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	targetWasAdmin := strings.EqualFold(existingUser.RoleName, "Admin")
	targetWillBeAdmin := strings.EqualFold(targetRole.Name, "Admin")

	if currUser := GetCurrentUser(r); currUser != nil && !strings.EqualFold(currUser.User.RoleName, "Admin") {
		if targetWasAdmin || targetWillBeAdmin {
			setHXTriggerToast(w, "error", "Only an Admin can modify or assign Admin users")
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("UpdateUserHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to start transaction")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Prevent deactivating or demoting the last active Admin user
	if targetWasAdmin && existingUser.IsActive && (!isActive || !targetWillBeAdmin) {
		var otherActiveAdmins int
		if err := tx.Get(&otherActiveAdmins, `
			SELECT COUNT(*)
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE LOWER(r.name) = 'admin' AND u.is_active = TRUE AND u.id != ?
		`, userID); err != nil {
			log.Printf("UpdateUserHandler count admins error: %v", err)
			setHXTriggerToast(w, "error", "Failed to verify Admin count")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if otherActiveAdmins == 0 {
			setHXTriggerToast(w, "error", "Cannot deactivate or demote the last active Admin user")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if newPassword != "" {
		hash, err := domain.HashPassword(newPassword)
		if err != nil {
			setHXTriggerToast(w, "error", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = tx.Exec("UPDATE users SET full_name = ?, role_id = ?, is_active = ?, password_hash = ? WHERE id = ?", fullName, roleID, isActive, hash, userID)
	} else {
		_, err = tx.Exec("UPDATE users SET full_name = ?, role_id = ?, is_active = ? WHERE id = ?", fullName, roleID, isActive, userID)
	}

	if err != nil {
		log.Printf("UpdateUserHandler update error: %v", err)
		setHXTriggerToast(w, "error", "Failed to update user")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Revoke all active sessions immediately when password changes or user is deactivated
	if newPassword != "" || !isActive {
		if _, err := tx.Exec("DELETE FROM user_sessions WHERE user_id = ?", userID); err != nil {
			log.Printf("UpdateUserHandler delete sessions error: %v", err)
			setHXTriggerToast(w, "error", "Failed to revoke active sessions")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("UpdateUserHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit user update")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app.LogActivity(r, "UPDATE_USER", "User", fmt.Sprintf("%d", userID), fmt.Sprintf("Updated user account ID %d (%s)", userID, fullName))
	setHXTriggerToast(w, "success", "User updated successfully")
	w.Header().Set("HX-Redirect", "/settings/users")
	w.WriteHeader(http.StatusOK)
}
