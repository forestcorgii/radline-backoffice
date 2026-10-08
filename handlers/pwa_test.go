package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPWAPublicBypass(t *testing.T) {
	app := &App{}

	// Dummy handler that records 200 OK
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := app.AuthMiddleware(dummyHandler)

	pwaPaths := []string{
		"/sw.js",
		"/manifest.webmanifest",
		"/offline.html",
		"/favicon.ico",
		"/static/icons/icon-192.png",
		"/static/icons/icon-512.png",
		"/static/icons/icon.svg",
	}

	for _, path := range pwaPaths {
		t.Run("Path_"+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for public PWA path %s without session, got %d", path, rec.Code)
			}
			if strings.Contains(rec.Header().Get("Location"), "/login") {
				t.Fatalf("path %s was redirected to login", path)
			}
		})
	}
}

func TestPWAProtectedRoutesStillEnforceAuth(t *testing.T) {
	app := &App{}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := app.AuthMiddleware(dummyHandler)

	protectedPaths := []string{
		"/",
		"/inventory",
		"/sales",
		"/settings",
	}

	for _, path := range protectedPaths {
		t.Run("Protected_"+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			// Should redirect to login
			if rec.Code != http.StatusSeeOther && rec.Code != http.StatusFound {
				t.Fatalf("expected redirect for %s, got %d", path, rec.Code)
			}
		})
	}
}
