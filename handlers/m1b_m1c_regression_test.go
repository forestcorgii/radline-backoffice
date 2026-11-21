package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"rav/db"
	"rav/domain"
	"rav/models"
)

func TestM1B_SSRF_ValidateAIProviderURL(t *testing.T) {
	t.Run("Allowed default HTTPS hosts", func(t *testing.T) {
		validURLs := []string{
			"https://api.deepseek.com/v1",
			"https://api.openai.com/v1",
			"https://openrouter.ai/api/v1",
			"https://api.groq.com/openai/v1",
			"https://api.anthropic.com/v1",
			"https://generativelanguage.googleapis.com/v1beta",
		}
		for _, u := range validURLs {
			if err := ValidateAIProviderURL(u); err != nil {
				t.Errorf("Expected %q to be valid, got error: %v", u, err)
			}
		}
	})

	t.Run("Custom DEEPSEEK_ALLOWED_HOSTS", func(t *testing.T) {
		t.Setenv("DEEPSEEK_ALLOWED_HOSTS", "api.siliconflow.cn, custom.llm.example.com")
		if err := ValidateAIProviderURL("https://api.siliconflow.cn/v1"); err != nil {
			t.Errorf("Expected custom allowed host to pass, got: %v", err)
		}
	})

	t.Run("Rejected SSRF & non-HTTPS URLs", func(t *testing.T) {
		invalidURLs := []string{
			"",
			"http://api.deepseek.com/v1",
			"https://user:pass@api.deepseek.com/v1",
			"https://localhost:8080/v1",
			"https://127.0.0.1/v1",
			"https://[::1]/v1",
			"https://169.254.169.254/latest/meta-data/",
			"https://10.0.0.1/v1",
			"https://192.168.1.1/v1",
			"https://172.16.0.1/v1",
			"https://0.0.0.0/v1",
			"https://evil.attacker.com/v1",
			"file:///etc/passwd",
		}
		for _, u := range invalidURLs {
			if err := ValidateAIProviderURL(u); err == nil {
				t.Errorf("Expected %q to be rejected by ValidateAIProviderURL, but it passed", u)
			}
		}
	})
}

func TestM1B_SaveDeepSeekConfig_SSRFAndKeyPreservation(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	// 1. Save valid initial config
	form := url.Values{}
	form.Set("api_key", "sk-real-secret-key-999")
	form.Set("api_base", "https://api.deepseek.com/v1")
	form.Set("model", "deepseek-chat")

	req := httptest.NewRequest(http.MethodPost, "/settings/deepseek", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.SaveDeepSeekConfigHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK saving valid config, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := db.GetSystemSetting("deepseek_api_key"); got != "sk-real-secret-key-999" {
		t.Fatalf("Expected saved key 'sk-real-secret-key-999', got %q", got)
	}

	// 2. Reject SSRF URL in SaveDeepSeekConfigHandler
	badForm := url.Values{}
	badForm.Set("api_key", "sk-new")
	badForm.Set("api_base", "http://169.254.169.254/latest/meta-data")
	badForm.Set("model", "deepseek-chat")

	badReq := httptest.NewRequest(http.MethodPost, "/settings/deepseek", strings.NewReader(badForm.Encode()))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badRec := httptest.NewRecorder()
	app.SaveDeepSeekConfigHandler(badRec, badReq)

	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for SSRF api_base, got %d", badRec.Code)
	}
	// Ensure existing settings were not overwritten
	if got := db.GetSystemSetting("deepseek_api_base"); got != "https://api.deepseek.com/v1" {
		t.Errorf("Expected api_base to remain 'https://api.deepseek.com/v1', got %q", got)
	}

	// 3. Preserve existing API key when blank or masked bullets are submitted
	for _, preservedInput := range []string{"", "••••••••", MaskedAPIKeyPlaceholder} {
		preserveForm := url.Values{}
		preserveForm.Set("api_key", preservedInput)
		preserveForm.Set("api_base", "https://openrouter.ai/api/v1")
		preserveForm.Set("model", "deepseek/deepseek-chat")

		pReq := httptest.NewRequest(http.MethodPost, "/settings/deepseek", strings.NewReader(preserveForm.Encode()))
		pReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		pRec := httptest.NewRecorder()
		app.SaveDeepSeekConfigHandler(pRec, pReq)

		if pRec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK when preserving key (input=%q), got %d", preservedInput, pRec.Code)
		}
		if got := db.GetSystemSetting("deepseek_api_key"); got != "sk-real-secret-key-999" {
			t.Fatalf("Expected API key to be preserved as 'sk-real-secret-key-999' when input=%q, got %q", preservedInput, got)
		}
	}

	// 4. Verify masked API key in view models and HTML rendering
	tmpl := template.Must(template.New("system_settings.html").Funcs(TemplateFuncMap()).ParseFiles("../templates/base.html", "../templates/system_settings.html"))
	scannerTmpl := template.Must(template.New("receipt_scanner.html").Funcs(TemplateFuncMap()).ParseFiles("../templates/base.html", "../templates/receipt_scanner.html"))
	app.Templates = map[string]*template.Template{
		"system_settings.html": tmpl,
		"receipt_scanner.html": scannerTmpl,
	}

	sysReq := httptest.NewRequest(http.MethodGet, "/settings/system", nil)
	sysRec := httptest.NewRecorder()
	app.SystemSettingsHandler(sysRec, sysReq)
	if sysRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from SystemSettingsHandler, got %d", sysRec.Code)
	}
	if strings.Contains(sysRec.Body.String(), "sk-real-secret-key-999") {
		t.Fatalf("Plaintext API key leaked in /settings/system HTML response!")
	}

	scanReq := httptest.NewRequest(http.MethodGet, "/tools/receipt-scanner", nil)
	scanRec := httptest.NewRecorder()
	app.ReceiptScannerHandler(scanRec, scanReq)
	if scanRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from ReceiptScannerHandler, got %d", scanRec.Code)
	}
	if strings.Contains(scanRec.Body.String(), "sk-real-secret-key-999") {
		t.Fatalf("Plaintext API key leaked in /tools/receipt-scanner HTML response!")
	}
}

func TestM1B_AuthRateLimitingAndUniform401(t *testing.T) {
	setupIsolatedTestDB(t)
	ResetLoginRateLimiter()
	defer ResetLoginRateLimiter()

	app := &App{}

	hash, err := domain.HashPassword("correct-password-123")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.DB.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES ('inactive_op', ?, 'Inactive Operator', 1, FALSE)
	`, hash)
	if err != nil {
		t.Fatalf("Failed to insert inactive user: %v", err)
	}

	// 1. Inactive account returns 401 with uniform message (no account status enumeration)
	inactForm := url.Values{}
	inactForm.Set("username", "inactive_op")
	inactForm.Set("password", "correct-password-123")
	inactReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(inactForm.Encode()))
	inactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	inactRec := httptest.NewRecorder()
	app.LoginSubmitHandler(inactRec, inactReq)

	if inactRec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for inactive user, got %d", inactRec.Code)
	}
	if strings.Contains(strings.ToLower(inactRec.Body.String()), "deactivated") {
		t.Errorf("Expected uniform error message without 'deactivated', got: %s", inactRec.Body.String())
	}

	// 2. Rate limiting after 10 failed attempts
	ResetLoginRateLimiter()
	for i := 1; i <= 10; i++ {
		f := url.Values{}
		f.Set("username", "admin")
		f.Set("password", "wrong-pass")
		r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.RemoteAddr = "198.51.100.42:12345"
		w := httptest.NewRecorder()
		app.LoginSubmitHandler(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Attempt %d: expected 401, got %d", i, w.Code)
		}
	}

	// 11th attempt must return 429 Too Many Requests
	f11 := url.Values{}
	f11.Set("username", "admin")
	f11.Set("password", "wrong-pass")
	r11 := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(f11.Encode()))
	r11.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r11.RemoteAddr = "198.51.100.42:12345"
	w11 := httptest.NewRecorder()
	app.LoginSubmitHandler(w11, r11)
	if w11.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected 429 Too Many Requests on 11th attempt, got %d", w11.Code)
	}
}

func TestM1B_UserChipRuneSlicingAndLogoutCookies(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	// 1. Multi-byte UTF-8 rune initial in UserChipHandler
	ctxUser := &CurrentUserContext{
		User: models.User{
			ID:       1,
			Username: "ñino",
			FullName: "Ñoño Reyes",
			RoleName: "Admin",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/auth/user-chip", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, ctxUser))
	rec := httptest.NewRecorder()
	app.UserChipHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ">Ñ<") && !strings.Contains(body, "Ñ\n") {
		t.Errorf("Expected multi-byte rune initial 'Ñ' in UserChipHandler output, got: %s", body)
	}

	// 2. LogoutHandler clears both rav_session and radline_session with SameSite=Lax
	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.Header.Set("X-Forwarded-Proto", "https")
	logoutRec := httptest.NewRecorder()
	app.LogoutHandler(logoutRec, logoutReq)

	cookies := logoutRec.Result().Cookies()
	foundRav := false
	foundLegacy := false
	for _, c := range cookies {
		if c.Name == SessionCookieName {
			foundRav = true
			if c.MaxAge != -1 || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Errorf("Unexpected cookie attributes on %s: %+v", c.Name, c)
			}
		}
		if c.Name == LegacySessionCookieName {
			foundLegacy = true
			if c.MaxAge != -1 || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
				t.Errorf("Unexpected cookie attributes on %s: %+v", c.Name, c)
			}
		}
	}
	if !foundRav || !foundLegacy {
		t.Errorf("Expected both %s and %s cookies to be cleared, got foundRav=%v foundLegacy=%v",
			SessionCookieName, LegacySessionCookieName, foundRav, foundLegacy)
	}
}

func TestM1B_RolesUsersPrivilegeEscalationAndSessionRevocation(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	// 1. AddRoleHandler rejects unknown permission key
	badRoleForm := url.Values{}
	badRoleForm.Set("name", "BogusRole")
	badRoleForm.Set("description", "Role with fake perm")
	badRoleForm.Add("permissions", "dashboard:view")
	badRoleForm.Add("permissions", "superadmin:pwn")

	badRoleReq := httptest.NewRequest(http.MethodPost, "/settings/roles/add", strings.NewReader(badRoleForm.Encode()))
	badRoleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badRoleRec := httptest.NewRecorder()
	app.AddRoleHandler(badRoleRec, badRoleReq)
	if badRoleRec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for invalid permission key, got %d", badRoleRec.Code)
	}

	// 2. Non-Admin with PermSettingsUsers cannot create an Admin user
	var adminRoleID, staffRoleID int64
	_ = db.DB.Get(&adminRoleID, "SELECT id FROM roles WHERE name = 'Admin'")
	_ = db.DB.Get(&staffRoleID, "SELECT id FROM roles WHERE name = 'Staff'")

	delegatedManagerCtx := &CurrentUserContext{
		User: models.User{
			ID:       50,
			Username: "user_mgr",
			FullName: "User Manager",
			RoleID:   staffRoleID,
			RoleName: "Manager",
		},
		Permissions: map[string]bool{
			domain.PermSettingsUsers: true,
		},
	}

	escalateForm := url.Values{}
	escalateForm.Set("username", "sneaky_admin")
	escalateForm.Set("full_name", "Sneaky Admin")
	escalateForm.Set("password", "ValidPass123!")
	escalateForm.Set("role_id", fmt.Sprintf("%d", adminRoleID))

	escReq := httptest.NewRequest(http.MethodPost, "/settings/users/add", strings.NewReader(escalateForm.Encode()))
	escReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	escReq = escReq.WithContext(context.WithValue(escReq.Context(), UserContextKey, delegatedManagerCtx))
	escRec := httptest.NewRecorder()
	app.AddUserHandler(escRec, escReq)
	if escRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-Admin assigns Admin role, got %d", escRec.Code)
	}

	// 3. Cannot deactivate or demote the last active Admin
	var adminUserID int64
	_ = db.DB.Get(&adminUserID, "SELECT id FROM users WHERE username = 'admin'")

	demoteForm := url.Values{}
	demoteForm.Set("full_name", "System Administrator")
	demoteForm.Set("role_id", fmt.Sprintf("%d", staffRoleID))
	demoteForm.Set("is_active", "1")

	demoteReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/settings/users/update/%d", adminUserID), strings.NewReader(demoteForm.Encode()))
	demoteReq.SetPathValue("id", fmt.Sprintf("%d", adminUserID))
	demoteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	demoteRec := httptest.NewRecorder()
	app.UpdateUserHandler(demoteRec, demoteReq)
	if demoteRec.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request when demoting last active Admin, got %d", demoteRec.Code)
	}

	// 4. Deactivating or changing password of a user revokes active sessions
	var staffUserID int64
	_ = db.DB.Get(&staffUserID, "SELECT id FROM users WHERE username = 'staff'")
	_, err := db.DB.Exec("INSERT INTO user_sessions (id, user_id, expires_at) VALUES ('staff-sess-1', ?, ?)", staffUserID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to seed staff session: %v", err)
	}

	deactForm := url.Values{}
	deactForm.Set("full_name", "Staff Operator")
	deactForm.Set("role_id", fmt.Sprintf("%d", staffRoleID))
	deactForm.Set("is_active", "0")

	deactReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/settings/users/update/%d", staffUserID), strings.NewReader(deactForm.Encode()))
	deactReq.SetPathValue("id", fmt.Sprintf("%d", staffUserID))
	deactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deactRec := httptest.NewRecorder()
	app.UpdateUserHandler(deactRec, deactReq)
	if deactRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK deactivating staff user, got %d", deactRec.Code)
	}

	var sessCount int
	_ = db.DB.Get(&sessCount, "SELECT COUNT(*) FROM user_sessions WHERE user_id = ?", staffUserID)
	if sessCount != 0 {
		t.Errorf("Expected active sessions for deactivated user to be revoked (0), got %d", sessCount)
	}
}

func TestM1B_RemindersIDORAndOpenRedirectSanitization(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	// Seed an admin-assigned reminder
	var adminUserID, staffUserID int64
	_ = db.DB.Get(&adminUserID, "SELECT id FROM users WHERE username = 'admin'")
	_ = db.DB.Get(&staffUserID, "SELECT id FROM users WHERE username = 'staff'")

	var adminRemID int64
	err := db.DB.QueryRow(`
		INSERT INTO reminders (title, details, due_date, priority, category, status, assigned_to_user_id)
		VALUES ('Admin Secret Task', 'Do not touch', ?, 'High', 'General', 'Pending', ?)
		RETURNING id
	`, time.Now(), adminUserID).Scan(&adminRemID)
	if err != nil {
		t.Fatalf("Failed to insert admin reminder: %v", err)
	}

	// 1. Read-only viewer (only PermDashboardView) cannot create reminders
	viewerCtx := &CurrentUserContext{
		User: models.User{
			ID:       999,
			Username: "viewer",
			RoleName: "Viewer",
		},
		Permissions: map[string]bool{
			domain.PermDashboardView: true,
		},
	}
	addForm := url.Values{}
	addForm.Set("title", "Viewer Reminder")
	addForm.Set("due_date", time.Now().Format("2006-01-02"))
	addReq := httptest.NewRequest(http.MethodPost, "/reminders/add", strings.NewReader(addForm.Encode()))
	addReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	addReq = addReq.WithContext(context.WithValue(addReq.Context(), UserContextKey, viewerCtx))
	addRec := httptest.NewRecorder()
	app.AddReminderHandler(addRec, addReq)
	if addRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for read-only viewer creating reminder, got %d", addRec.Code)
	}

	// 2. Staff operator with PermSalesCreate can create their own reminder, but cannot toggle/delete Admin's reminder (IDOR)
	staffCtx := &CurrentUserContext{
		User: models.User{
			ID:       staffUserID,
			Username: "staff",
			RoleName: "Staff",
		},
		Permissions: map[string]bool{
			domain.PermDashboardView: true,
			domain.PermSalesCreate:   true,
		},
	}

	toggleReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/reminders/toggle/%d", adminRemID), nil)
	toggleReq.SetPathValue("id", fmt.Sprintf("%d", adminRemID))
	toggleReq = toggleReq.WithContext(context.WithValue(toggleReq.Context(), UserContextKey, staffCtx))
	toggleRec := httptest.NewRecorder()
	app.ToggleReminderHandler(toggleRec, toggleReq)
	if toggleRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when staff toggles admin's reminder, got %d", toggleRec.Code)
	}

	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/reminders/delete/%d", adminRemID), nil)
	delReq.SetPathValue("id", fmt.Sprintf("%d", adminRemID))
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), UserContextKey, staffCtx))
	delRec := httptest.NewRecorder()
	app.DeleteReminderHandler(delRec, delReq)
	if delRec.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when staff deletes admin's reminder, got %d", delRec.Code)
	}

	// 3. Open-redirect and header-injection sanitization on returnUrl
	maliciousURLs := []string{
		"https://evil.example.com/phish",
		"//evil.example.com/phish",
		"/calendar\" , \"target\": \"body",
		"javascript:alert(1)",
		"/calendar\r\nX-Injected: true",
	}
	dummyReq := httptest.NewRequest(http.MethodPost, "http://localhost:8080/reminders/add", nil)
	for _, raw := range maliciousURLs {
		got := sanitizeReminderReturnURL(raw, dummyReq)
		if got != "/reminders" {
			t.Errorf("Expected malicious returnUrl %q to sanitize to '/reminders', got %q", raw, got)
		}
	}

	// Valid same-origin URL preserves path and query
	gotValid := sanitizeReminderReturnURL("http://localhost:8080/calendar?month=2026-10", dummyReq)
	if gotValid != "/calendar?month=2026-10" {
		t.Errorf("Expected same-origin URL to strip host and preserve '/calendar?month=2026-10', got %q", gotValid)
	}
}

func TestM1C_PaginationCapAndPreferenceBodyCap(t *testing.T) {
	setupIsolatedTestDB(t)
	app := &App{}

	// 1. GetLimitParam caps at 500
	reqLimit := httptest.NewRequest(http.MethodGet, "/sales?limit=999999", nil)
	if got := GetLimitParam(reqLimit); got != 500 {
		t.Errorf("Expected GetLimitParam to cap at 500, got %d", got)
	}

	// 2. SaveUserPreferenceHandler rejects value > 32KB and body > 64KB
	userCtx := &CurrentUserContext{
		User: models.User{ID: 1, Username: "admin"},
	}

	bigVal := strings.Repeat("A", 40000)
	bodyJSON, _ := json.Marshal(map[string]string{
		"key":   "test_oversized_pref",
		"value": bigVal,
	})
	reqPref := httptest.NewRequest(http.MethodPost, "/api/user-preferences", bytes.NewReader(bodyJSON))
	reqPref = reqPref.WithContext(context.WithValue(reqPref.Context(), UserContextKey, userCtx))
	recPref := httptest.NewRecorder()
	app.SaveUserPreferenceHandler(recPref, reqPref)
	if recPref.Code != http.StatusBadRequest && recPref.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected 400 or 413 for 40KB preference value, got %d", recPref.Code)
	}

	hugeBody := strings.Repeat("X", 70*1024)
	reqHuge := httptest.NewRequest(http.MethodPost, "/api/user-preferences", strings.NewReader(hugeBody))
	reqHuge = reqHuge.WithContext(context.WithValue(reqHuge.Context(), UserContextKey, userCtx))
	recHuge := httptest.NewRecorder()
	app.SaveUserPreferenceHandler(recHuge, reqHuge)
	if recHuge.Code != http.StatusRequestEntityTooLarge && recHuge.Code != http.StatusBadRequest {
		t.Errorf("Expected 413 or 400 for >64KB request body, got %d", recHuge.Code)
	}
}

func TestM1B_ClientIPSpoofingProtection(t *testing.T) {
	// Untrusted direct peer attempting to spoof X-Forwarded-For
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.RemoteAddr = "203.0.113.50:54321"
	r1.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := getClientIP(r1); got != "203.0.113.50" {
		t.Errorf("Expected untrusted peer IP 203.0.113.50 (ignoring spoofed XFF), got %q", got)
	}

	// Trusted loopback reverse proxy forwarding valid IP
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.RemoteAddr = "127.0.0.1:54321"
	r2.Header.Set("X-Forwarded-For", "198.51.100.77, 127.0.0.1")
	if got := getClientIP(r2); got != "198.51.100.77" {
		t.Errorf("Expected trusted loopback proxy XFF 198.51.100.77, got %q", got)
	}

	// Ensure unused import warning doesn't happen if os is used
	_ = os.Getenv("APP_ENV")
}
