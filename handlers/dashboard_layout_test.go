package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"radline/db"
	"radline/models"
)

func TestDefaultDashboardCards(t *testing.T) {
	cards := DefaultDashboardCards()
	if len(cards) != 9 {
		t.Fatalf("Expected 9 default cards, got %d", len(cards))
	}

	expectedIDs := []string{
		"card-workflow",
		"card-kpi-sales",
		"card-kpi-costs",
		"card-kpi-profit",
		"card-kpi-margin",
		"card-trends",
		"card-top-products",
		"card-goals",
		"card-reminders",
	}

	for i, expected := range expectedIDs {
		if cards[i].ID != expected {
			t.Errorf("Card at index %d expected %s, got %s", i, expected, cards[i].ID)
		}
	}
}

func TestGetUserDashboardCards(t *testing.T) {
	_ = db.InitDB(":memory:")
	app := &App{}

	// Case 1: user with no preference returns default layout
	cards := app.GetUserDashboardCards(999)
	if len(cards) != 9 {
		t.Fatalf("Expected 9 cards, got %d", len(cards))
	}

	// Case 2: user with custom preference (use seeded user ID 1 = admin)
	customOrder := []models.DashboardCardConfig{
		{ID: "card-trends", Width: "half", Height: "large"},
		{ID: "card-workflow", Width: "full", Height: "small"},
	}
	bytesVal, _ := json.Marshal(customOrder)
	err := db.SetUserPreference(1, "dashboard_layout", string(bytesVal))
	if err != nil {
		t.Fatalf("Failed to set user preference: %v", err)
	}

	customCards := app.GetUserDashboardCards(1)
	if len(customCards) != 9 {
		t.Fatalf("Expected 9 total cards (with missing cards appended), got %d", len(customCards))
	}
	if customCards[0].ID != "card-trends" || customCards[0].Width != "half" || customCards[0].Height != "large" {
		t.Errorf("Expected first card to be card-trends (half, large), got %+v", customCards[0])
	}
	if customCards[1].ID != "card-workflow" {
		t.Errorf("Expected second card to be card-workflow, got %s", customCards[1].ID)
	}
}

func TestSaveAndResetDashboardLayoutHandler(t *testing.T) {
	_ = db.InitDB(":memory:")
	app := &App{}

	// 1. Unauthorized
	req := httptest.NewRequest(http.MethodPost, "/dashboard/layout", bytes.NewBufferString("[]"))
	rec := httptest.NewRecorder()
	app.SaveDashboardLayoutHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. Authorized POST Save (use seeded user ID 1)
	userCtx := &CurrentUserContext{
		User: models.User{
			ID:       1,
			Username: "admin",
		},
	}
	payload := []models.DashboardCardConfig{
		{ID: "card-kpi-profit", Width: "half", Height: "small"},
		{ID: "card-trends", Width: "full", Height: "large"},
	}
	payloadBytes, _ := json.Marshal(payload)

	req = httptest.NewRequest(http.MethodPost, "/dashboard/layout", bytes.NewBuffer(payloadBytes))
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, userCtx))
	rec = httptest.NewRecorder()

	app.SaveDashboardLayoutHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify in DB
	prefVal, err := db.GetUserPreference(1, "dashboard_layout")
	if err != nil || prefVal == "" {
		t.Fatalf("Expected saved preference in DB, got error: %v", err)
	}

	// 3. Authorized POST Reset
	reqReset := httptest.NewRequest(http.MethodPost, "/dashboard/layout/reset", nil)
	reqReset = reqReset.WithContext(context.WithValue(reqReset.Context(), UserContextKey, userCtx))
	reqReset.Header.Set("HX-Request", "true")
	recReset := httptest.NewRecorder()

	app.ResetDashboardLayoutHandler(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on HTMX reset, got %d", recReset.Code)
	}
	if recReset.Header().Get("HX-Refresh") != "true" {
		t.Errorf("Expected HX-Refresh: true header on reset")
	}

	// Verify preference deleted from DB
	prefValAfter, _ := db.GetUserPreference(1, "dashboard_layout")
	if prefValAfter != "" {
		t.Errorf("Expected preference deleted from DB, got %q", prefValAfter)
	}
}
