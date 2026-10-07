package domain_test

import (
	"testing"

	"radline/domain"
)

func TestRoleHasPermission(t *testing.T) {
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

	// Too short password
	_, errShort := domain.HashPassword("12")
	if errShort == nil {
		t.Errorf("expected error for too short password")
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

func TestGetSystemPermissions(t *testing.T) {
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
}
