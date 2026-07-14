package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/items", nil)
	if err != nil {
		log.Fatalf("Request error: %v", err)
	}
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "items-tbody")

	log.Println("Sending request...")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Do error: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response Status: %s", resp.Status)
	log.Println("Response Headers:")
	for k, v := range resp.Header {
		log.Printf("  %s: %v", k, v)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Read body error: %v", err)
	}
	log.Printf("Response Body Length: %d bytes", len(body))
	if len(body) > 500 {
		log.Printf("Response Body End Preview: %s", string(body[len(body)-500:]))
	} else {
		log.Printf("Response Body Preview: %s", string(body))
	}
}
