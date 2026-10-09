package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"radline/db"
)

func TestLogoutHandler(t *testing.T) {
	// Initialize test db
	err := db.InitDB("../backoffice.db")
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}
	defer db.DB.Close()

	app := &App{}

	t.Run("Non-HTMX Request", func(t *testing.T) {
		// Create a mock user session for testing
		sessionToken := "test-session-logout-1"
		expiresAt := time.Now().Add(1 * time.Hour)
		// Instead of assuming user 1 exists, we just insert a user for the test
		_, err = db.DB.Exec("INSERT INTO users (id, username, password_hash, full_name, role_id, is_active) VALUES (99991, 'testuser1', 'hash', 'Test User', 1, 1) ON CONFLICT DO NOTHING")
		_, err = db.DB.Exec("INSERT INTO user_sessions (id, user_id, expires_at) VALUES (?, 99991, ?) ON CONFLICT DO NOTHING", sessionToken, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		// Add the session cookie
		req.AddCookie(&http.Cookie{
			Name:  SessionCookieName,
			Value: sessionToken,
		})

		rec := httptest.NewRecorder()

		app.LogoutHandler(rec, req)

		// Check the response code (should be a redirect for non-HTMX requests)
		if rec.Code != http.StatusSeeOther {
			t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rec.Code)
		}

		// Check the redirect location
		loc := rec.Header().Get("Location")
		if loc != "/login" {
			t.Errorf("Expected redirect location to be /login, got %s", loc)
		}

		// Check that the session was deleted from the database
		var count int
		err = db.DB.Get(&count, "SELECT COUNT(*) FROM user_sessions WHERE id = ?", sessionToken)
		if err != nil {
			t.Fatalf("Failed to count user sessions: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected session to be deleted, but it still exists")
		}

		// Check that the cookie was cleared
		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == SessionCookieName {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Errorf("Expected %s cookie to be present in response", SessionCookieName)
		} else {
			if sessionCookie.Value != "" {
				t.Errorf("Expected cookie value to be empty, got %s", sessionCookie.Value)
			}
			if sessionCookie.MaxAge != -1 {
				t.Errorf("Expected cookie MaxAge to be -1, got %d", sessionCookie.MaxAge)
			}
		}

		// Cleanup
		db.DB.Exec("DELETE FROM users WHERE id = 99991")
	})

	t.Run("HTMX Request", func(t *testing.T) {
		// Create a mock user session for testing
		sessionToken := "test-session-logout-2"
		expiresAt := time.Now().Add(1 * time.Hour)
		// Instead of assuming user 1 exists, we just insert a user for the test
		_, err = db.DB.Exec("INSERT INTO users (id, username, password_hash, full_name, role_id, is_active) VALUES (99992, 'testuser2', 'hash', 'Test User', 1, 1) ON CONFLICT DO NOTHING")
		_, err = db.DB.Exec("INSERT INTO user_sessions (id, user_id, expires_at) VALUES (?, 99992, ?) ON CONFLICT DO NOTHING", sessionToken, expiresAt)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		// Add the session cookie
		req.AddCookie(&http.Cookie{
			Name:  SessionCookieName,
			Value: sessionToken,
		})
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		app.LogoutHandler(rec, req)

		// Check the response code (should be OK for HTMX requests)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		// Check the HX-Redirect header
		loc := rec.Header().Get("HX-Redirect")
		if loc != "/login" {
			t.Errorf("Expected HX-Redirect to be /login, got %s", loc)
		}

		// Check that the session was deleted from the database
		var count int
		err = db.DB.Get(&count, "SELECT COUNT(*) FROM user_sessions WHERE id = ?", sessionToken)
		if err != nil {
			t.Fatalf("Failed to count user sessions: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected session to be deleted, but it still exists")
		}

		// Check that the cookie was cleared
		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == SessionCookieName {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Errorf("Expected %s cookie to be present in response", SessionCookieName)
		} else {
			if sessionCookie.Value != "" {
				t.Errorf("Expected cookie value to be empty, got %s", sessionCookie.Value)
			}
			if sessionCookie.MaxAge != -1 {
				t.Errorf("Expected cookie MaxAge to be -1, got %d", sessionCookie.MaxAge)
			}
		}

		// Cleanup
		db.DB.Exec("DELETE FROM users WHERE id = 99992")
	})

	t.Run("No Cookie Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)

		rec := httptest.NewRecorder()

		app.LogoutHandler(rec, req)

		// Check the response code (should be a redirect for non-HTMX requests)
		if rec.Code != http.StatusSeeOther {
			t.Errorf("Expected status %d, got %d", http.StatusSeeOther, rec.Code)
		}

		// Check the redirect location
		loc := rec.Header().Get("Location")
		if loc != "/login" {
			t.Errorf("Expected redirect location to be /login, got %s", loc)
		}
	})
}
