package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"radline/db"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginSubmitHandler(t *testing.T) {
	// Initialize in-memory DB
	err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	app := &App{}

	// Insert inactive user
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	hashStr := string(hashBytes)

	_, err = db.DB.Exec(`
		INSERT INTO users (username, password_hash, full_name, role_id, is_active)
		VALUES ('inactive_user', ?, 'Inactive User', 1, FALSE)
	`, hashStr)
	if err != nil {
		t.Fatalf("Failed to create inactive user: %v", err)
	}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		expectedHeader map[string]string
		checkCookie    bool
	}{
		{
			name:           "Missing credentials",
			username:       "",
			password:       "",
			expectedStatus: http.StatusBadRequest,
			expectedHeader: map[string]string{"HX-Trigger": `{"show-toast": {"type": "error", "message": "Please enter both username and password"}}`},
			checkCookie:    false,
		},
		{
			name:           "Invalid credentials",
			username:       "admin",
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
			expectedHeader: map[string]string{"HX-Trigger": `{"show-toast": {"type": "error", "message": "Invalid username or password"}}`},
			checkCookie:    false,
		},
		{
			name:           "Inactive account",
			username:       "inactive_user",
			password:       "password",
			expectedStatus: http.StatusForbidden,
			expectedHeader: map[string]string{"HX-Trigger": `{"show-toast": {"type": "error", "message": "Account has been deactivated. Contact an administrator."}}`},
			checkCookie:    false,
		},
		{
			name:           "Valid login",
			username:       "admin", // Seeded by db.InitDB
			password:       "admin123",
			expectedStatus: http.StatusOK,
			expectedHeader: map[string]string{"HX-Redirect": "/"},
			checkCookie:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("username", tt.username)
			form.Add("password", tt.password)
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			app.LoginSubmitHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			for k, v := range tt.expectedHeader {
				if rec.Header().Get(k) != v {
					t.Errorf("Expected header %s: %s, got %s", k, v, rec.Header().Get(k))
				}
			}

			if tt.checkCookie {
				cookies := rec.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == SessionCookieName {
						found = true
						if c.Value == "" {
							t.Errorf("Expected non-empty session cookie")
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected session cookie to be set")
				}
			}
		})
	}
}
