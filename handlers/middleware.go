package handlers

import (
	"context"
	"net/http"
	"strings"
)

// GetCurrentUser returns the authenticated user context or nil
func GetCurrentUser(r *http.Request) *CurrentUserContext {
	val := r.Context().Value(UserContextKey)
	if val == nil {
		return nil
	}
	ctxUser, ok := val.(*CurrentUserContext)
	if !ok {
		return nil
	}
	return ctxUser
}

// HasPermission checks if the user in request context holds a specific permission
func HasPermission(r *http.Request, permKey string) bool {
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		return false
	}
	return ctxUser.Permissions[permKey]
}

// AuthMiddleware inspects session cookie and enforces authentication for non-public routes
func (app *App) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Public bypass paths (including PWA shell assets)
		if path == "/login" || strings.HasPrefix(path, "/static/") || path == "/favicon.ico" ||
			path == "/sw.js" || path == "/manifest.webmanifest" || path == "/offline.html" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			app.handleUnauthenticated(w, r)
			return
		}

		ctxUser, err := GetUserFromSession(cookie.Value)
		if err != nil || ctxUser == nil {
			app.handleUnauthenticated(w, r)
			return
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), UserContextKey, ctxUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) handleUnauthenticated(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RequirePermission wraps a handler with strict RBAC permission check
func (app *App) RequirePermission(permKey string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctxUser := GetCurrentUser(r)
		if ctxUser == nil {
			app.handleUnauthenticated(w, r)
			return
		}

		if !ctxUser.Permissions[permKey] {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Access Denied: You do not have permission to perform this action."}}`)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`<div class="badge badge-danger" style="display:inline-block; padding:0.5rem 0.75rem;">Access Denied (Insufficient Permissions)</div>`))
				return
			}
			http.Error(w, "403 Forbidden: Insufficient Permissions", http.StatusForbidden)
			return
		}

		handler(w, r)
	}
}
