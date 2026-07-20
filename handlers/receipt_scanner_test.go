package handlers

import (
	"os"
	"testing"
)

func TestParseDeepSeekJSON_ValidJSON(t *testing.T) {
	rawJSON := `{
		"doc_type": "SI",
		"doc_number": "3907",
		"doc_date": "2026-07-14",
		"supplier": "RADLINE INDUSTRIAL TOOLS SUPPLIES",
		"customer": "IVAN",
		"items": [
			{
				"description": "WADFOW WPB2915 P. BRUSH",
				"qty": 2.0,
				"uom": "PC",
				"price": 30.00,
				"total": 60.00
			}
		]
	}`

	docType, docNumber, docDate, supplier, customer, items, err := parseDeepSeekJSON(rawJSON)
	if err != nil {
		t.Fatalf("Unexpected error parsing JSON: %v", err)
	}

	if docType != "SI" {
		t.Errorf("Expected DocType SI, got %s", docType)
	}
	if docNumber != "3907" {
		t.Errorf("Expected DocNumber 3907, got %s", docNumber)
	}
	if docDate != "2026-07-14" {
		t.Errorf("Expected DocDate 2026-07-14, got %s", docDate)
	}
	if supplier != "RADLINE INDUSTRIAL TOOLS SUPPLIES" {
		t.Errorf("Expected supplier RADLINE INDUSTRIAL TOOLS SUPPLIES, got %s", supplier)
	}
	if customer != "IVAN" {
		t.Errorf("Expected customer IVAN, got %s", customer)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].ItemDescription != "WADFOW WPB2915 P. BRUSH" {
		t.Errorf("Expected description 'WADFOW WPB2915 P. BRUSH', got %s", items[0].ItemDescription)
	}
	if items[0].Qty != 2.0 {
		t.Errorf("Expected Qty 2.0, got %f", items[0].Qty)
	}
	if items[0].Price != 30.00 {
		t.Errorf("Expected Price 30.00, got %f", items[0].Price)
	}
	if items[0].Total != 60.00 {
		t.Errorf("Expected Total 60.00, got %f", items[0].Total)
	}
}

func TestParseDeepSeekJSON_MarkdownFencedJSON(t *testing.T) {
	markdownJSON := "```json\n" + `{
		"doc_type": "SALES",
		"doc_number": "98765",
		"doc_date": "2026-12-25",
		"supplier": "ACME CORP",
		"customer": "John Doe",
		"items": [
			{
				"description": "HAMMER 16OZ",
				"qty": 1.0,
				"uom": "PC",
				"price": 150.00,
				"total": 0.0
			}
		]
	}` + "\n```"

	docType, docNumber, docDate, supplier, customer, items, err := parseDeepSeekJSON(markdownJSON)
	if err != nil {
		t.Fatalf("Unexpected error parsing fenced JSON: %v", err)
	}

	if docType != "SALES" {
		t.Errorf("Expected DocType SALES, got %s", docType)
	}
	if docNumber != "98765" {
		t.Errorf("Expected DocNumber 98765, got %s", docNumber)
	}
	if docDate != "2026-12-25" {
		t.Errorf("Expected DocDate 2026-12-25, got %s", docDate)
	}
	if supplier != "ACME CORP" {
		t.Errorf("Expected supplier ACME CORP, got %s", supplier)
	}
	if customer != "John Doe" {
		t.Errorf("Expected customer John Doe, got %s", customer)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	// Check total auto-calculation (qty 1 * price 150)
	if items[0].Total != 150.00 {
		t.Errorf("Expected calculated Total 150.00, got %f", items[0].Total)
	}
}

func TestGetDeepSeekConfig(t *testing.T) {
	origKey := os.Getenv("DEEPSEEK_API_KEY")
	origBase := os.Getenv("DEEPSEEK_API_BASE")
	origModel := os.Getenv("DEEPSEEK_MODEL")
	defer func() {
		os.Setenv("DEEPSEEK_API_KEY", origKey)
		os.Setenv("DEEPSEEK_API_BASE", origBase)
		os.Setenv("DEEPSEEK_MODEL", origModel)
	}()

	os.Setenv("DEEPSEEK_API_KEY", "test-key-123")
	os.Unsetenv("DEEPSEEK_API_BASE")
	os.Unsetenv("DEEPSEEK_MODEL")

	cfg := getDeepSeekConfig()
	if cfg.APIKey != "test-key-123" {
		t.Errorf("Expected key 'test-key-123', got '%s'", cfg.APIKey)
	}
	if cfg.APIBase != "https://api.deepseek.com/v1" {
		t.Errorf("Expected default base 'https://api.deepseek.com/v1', got '%s'", cfg.APIBase)
	}
	if cfg.Model != "deepseek-chat" {
		t.Errorf("Expected default model 'deepseek-chat', got '%s'", cfg.Model)
	}
	if !cfg.HasAPIKey {
		t.Errorf("Expected HasAPIKey to be true")
	}
}
