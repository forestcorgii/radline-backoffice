package domain_test

import (
	"strings"
	"testing"

	"rav/domain"
)

func TestRoleHasPermission(t *testing.T) {
	// Nil role is safe
	var nilRole *domain.Role
	if nilRole.HasPermission(domain.PermDashboardView) {
		t.Errorf("expected nil Role NOT to have any permission")
	}

	// Admin has all permissions automatically
	adminRole := domain.Role{
		Name:        "Admin",
		Permissions: []string{},
	}
	if !adminRole.HasPermission(domain.PermDashboardView) {
		t.Errorf("expected Admin to have PermDashboardView")
	}
	if !adminRole.HasPermission(domain.PermSettingsRoles) {
		t.Errorf("expected Admin to have PermSettingsRoles")
	}

	// Custom role only has assigned permissions
	staffRole := domain.Role{
		Name: "Staff",
		Permissions: []string{
			domain.PermSalesView,
			domain.PermSalesCreate,
		},
	}
	if !staffRole.HasPermission(domain.PermSalesView) {
		t.Errorf("expected Staff to have PermSalesView")
	}
	if !staffRole.HasPermission(domain.PermSalesCreate) {
		t.Errorf("expected Staff to have PermSalesCreate")
	}
	if staffRole.HasPermission(domain.PermSalesDelete) {
		t.Errorf("expected Staff NOT to have PermSalesDelete")
	}
	if staffRole.HasPermission(domain.PermSettingsRoles) {
		t.Errorf("expected Staff NOT to have PermSettingsRoles")
	}
}

func TestPasswordHashingAndVerification(t *testing.T) {
	password := "Secret123!"
	hash, err := domain.HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if !domain.CheckPasswordHash(password, hash) {
		t.Errorf("expected valid password verification to succeed")
	}

	if domain.CheckPasswordHash("WrongPassword", hash) {
		t.Errorf("expected invalid password verification to fail")
	}

	// Too short password (< 8 chars)
	_, errShort := domain.HashPassword("1234567")
	if errShort == nil {
		t.Errorf("expected error for password shorter than 8 characters")
	}

	// Whitespace-padded short password
	_, errSpace := domain.HashPassword("   1234   ")
	if errSpace == nil {
		t.Errorf("expected error for whitespace-padded password shorter than 8 characters")
	}

	// Too long password (> 72 bytes)
	_, errLong := domain.HashPassword(strings.Repeat("a", 73))
	if errLong == nil {
		t.Errorf("expected error for password exceeding 72 bytes")
	}
}

func TestValidateUsername(t *testing.T) {
	if err := domain.ValidateUsername("admin"); err != nil {
		t.Errorf("expected valid username 'admin', got error: %v", err)
	}

	if err := domain.ValidateUsername("ab"); err == nil {
		t.Errorf("expected error for username shorter than 3 chars")
	}

	if err := domain.ValidateUsername("admin user"); err == nil {
		t.Errorf("expected error for username containing space")
	}
}

func TestValidateRole(t *testing.T) {
	if err := domain.ValidateRole("Manager"); err != nil {
		t.Errorf("expected 'Manager' to be valid, got: %v", err)
	}
	if err := domain.ValidateRole("a"); err == nil {
		t.Errorf("expected error for 1-char role name")
	}
	if err := domain.ValidateRole("Admin"); err == nil {
		t.Errorf("expected error for reserved role name 'Admin'")
	}
	if err := domain.ValidateRole("  admin  "); err == nil {
		t.Errorf("expected error for reserved role name 'admin' (case-insensitive)")
	}
}

func TestGetSystemPermissionsAndIsValidPermission(t *testing.T) {
	perms := domain.GetSystemPermissions()
	if len(perms) == 0 {
		t.Errorf("expected non-empty list of system permissions")
	}

	foundDashboard := false
	for _, p := range perms {
		if p.Key == domain.PermDashboardView {
			foundDashboard = true
			break
		}
	}
	if !foundDashboard {
		t.Errorf("expected PermDashboardView to be in catalog")
	}

	if !domain.IsValidPermission(domain.PermSalesView) {
		t.Errorf("expected PermSalesView to be valid")
	}
	if domain.IsValidPermission("unknown:perm") {
		t.Errorf("expected 'unknown:perm' to be invalid")
	}
	if domain.IsValidPermission("") {
		t.Errorf("expected empty permission key to be invalid")
	}
}
