package domain

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Standard Permission Keys
const (
	PermDashboardView   = "dashboard:view"
	PermSalesView       = "sales:view"
	PermSalesCreate     = "sales:create"
	PermSalesEdit       = "sales:edit"
	PermSalesDelete     = "sales:delete"
	PermInventoryView   = "inventory:view"
	PermInventoryReceive = "inventory:receive"
	PermInventoryAdjust = "inventory:adjust"
	PermItemsView       = "items:view"
	PermItemsEdit       = "items:edit"
	PermSettingsView    = "settings:view"
	PermSettingsEdit    = "settings:edit"
	PermSettingsRoles   = "settings:roles"
	PermSettingsUsers   = "settings:users"
	PermLogsView        = "logs:view"
	PermToolsImport     = "tools:import"
	PermToolsScanner    = "tools:scanner"
)

// AllPermissions returns the comprehensive catalog of system permissions with descriptions
type PermissionDef struct {
	Key         string
	Name        string
	Category    string
	Description string
}

func GetSystemPermissions() []PermissionDef {
	return []PermissionDef{
		{Key: PermDashboardView, Name: "View Dashboard", Category: "Dashboard", Description: "Access to main financial dashboard and analytics"},
		{Key: PermSalesView, Name: "View Sales", Category: "Sales", Description: "Access to sales list and invoice details"},
		{Key: PermSalesCreate, Name: "Create Sales", Category: "Sales", Description: "Issue and record new sales transactions"},
		{Key: PermSalesEdit, Name: "Edit Sales", Category: "Sales", Description: "Modify existing sales transactions"},
		{Key: PermSalesDelete, Name: "Delete Sales", Category: "Sales", Description: "Delete or cancel sales transactions"},
		{Key: PermInventoryView, Name: "View Inventory", Category: "Inventory", Description: "View stock levels and monthly inventory reports"},
		{Key: PermInventoryReceive, Name: "Receive Stock", Category: "Inventory", Description: "Log incoming items and supplier deliveries"},
		{Key: PermInventoryAdjust, Name: "Adjust Stock", Category: "Inventory", Description: "Perform inventory count reconciliation & adjustments"},
		{Key: PermItemsView, Name: "View Items", Category: "Masterlist", Description: "Browse catalog of items and product details"},
		{Key: PermItemsEdit, Name: "Manage Items", Category: "Masterlist", Description: "Create, update, or delete items"},
		{Key: PermSettingsView, Name: "View Settings", Category: "Settings", Description: "Access system configuration and master data lists"},
		{Key: PermSettingsEdit, Name: "Edit Settings", Category: "Settings", Description: "Update system settings, brands, categories, and UOMs"},
		{Key: PermSettingsRoles, Name: "Manage Roles", Category: "Security", Description: "Create, edit, or configure user roles and permissions"},
		{Key: PermSettingsUsers, Name: "Manage Users", Category: "Security", Description: "Create, edit, activate, or deactivate user accounts"},
		{Key: PermLogsView, Name: "View Activity Logs", Category: "Security", Description: "Inspect audit trail and user action history"},
		{Key: PermToolsImport, Name: "Data Import", Category: "Tools", Description: "Upload and import Excel data files"},
		{Key: PermToolsScanner, Name: "Receipt Scanner", Category: "Tools", Description: "Access AI receipt scanning and OCR parsing"},
	}
}

// User represents an authenticated operator in the system
type User struct {
	ID           int64     `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	FullName     string    `json:"full_name" db:"full_name"`
	RoleID       int64     `json:"role_id" db:"role_id"`
	RoleName     string    `json:"role_name" db:"role_name"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Role represents an authorization tier grouping permissions
type Role struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Permissions []string  `json:"permissions" db:"-"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// HasPermission checks if a role contains a specific permission key.
// "Admin" role always evaluates to true for all permissions.
func (r *Role) HasPermission(permissionKey string) bool {
	if strings.EqualFold(r.Name, "Admin") {
		return true
	}
	for _, p := range r.Permissions {
		if p == permissionKey {
			return true
		}
	}
	return false
}

// ActivityLog represents an audit log entry tracking user operations
type ActivityLog struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Username   string    `json:"username" db:"username"`
	Action     string    `json:"action" db:"action"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   string    `json:"entity_id" db:"entity_id"`
	Details    string    `json:"details" db:"details"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Password helpers
func HashPassword(password string) (string, error) {
	if len(password) < 4 {
		return "", errors.New("password must be at least 4 characters")
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Domain validations
func ValidateUsername(username string) error {
	trimmed := strings.TrimSpace(username)
	if len(trimmed) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if strings.ContainsAny(trimmed, " \t\r\n/\\") {
		return errors.New("username cannot contain whitespace or slashes")
	}
	return nil
}

func ValidateRole(name string) error {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < 2 {
		return errors.New("role name must be at least 2 characters")
	}
	return nil
}
