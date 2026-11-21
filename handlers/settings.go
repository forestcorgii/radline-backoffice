package handlers

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"rav/db"
	"rav/models"
)

const MaskedAPIKeyPlaceholder = "••••••••••••••••"

var defaultAllowedAIHosts = map[string]bool{
	"api.deepseek.com":                  true,
	"api.openai.com":                    true,
	"openrouter.ai":                     true,
	"api.groq.com":                      true,
	"api.anthropic.com":                 true,
	"generativelanguage.googleapis.com": true,
}

func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func isMaskedOrBlankAPIKey(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	cleaned := strings.ReplaceAll(trimmed, "•", "")
	cleaned = strings.ReplaceAll(cleaned, "*", "")
	return cleaned == ""
}

// ValidateAIProviderURL validates an AI provider base URL against SSRF rules and an explicit host allowlist.
func ValidateAIProviderURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Errorf("AI provider URL cannot be empty")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid AI provider URL: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("AI provider URL must use https scheme")
	}
	if u.User != nil {
		return fmt.Errorf("AI provider URL must not contain userinfo")
	}

	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return fmt.Errorf("AI provider URL must specify a valid hostname")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("AI provider hostname %q is not allowed", host)
	}

	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedIP(ip) {
			return fmt.Errorf("AI provider IP %q is not allowed", host)
		}
		return fmt.Errorf("AI provider IP literal %q is not in the allowed host list", host)
	}

	allowed := make(map[string]bool, len(defaultAllowedAIHosts))
	for k, v := range defaultAllowedAIHosts {
		allowed[k] = v
	}
	if extra := strings.TrimSpace(os.Getenv("DEEPSEEK_ALLOWED_HOSTS")); extra != "" {
		for _, part := range strings.Split(extra, ",") {
			h := strings.ToLower(strings.TrimSpace(part))
			if h != "" {
				allowed[h] = true
			}
		}
	}

	if !allowed[host] {
		return fmt.Errorf("AI provider hostname %q is not in the allowed host list", host)
	}

	// If the hostname resolves in DNS, ensure none of the resolved IPs are loopback/private/link-local.
	if ips, err := net.LookupIP(host); err == nil {
		for _, ip := range ips {
			if isDisallowedIP(ip) {
				return fmt.Errorf("AI provider hostname %q resolves to a disallowed IP address", host)
			}
		}
	}

	return nil
}

// SettingsHandler redirects to /brands now that brands, categories, and UOMs have separate pages
func (app *App) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/brands", http.StatusSeeOther)
}

// SystemSettingsHandler renders system and AI vision configurations
func (app *App) SystemSettingsHandler(w http.ResponseWriter, r *http.Request) {
	dsConfig := getDeepSeekConfigMasked()
	data := struct {
		DeepSeekConfig DeepSeekConfig
	}{
		DeepSeekConfig: dsConfig,
	}
	app.RenderPage(w, r, "system_settings.html", data)
}

// SaveDeepSeekConfigHandler updates DeepSeek Vision API configuration in system_settings
func (app *App) SaveDeepSeekConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setHXTriggerToast(w, "error", "Invalid form data.")
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	apiBase := strings.TrimSpace(r.FormValue("api_base"))
	model := strings.TrimSpace(r.FormValue("model"))

	if apiBase != "" {
		if err := ValidateAIProviderURL(apiBase); err != nil {
			setHXTriggerToast(w, "error", "Invalid API Base URL: "+err.Error())
			http.Error(w, "Invalid API Base URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	if db.DB == nil {
		setHXTriggerToast(w, "error", "Database is not initialized.")
		http.Error(w, "Database is not initialized", http.StatusInternalServerError)
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		log.Printf("SaveDeepSeekConfigHandler begin tx error: %v", err)
		setHXTriggerToast(w, "error", "Failed to save configuration.")
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	upsertQuery := `
		INSERT INTO system_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = EXCLUDED.value
	`

	if !isMaskedOrBlankAPIKey(apiKey) {
		if _, err := tx.Exec(upsertQuery, "deepseek_api_key", apiKey); err != nil {
			log.Printf("SaveDeepSeekConfigHandler save api_key error: %v", err)
			setHXTriggerToast(w, "error", "Failed to save API key.")
			http.Error(w, "Failed to save API key", http.StatusInternalServerError)
			return
		}
	}

	if _, err := tx.Exec(upsertQuery, "deepseek_api_base", apiBase); err != nil {
		log.Printf("SaveDeepSeekConfigHandler save api_base error: %v", err)
		setHXTriggerToast(w, "error", "Failed to save API base URL.")
		http.Error(w, "Failed to save API base URL", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(upsertQuery, "deepseek_model", model); err != nil {
		log.Printf("SaveDeepSeekConfigHandler save model error: %v", err)
		setHXTriggerToast(w, "error", "Failed to save model.")
		http.Error(w, "Failed to save model", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("SaveDeepSeekConfigHandler commit error: %v", err)
		setHXTriggerToast(w, "error", "Failed to commit configuration.")
		http.Error(w, "Failed to commit configuration", http.StatusInternalServerError)
		return
	}

	setHXTriggerToast(w, "success", "DeepSeek Vision configuration saved successfully!")
	w.WriteHeader(http.StatusOK)
}

// NewUomSettingPageHandler redirects to uom-settings list page
func (app *App) NewUomSettingPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/uom-settings", http.StatusSeeOther)
}

// AddUomSettingHandler inserts a new UOM setting into the database
func (app *App) AddUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.FormValue("item_id")
	muom := r.FormValue("muom")
	factorStr := r.FormValue("conversion_factor")

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil || itemID <= 0 {
		setHXTriggerToast(w, "error", "Invalid item selected.")
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	if muom == "" {
		setHXTriggerToast(w, "error", "MUOM is required.")
		http.Error(w, "MUOM is required", http.StatusBadRequest)
		return
	}

	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil || factor <= 0 {
		setHXTriggerToast(w, "error", "Conversion rate must be greater than zero.")
		http.Error(w, "Invalid conversion factor", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		INSERT INTO uom_settings (item_id, muom, conversion_factor)
		VALUES (?, ?, ?)
	`, itemID, muom, factor)
	if err != nil {
		log.Printf("AddUomSettingHandler insert error: %v", err)
		setHXTriggerToast(w, "error", "Failed to add UOM setting. The item might already have this MUOM.")
		http.Error(w, "Failed to add UOM setting", http.StatusInternalServerError)
		return
	}

	setHXTrigger(w, map[string]any{
		"show-toast": map[string]string{
			"type":    "success",
			"message": "UOM Setting added successfully!",
		},
		"uom-setting-added": "",
	})
	w.WriteHeader(http.StatusOK)
}

// EditUomSettingFormHandler renders the inline edit form for a UOM setting row
func (app *App) EditUomSettingFormHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "UOM setting not found", http.StatusNotFound)
		return
	}

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	data := struct {
		Setting models.UomSettingWithItem
		Uoms    []models.Uom
	}{
		Setting: uomSetting,
		Uoms:    uoms,
	}

	app.Render(w, "uom_setting_edit_row.html", data)
}

// UpdateUomSettingHandler updates an existing UOM setting
func (app *App) UpdateUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	muom := r.FormValue("muom")
	factorStr := r.FormValue("conversion_factor")

	if muom == "" {
		setHXTriggerToast(w, "error", "MUOM is required.")
		http.Error(w, "MUOM is required", http.StatusBadRequest)
		return
	}

	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil || factor <= 0 {
		setHXTriggerToast(w, "error", "Conversion rate must be greater than zero.")
		http.Error(w, "Invalid conversion factor", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("UPDATE uom_settings SET muom = ?, conversion_factor = ? WHERE id = ?", muom, factor, id)
	if err != nil {
		log.Printf("UpdateUomSettingHandler update error: %v", err)
		setHXTriggerToast(w, "error", "Failed to update UOM setting.")
		http.Error(w, "Failed to update UOM setting", http.StatusInternalServerError)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "Failed to retrieve UOM setting", http.StatusInternalServerError)
		return
	}

	setHXTriggerToast(w, "success", "UOM setting updated successfully!")
	app.Render(w, "uom_setting_row.html", uomSetting)
}

// UomSettingRowHandler renders a single UOM setting row
func (app *App) UomSettingRowHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	var uomSetting models.UomSettingWithItem
	err = db.DB.Get(&uomSetting, `
		SELECT u.id, u.item_id, u.muom, u.conversion_factor,
		       i.code as item_code, i.description as item_description, i.default_uom as uom
		FROM uom_settings u
		JOIN items i ON u.item_id = i.id
		WHERE u.id = ?
	`, id)
	if err != nil {
		http.Error(w, "UOM setting not found", http.StatusNotFound)
		return
	}

	app.Render(w, "uom_setting_row.html", uomSetting)
}

// DeleteUomSettingHandler deletes a UOM setting
func (app *App) DeleteUomSettingHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid UOM setting ID", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec("DELETE FROM uom_settings WHERE id = ?", id)
	if err != nil {
		log.Printf("DeleteUomSettingHandler delete error: %v", err)
		setHXTriggerToast(w, "error", "Failed to delete UOM setting.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setHXTriggerToast(w, "success", "UOM setting deleted successfully!")
	w.WriteHeader(http.StatusOK)
}
