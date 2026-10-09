package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"radline/models"
)

func TestUserChipHandler(t *testing.T) {
	app := &App{}

	tests := []struct {
		name          string
		ctxUser       *CurrentUserContext
		expectedStr   []string
		unexpectedStr []string
	}{
		{
			name:    "Unauthenticated",
			ctxUser: nil,
			expectedStr: []string{
				`<a href="/login"`,
			},
		},
		{
			name: "Admin User with Full Name",
			ctxUser: &CurrentUserContext{
				User: models.User{
					Username: "admin_user",
					FullName: "Admin Name",
					RoleName: "Admin",
				},
			},
			expectedStr: []string{
				"Admin Name",
				"@admin_user",
				"#dc2626", // Admin color
				"A", // from Admin Name
			},
		},
		{
			name: "Manager User with only Username",
			ctxUser: &CurrentUserContext{
				User: models.User{
					Username: "mgr_user",
					RoleName: "Manager",
				},
			},
			expectedStr: []string{
				"@mgr_user",
				"#d97706", // Manager color
				"M", // from mgr_user
			},
		},
		{
			name: "Standard User",
			ctxUser: &CurrentUserContext{
				User: models.User{
					Username: "std_user",
					RoleName: "Staff",
				},
			},
			expectedStr: []string{
				"@std_user",
				"#059669", // Staff/Default color
				"S", // from std_user
			},
		},
		{
			name: "HTML Escaping",
			ctxUser: &CurrentUserContext{
				User: models.User{
					Username: "evil_user",
					FullName: "<script>alert('xss')</script>",
					RoleName: "Bad Role <br>",
				},
			},
			expectedStr: []string{
				"&lt;script&gt;alert('xss')&lt;/script&gt;",
				"Bad Role &lt;br&gt;",
			},
			unexpectedStr: []string{
				"<script>",
				"<br>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/user-chip", nil)
			if tt.ctxUser != nil {
				ctx := context.WithValue(req.Context(), UserContextKey, tt.ctxUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()

			app.UserChipHandler(rec, req)

			respBody := rec.Body.String()

			for _, exp := range tt.expectedStr {
				if !strings.Contains(respBody, exp) {
					t.Errorf("Expected response to contain %q", exp)
				}
			}
			for _, unexp := range tt.unexpectedStr {
				if strings.Contains(respBody, unexp) {
					t.Errorf("Expected response not to contain %q", unexp)
				}
			}
		})
	}
}
