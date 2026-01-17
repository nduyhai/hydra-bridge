package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	homeTemplate    *template.Template
	successTemplate *template.Template
)

func init() {
	var err error

	// Parse layout + home
	homeTemplate, err = template.ParseFiles(
		"mock-relying-party/templates/layout.html",
		"mock-relying-party/templates/home.html",
	)
	if err != nil {
		log.Fatalf("Error parsing home template: %v", err)
	}

	// Parse layout + success
	successTemplate, err = template.ParseFiles(
		"mock-relying-party/templates/layout.html",
		"mock-relying-party/templates/success.html",
	)
	if err != nil {
		log.Fatalf("Error parsing success template: %v", err)
	}

	fmt.Println("Templates loaded successfully")
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	loginURL := "http://localhost:4444/oauth2/auth?" + url.Values{
		"client_id":     []string{"demo-client"},
		"response_type": []string{"code"},
		"scope":         []string{"openid profile email"},
		"redirect_uri":  []string{"http://localhost:8091/success"},
		"state":         []string{"random-state-123"},
	}.Encode()

	data := map[string]interface{}{
		"Title":    "Home",
		"LoginURL": loginURL,
	}

	if err := homeTemplate.ExecuteTemplate(w, "home.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleSuccess(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	data := map[string]interface{}{
		"Title": "Success",
		"State": state,
		"Code":  code,
	}

	if err := successTemplate.ExecuteTemplate(w, "success.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "missing_code",
			"error_description": "Authorization code is required",
		})
		return
	}

	if redirectURI == "" {
		redirectURI = "http://localhost:8091/success"
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	tokenURL := "http://host.docker.internal:4444/oauth2/token"

	fmt.Printf("Exchanging code: %s\n", code)
	fmt.Printf("Using redirect_uri: %s\n", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "request_failed",
			"error_description": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("demo-client", "demo-secret")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "connection_failed",
			"error_description": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "read_failed",
			"error_description": err.Error(),
		})
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func handleIntrospectToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.FormValue("token")
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "missing_token",
			"error_description": "Access token is required",
		})
		return
	}

	data := url.Values{}
	data.Set("token", token)

	introspectURL := "http://host.docker.internal:4445/oauth2/introspect"

	fmt.Printf("Introspecting token: %s\n", token)

	req, err := http.NewRequest("POST", introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "request_failed",
			"error_description": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("demo-client", "demo-secret")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "connection_failed",
			"error_description": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "read_failed",
			"error_description": err.Error(),
		})
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body: %s\n", string(body))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func main() {
	// Register specific routes BEFORE the catch-all
	http.HandleFunc("/success", handleSuccess)
	http.HandleFunc("/exchange-token", handleTokenExchange)
	http.HandleFunc("/introspect-token", handleIntrospectToken)

	// Root handler - must be last
	http.HandleFunc("/", handleHome)

	addr := ":8091"
	fmt.Printf("Relying Party running on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
