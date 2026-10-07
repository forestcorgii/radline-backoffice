package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"radline/db"
	"radline/domain"
	"radline/models"
)

const SessionCookieName = "radline_session"
const SessionDuration = 7 * 24 * time.Hour

type contextKey string

const UserContextKey = contextKey("current_user")

// CurrentUserContext stores the authenticated user and their active permissions
type CurrentUserContext struct {
	User        models.User
	Permissions map[string]bool
}

// GenerateSessionToken creates a cryptographically secure random token
func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetUserFromSession fetches user & role & permissions from a session token
func GetUserFromSession(token string) (*CurrentUserContext, error) {
	if token == "" || db.DB == nil {
		return nil, sql.ErrNoRows
	}

	var session models.UserSession
	err := db.DB.Get(&session, "SELECT id, user_id, expires_at FROM user_sessions WHERE id = ? AND expires_at > ?", token, time.Now())
	if err != nil {
		return nil, err
	}

	var user models.User
	err = db.DB.Get(&user, `
		SELECT u.id, u.username, u.password_hash, u.full_name, u.role_id, r.name as role_name, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = ? AND u.is_active = 1
	`, session.UserID)
	if err != nil {
		return nil, err
	}

	// Fetch role permissions
	var permKeys []string
	_ = db.DB.Select(&permKeys, "SELECT permission_key FROM role_permissions WHERE role_id = ?", user.RoleID)

	permMap := make(map[string]bool)
	isAdmin := strings.EqualFold(user.RoleName, "Admin")
	if isAdmin {
		for _, p := range domain.GetSystemPermissions() {
			permMap[p.Key] = true
		}
	} else {
		for _, key := range permKeys {
			permMap[key] = true
		}
	}

	return &CurrentUserContext{
		User:        user,
		Permissions: permMap,
	}, nil
}

// LoginPageHandler renders the login screen
func (app *App) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		if ctxUser, errUser := GetUserFromSession(cookie.Value); errUser == nil && ctxUser != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	t, ok := app.Templates["login.html"]
	if !ok {
		http.Error(w, "Login template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = t.Execute(w, nil)
}

// LoginSubmitHandler handles credentials verification and session creation
func (app *App) LoginSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Please enter both username and password"}}`)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="badge badge-danger" style="display:block; padding:0.6rem; margin-top:0.5rem; text-align:center;">Username and password required</div>`))
		return
	}

	var user models.User
	err := db.DB.Get(&user, `
		SELECT u.id, u.username, u.password_hash, u.full_name, u.role_id, r.name as role_name, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username = ?
	`, username)

	if err != nil || !domain.CheckPasswordHash(password, user.PasswordHash) {
		app.LogActivity(r, "LOGIN_FAILED", "User", username, "Invalid login attempt for "+username)
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Invalid username or password"}}`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="badge badge-danger" style="display:block; padding:0.6rem; margin-top:0.5rem; text-align:center;">Invalid username or password</div>`))
		return
	}

	if !user.IsActive {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Account has been deactivated. Contact an administrator."}}`)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<div class="badge badge-danger" style="display:block; padding:0.6rem; margin-top:0.5rem; text-align:center;">Account deactivated</div>`))
		return
	}

	// Create session
	token, err := GenerateSessionToken()
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(SessionDuration)
	_, err = db.DB.Exec("INSERT INTO user_sessions (id, user_id, expires_at) VALUES (?, ?, ?)", token, user.ID, expiresAt)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	app.LogActivityWithUser(user.ID, user.Username, r, "LOGIN", "User", fmt.Sprintf("%d", user.ID), "User logged in successfully")

	// Trigger full redirect to dashboard
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// LogoutHandler terminates the user session
func (app *App) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		// Log activity before deleting session
		if ctxUser, errU := GetUserFromSession(cookie.Value); errU == nil && ctxUser != nil {
			app.LogActivityWithUser(ctxUser.User.ID, ctxUser.User.Username, r, "LOGOUT", "User", fmt.Sprintf("%d", ctxUser.User.ID), "User logged out")
		}
		_, _ = db.DB.Exec("DELETE FROM user_sessions WHERE id = ?", cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// UserChipHandler returns the current authenticated user's badge for the sidebar/header
func (app *App) UserChipHandler(w http.ResponseWriter, r *http.Request) {
	ctxUser := GetCurrentUser(r)
	if ctxUser == nil {
		w.Write([]byte(`<a href="/login" class="btn btn-secondary btn-sm" style="font-size:0.75rem;">Login</a>`))
		return
	}

	// Output clean user badge with avatar initial and role pill
	initial := "U"
	if len(ctxUser.User.FullName) > 0 {
		initial = strings.ToUpper(string(ctxUser.User.FullName[0]))
	} else if len(ctxUser.User.Username) > 0 {
		initial = strings.ToUpper(string(ctxUser.User.Username[0]))
	}

	roleColor := "var(--primary-color)"
	if strings.EqualFold(ctxUser.User.RoleName, "Admin") {
		roleColor = "#dc2626"
	} else if strings.EqualFold(ctxUser.User.RoleName, "Manager") {
		roleColor = "#d97706"
	} else {
		roleColor = "#059669"
	}

	html := fmt.Sprintf(`
	<div id="user-chip-container" class="user-chip-container" style="display:flex; align-items:center; justify-content:space-between; padding:0.625rem 0.75rem; background:var(--glass-bg); border-top:1px solid var(--border-color); gap:0.5rem;">
		<div style="display:flex; align-items:center; gap:0.5rem; overflow:hidden;">
			<div style="width:28px; height:28px; border-radius:50%%; background:%s; color:#ffffff; display:flex; align-items:center; justify-content:center; font-weight:700; font-size:0.8125rem; flex-shrink:0;">
				%s
			</div>
			<div style="display:flex; flex-direction:column; min-width:0;">
				<span style="font-size:0.78125rem; font-weight:600; color:var(--text-main); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">%s</span>
				<span style="font-size:0.65625rem; color:%s; font-weight:600; text-transform:uppercase; letter-spacing:0.04em;">%s</span>
			</div>
		</div>
		<button hx-post="/logout" hx-confirm="Are you sure you want to log out?" class="btn-logout" title="Sign Out" style="background:none; border:none; color:var(--text-muted); cursor:pointer; padding:0.25rem; border-radius:4px; display:flex; align-items:center;" onmouseover="this.style.color='#ef4444'" onmouseout="this.style.color='var(--text-muted)'">
			<svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
				<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
				<polyline points="16 17 21 12 16 7"></polyline>
				<line x1="21" y1="12" x2="9" y2="12"></line>
			</svg>
		</button>
	</div>
	`, roleColor, initial, htmlEscape(ctxUser.User.FullName), roleColor, htmlEscape(ctxUser.User.RoleName))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
