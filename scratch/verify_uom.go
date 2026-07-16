package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	baseURL := "http://localhost:8080"
	client := &http.Client{}

	fmt.Println("=== Starting UOM Verification Tests ===")

	// 1. Verify GET /settings includes the UOM List card
	fmt.Print("1. Testing GET /settings contains UOM List card... ")
	req, _ := http.NewRequest("GET", baseURL+"/settings", nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	body := string(bodyBytes)
	if !strings.Contains(body, "UOM List") {
		fmt.Println("FAIL: 'UOM List' header not found in settings page.")
		os.Exit(1)
	}
	fmt.Println("PASS")

	// 2. Verify GET /uoms returns UOM results
	fmt.Print("2. Testing GET /uoms results... ")
	req, _ = http.NewRequest("GET", baseURL+"/uoms", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "uoms-results")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	body = string(bodyBytes)
	if !strings.Contains(body, "PCS") && !strings.Contains(body, "PC/S") {
		fmt.Printf("FAIL: Pre-populated UOMs like PCS/PC/S not found in results. Got body:\n%s\n", body)
		os.Exit(1)
	}
	fmt.Println("PASS")

	// 3. Add a new UOM
	fmt.Print("3. Testing POST /uoms/add... ")
	formData := url.Values{}
	formData.Set("code", "TSTUOM")
	req, _ = http.NewRequest("POST", baseURL+"/uoms/add", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: Status code %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("PASS")

	// 4. Verify added UOM is returned
	fmt.Print("4. Testing UOM is listed... ")
	req, _ = http.NewRequest("GET", baseURL+"/uoms", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "uoms-results")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	body = string(bodyBytes)
	if !strings.Contains(body, "TSTUOM") {
		fmt.Println("FAIL: Added UOM 'TSTUOM' not found in list.")
		os.Exit(1)
	}
	fmt.Println("PASS")

	// Find the row containing TSTUOM, and extract its ID
	tstIndex := strings.Index(body, "TSTUOM")
	if tstIndex == -1 {
		fmt.Println("FAIL: Could not find TSTUOM in body.")
		os.Exit(1)
	}
	// Look backwards from TSTUOM to find the closest "uom-row-"
	rowPart := body[:tstIndex]
	idStartIndex := strings.LastIndex(rowPart, "uom-row-")
	if idStartIndex == -1 {
		fmt.Println("FAIL: Could not find uom-row- ID for TSTUOM.")
		os.Exit(1)
	}
	var uomID string
	for i := idStartIndex + 8; i < len(rowPart); i++ {
		if rowPart[i] == '"' || rowPart[i] == ' ' || rowPart[i] == '>' {
			break
		}
		uomID += string(rowPart[i])
	}
	fmt.Printf("Resolved UOM ID for TSTUOM: %s\n", uomID)

	// 5. Verify select dropdown returns the new UOM
	fmt.Print("5. Testing /uoms/select includes new UOM... ")
	req, _ = http.NewRequest("GET", baseURL+"/uoms/select?name=default_uom", nil)
	req.Header.Set("HX-Request", "true")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	body = string(bodyBytes)
	if !strings.Contains(body, "value=\"TSTUOM\"") {
		fmt.Println("FAIL: Select dropdown does not contain TSTUOM option.")
		os.Exit(1)
	}
	fmt.Println("PASS")

	// 6. Delete the UOM
	fmt.Printf("6. Testing DELETE /uoms/delete/%s... ", uomID)
	req, _ = http.NewRequest("DELETE", baseURL+"/uoms/delete/"+uomID, nil)
	req.Header.Set("HX-Request", "true")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: Status code %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("PASS")

	// 7. Verify it is deleted
	fmt.Print("7. Testing UOM is deleted... ")
	req, _ = http.NewRequest("GET", baseURL+"/uoms", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "uoms-results")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		os.Exit(1)
	}
	bodyBytes, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	body = string(bodyBytes)
	if strings.Contains(body, "TSTUOM") {
		fmt.Println("FAIL: Deleted UOM 'TSTUOM' still found in list.")
		os.Exit(1)
	}
	fmt.Println("PASS")

	fmt.Println("\nAll UOM Verification Tests Passed Successfully!")
}
