package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"radline/db"
	"radline/models"
)

type ExtractedItem struct {
	ItemID          int
	ItemDescription string
	Qty             float64
	Uom             string
	Price           float64
	Cost            float64
	Total           float64
	RefPL           string
}

type deepSeekMessageContent struct {
	Type     string                   `json:"type"`
	Text     string                   `json:"text,omitempty"`
	ImageURL *deepSeekImageURLContent `json:"image_url,omitempty"`
}

type deepSeekImageURLContent struct {
	URL string `json:"url"`
}

type deepSeekMessage struct {
	Role    string                  `json:"role"`
	Content []deepSeekMessageContent `json:"content"`
}

type deepSeekResponseFormat struct {
	Type string `json:"type"`
}

type deepSeekRequest struct {
	Model          string                 `json:"model"`
	Messages       []deepSeekMessage      `json:"messages"`
	ResponseFormat *deepSeekResponseFormat `json:"response_format,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
}

type deepSeekTextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepSeekTextRequest struct {
	Model          string                 `json:"model"`
	Messages       []deepSeekTextMessage  `json:"messages"`
	ResponseFormat *deepSeekResponseFormat `json:"response_format,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
}

type deepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type deepSeekReceiptJSON struct {
	DocType   string `json:"doc_type"`
	DocNumber string `json:"doc_number"`
	DocDate   string `json:"doc_date"`
	Supplier  string `json:"supplier"`
	Customer  string `json:"customer"`
	Items     []struct {
		Description string  `json:"description"`
		Qty         float64 `json:"qty"`
		Uom         string  `json:"uom"`
		Price       float64 `json:"price"`
		Total       float64 `json:"total"`
	} `json:"items"`
}

type DeepSeekConfig struct {
	APIKey    string
	APIBase   string
	Model     string
	HasAPIKey bool
}

// ReceiptScannerHandler renders the receipt scanner page
func (app *App) ReceiptScannerHandler(w http.ResponseWriter, r *http.Request) {
	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	dsConfig := getDeepSeekConfig()

	data := struct {
		Items          []models.Item
		Uoms           []models.Uom
		DeepSeekConfig DeepSeekConfig
	}{
		Items:          items,
		Uoms:           uoms,
		DeepSeekConfig: dsConfig,
	}

	app.RenderPage(w, r, "receipt_scanner.html", data)
}

// ReceiptScannerParseHandler handles image upload and extracts receipt data via DeepSeek (Vision or Text+OCR)
func (app *App) ReceiptScannerParseHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to parse form."}}`)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("receipt_image")
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "No image uploaded."}}`)
		http.Error(w, "No image uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		w.Header().Set("HX-Trigger", `{"show-toast": {"type": "error", "message": "Failed to read image file."}}`)
		http.Error(w, "Failed to read image file", http.StatusBadRequest)
		return
	}

	cfg := getDeepSeekConfig()
	var rawContent string

	// Determine processing pipeline:
	// If base URL is official api.deepseek.com (text-only V3/R1 models), use OCR + DeepSeek Text LLM parsing
	isOfficialTextOnly := strings.Contains(cfg.APIBase, "api.deepseek.com")

	if isOfficialTextOnly {
		log.Printf("Using official DeepSeek text endpoint (%s). Executing OCR + DeepSeek V3 text extraction...", cfg.APIBase)
		ocrText, ocrErr := callOCRSpaceAPI(fileBytes, header.Filename)
		if ocrErr != nil {
			err = fmt.Errorf("OCR preprocessing failed: %w", ocrErr)
		} else {
			rawContent, err = callDeepSeekTextAPI(ocrText, cfg)
		}
	} else {
		// Call Direct DeepSeek Vision API
		mimeType := header.Header.Get("Content-Type")
		rawContent, err = callDeepSeekVisionAPI(fileBytes, mimeType)

		// Fallback to OCR + DeepSeek Text if vision fails due to text-only endpoint rejection
		if err != nil && (strings.Contains(err.Error(), "text-only") || strings.Contains(err.Error(), "unknown variant `image_url`")) {
			log.Printf("Vision API rejected by endpoint. Falling back to OCR + DeepSeek Text API...")
			ocrText, ocrErr := callOCRSpaceAPI(fileBytes, header.Filename)
			if ocrErr == nil {
				rawContent, err = callDeepSeekTextAPI(ocrText, cfg)
			}
		}
	}

	var docType, docNumber, docDate, supplier, customer string
	var extractedItems []ExtractedItem

	if err != nil {
		log.Printf("DeepSeek processing failed: %v", err)
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"type": "warning", "message": "DeepSeek error: %s. Loaded empty form."}}`, strings.ReplaceAll(err.Error(), `"`, `'`)))
		docType = "SALES"
	} else {
		docType, docNumber, docDate, supplier, customer, extractedItems, err = parseDeepSeekJSON(rawContent)
		if err != nil {
			log.Printf("Failed to parse DeepSeek JSON response: %v", err)
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "warning", "message": "Failed to parse receipt JSON structure. Loaded empty form."}}`)
			docType = "SALES"
		} else {
			w.Header().Set("HX-Trigger", `{"show-toast": {"type": "success", "message": "DeepSeek successfully extracted sales receipt!"}}`)
		}
	}

	// Resolve extracted items against database items
	for i := range extractedItems {
		itemID, resolvedUOM := findMatchingItem(extractedItems[i].ItemDescription)
		if itemID > 0 {
			extractedItems[i].ItemID = itemID
			if resolvedUOM != "" {
				extractedItems[i].Uom = resolvedUOM
			}
			var item struct {
				Code        string `db:"code"`
				Description string `db:"description"`
			}
			err := db.DB.Get(&item, "SELECT code, description FROM items WHERE id = ?", itemID)
			if err == nil {
				extractedItems[i].ItemDescription = fmt.Sprintf("%s — %s", item.Code, item.Description)
			}
		}
	}

	var items []models.Item
	_ = db.DB.Select(&items, "SELECT id, code, description, default_uom FROM items ORDER BY code ASC")

	var uoms []models.Uom
	_ = db.DB.Select(&uoms, "SELECT id, code FROM uoms ORDER BY code ASC")

	dsConfig := getDeepSeekConfig()

	// If HTMX request, render just the extracted data section
	if r.Header.Get("HX-Request") == "true" {
		data := struct {
			Items          []models.Item
			Uoms           []models.Uom
			DocType        string
			DocNumber      string
			DocDate        string
			Supplier       string
			Customer       string
			ExtractedItems []ExtractedItem
			HasResults     bool
			DeepSeekConfig DeepSeekConfig
		}{
			Items:          items,
			Uoms:           uoms,
			DocType:        docType,
			DocNumber:      docNumber,
			DocDate:        docDate,
			Supplier:       supplier,
			Customer:       customer,
			ExtractedItems: extractedItems,
			HasResults:     true,
			DeepSeekConfig: dsConfig,
		}
		app.Render(w, "receipt_scanner_results.html", data)
		return
	}

	// Full page render fallback
	app.RenderPage(w, r, "receipt_scanner.html", map[string]interface{}{
		"Items":          items,
		"Uoms":           uoms,
		"DeepSeekConfig": dsConfig,
	})
}

// getDeepSeekConfig returns configured API key, base URL, and model from DB system_settings or env vars
func getDeepSeekConfig() DeepSeekConfig {
	apiKey := strings.TrimSpace(db.GetSystemSetting("deepseek_api_key"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	}

	apiBase := strings.TrimSpace(db.GetSystemSetting("deepseek_api_base"))
	if apiBase == "" {
		apiBase = strings.TrimSpace(os.Getenv("DEEPSEEK_API_BASE"))
	}
	if apiBase == "" {
		apiBase = "https://api.deepseek.com/v1"
	}

	model := strings.TrimSpace(db.GetSystemSetting("deepseek_model"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("DEEPSEEK_MODEL"))
	}
	if model == "" {
		model = "deepseek-chat"
	}

	return DeepSeekConfig{
		APIKey:    apiKey,
		APIBase:   apiBase,
		Model:     model,
		HasAPIKey: apiKey != "",
	}
}

// callDeepSeekVisionAPI makes an HTTP request to DeepSeek's OpenAI-compatible Chat Completions API
func callDeepSeekVisionAPI(imageBytes []byte, mimeType string) (string, error) {
	cfg := getDeepSeekConfig()
	if !cfg.HasAPIKey || strings.TrimSpace(cfg.APIKey) == "" {
		return "", fmt.Errorf("API Key is missing. Please configure your API key in Settings or Receipt Scanner drawer.")
	}

	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(imageBytes)
	}

	base64Img := base64.StdEncoding.EncodeToString(imageBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Img)

	prompt := `You are an expert receipt reader. Analyze the attached sales receipt or invoice image and extract structured data into JSON with keys:
- "doc_type": string ("SI", "DR", or "SALES")
- "doc_number": string (invoice/receipt number)
- "doc_date": string (YYYY-MM-DD)
- "supplier": string (store or supplier name)
- "customer": string (customer name)
- "items": array of objects with keys: "description" (string), "qty" (number), "uom" (string), "price" (number), "total" (number)

Return ONLY valid JSON matching this schema.`

	reqBody := deepSeekRequest{
		Model: cfg.Model,
		Messages: []deepSeekMessage{
			{
				Role: "user",
				Content: []deepSeekMessageContent{
					{
						Type: "text",
						Text: prompt,
					},
					{
						Type: "image_url",
						ImageURL: &deepSeekImageURLContent{
							URL: dataURL,
						},
					},
				},
			},
		},
		ResponseFormat: &deepSeekResponseFormat{
			Type: "json_object",
		},
		Temperature: 0.1,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimSuffix(cfg.APIBase, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "Radline BackOffice")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to DeepSeek API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respStr := string(bodyBytes)
		if resp.StatusCode == http.StatusUnauthorized || strings.Contains(respStr, "Missing Authentication") || strings.Contains(respStr, "401") || strings.Contains(respStr, "invalid_api_key") {
			return "", fmt.Errorf("Authentication failed (401 Unauthorized). Please verify that your API Key for '%s' is correct in settings.", cfg.APIBase)
		}
		if strings.Contains(respStr, "unknown variant `image_url`") || strings.Contains(respStr, "expected `text`") {
			return "", fmt.Errorf("endpoint '%s' is text-only and rejected image input. Please set Base URL & Model to a Vision-capable provider (e.g. OpenRouter https://openrouter.ai/api/v1 or SiliconFlow)", cfg.APIBase)
		}
		return "", fmt.Errorf("DeepSeek API returned status %d: %s", resp.StatusCode, respStr)
	}

	var dsResp deepSeekResponse
	if err := json.Unmarshal(bodyBytes, &dsResp); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}

	if dsResp.Error != nil && dsResp.Error.Message != "" {
		return "", fmt.Errorf("DeepSeek API error: %s", dsResp.Error.Message)
	}

	if len(dsResp.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API returned no choices")
	}

	return dsResp.Choices[0].Message.Content, nil
}

// parseDeepSeekJSON parses raw JSON string returned by DeepSeek Vision API into receipt fields
func parseDeepSeekJSON(rawJSON string) (docType, docNumber, docDate, supplier, customer string, items []ExtractedItem, err error) {
	cleaned := strings.TrimSpace(rawJSON)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		if len(lines) >= 2 {
			if strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			cleaned = strings.Join(lines, "\n")
		}
	}
	cleaned = strings.TrimSpace(cleaned)

	var res deepSeekReceiptJSON
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return "", "", "", "", "", nil, err
	}

	docType = res.DocType
	if docType == "" {
		docType = "SALES"
	}
	docNumber = res.DocNumber
	docDate = res.DocDate
	supplier = res.Supplier
	customer = res.Customer

	for _, item := range res.Items {
		extracted := ExtractedItem{
			ItemDescription: item.Description,
			Qty:             item.Qty,
			Uom:             item.Uom,
			Price:           item.Price,
			Total:           item.Total,
		}
		if extracted.Total == 0 && extracted.Qty > 0 && extracted.Price > 0 {
			extracted.Total = extracted.Qty * extracted.Price
		}
		items = append(items, extracted)
	}

	if len(items) == 0 {
		items = []ExtractedItem{
			{
				ItemID:          0,
				ItemDescription: "",
				Qty:             1.0,
				Uom:             "",
				Price:           0.0,
				Total:           0.0,
			},
		}
	}

	return docType, docNumber, docDate, supplier, customer, items, nil
}

// findMatchingItem queries the database to find a matching item ID and Default UOM
func findMatchingItem(description string) (int, string) {
	if description == "" {
		return 0, ""
	}
	var match struct {
		ID         int    `db:"id"`
		DefaultUOM string `db:"default_uom"`
	}
	// Try exact match on code
	err := db.DB.Get(&match, "SELECT id, default_uom FROM items WHERE code = ? LIMIT 1", description)
	if err == nil {
		return match.ID, match.DefaultUOM
	}
	// Try exact match on description
	err = db.DB.Get(&match, "SELECT id, default_uom FROM items WHERE description = ? LIMIT 1", description)
	if err == nil {
		return match.ID, match.DefaultUOM
	}
	// Try partial match on description
	err = db.DB.Get(&match, "SELECT id, default_uom FROM items WHERE description LIKE ? LIMIT 1", "%"+description+"%")
	if err == nil {
		return match.ID, match.DefaultUOM
	}
	// Try partial match on code
	err = db.DB.Get(&match, "SELECT id, default_uom FROM items WHERE code LIKE ? LIMIT 1", "%"+description+"%")
	if err == nil {
		return match.ID, match.DefaultUOM
	}
	return 0, ""
}

// callOCRSpaceAPI calls the OCR.space Free API to extract raw text from image
func callOCRSpaceAPI(imageBytes []byte, filename string) (string, error) {
	apiKey := os.Getenv("OCR_SPACE_API_KEY")
	if apiKey == "" {
		apiKey = "helloworld"
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return "", err
	}

	_ = writer.WriteField("apikey", apiKey)
	_ = writer.WriteField("language", "eng")
	_ = writer.WriteField("isOverlayRequired", "false")
	_ = writer.WriteField("ocrEngine", "2")

	if err = writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.ocr.space/parse/image", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OCR API returned status: %s", resp.Status)
	}

	var ocrResult struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		OCRExitCode           int  `json:"OCRExitCode"`
		IsErroredOnProcessing bool `json:"IsErroredOnProcessing"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ocrResult); err != nil {
		return "", err
	}

	if ocrResult.IsErroredOnProcessing || len(ocrResult.ParsedResults) == 0 {
		return "", fmt.Errorf("OCR processing error, exit code: %d", ocrResult.OCRExitCode)
	}

	return ocrResult.ParsedResults[0].ParsedText, nil
}

// callDeepSeekTextAPI sends raw OCR text to DeepSeek text LLM to parse into receipt JSON
func callDeepSeekTextAPI(ocrText string, cfg DeepSeekConfig) (string, error) {
	if !cfg.HasAPIKey || strings.TrimSpace(cfg.APIKey) == "" {
		return "", fmt.Errorf("API Key is missing. Please configure your API key in Settings or Receipt Scanner drawer.")
	}

	prompt := fmt.Sprintf(`You are an expert receipt parser. Analyze the following raw text extracted from a sales receipt or invoice, and extract structured data into a JSON object with keys:
- "doc_type": string ("SI", "DR", or "SALES")
- "doc_number": string (invoice/receipt number)
- "doc_date": string (YYYY-MM-DD)
- "supplier": string (store or supplier name)
- "customer": string (customer name)
- "items": array of objects with keys: "description" (string), "qty" (number), "uom" (string), "price" (number), "total" (number)

Return ONLY valid JSON matching this schema.

RAW RECEIPT TEXT:
---
%s
---`, ocrText)

	reqBody := deepSeekTextRequest{
		Model: cfg.Model,
		Messages: []deepSeekTextMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat: &deepSeekResponseFormat{
			Type: "json_object",
		},
		Temperature: 0.1,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimSuffix(cfg.APIBase, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to DeepSeek API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respStr := string(bodyBytes)
		if resp.StatusCode == http.StatusUnauthorized || strings.Contains(respStr, "401") {
			return "", fmt.Errorf("Authentication failed (401 Unauthorized) for '%s'. Please verify your API Key.", cfg.APIBase)
		}
		return "", fmt.Errorf("DeepSeek API returned status %d: %s", resp.StatusCode, respStr)
	}

	var dsResp deepSeekResponse
	if err := json.Unmarshal(bodyBytes, &dsResp); err != nil {
		return "", fmt.Errorf("failed to decode API response: %w", err)
	}

	if dsResp.Error != nil && dsResp.Error.Message != "" {
		return "", fmt.Errorf("DeepSeek API error: %s", dsResp.Error.Message)
	}

	if len(dsResp.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek API returned no choices")
	}

	return dsResp.Choices[0].Message.Content, nil
}
