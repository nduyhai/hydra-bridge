package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Configuration
type Config struct {
	Port         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	UsePKCE      bool // true for public clients, false for confidential
}

var (
	homeTemplate    *template.Template
	successTemplate *template.Template
	config          Config
	codeVerifiers   = make(map[string]string) // For PKCE
)

func init() {
	var err error

	// Determine template base path
	templateBase := "mock-relying-party/templates"

	// Check if templates exist at this path
	if _, err := os.Stat(templateBase); os.IsNotExist(err) {
		// Try alternative path (for local development)
		templateBase = "templates"
	}

	layoutPath := filepath.Join(templateBase, "layout.html")
	homePath := filepath.Join(templateBase, "home.html")
	successPath := filepath.Join(templateBase, "success.html")

	fmt.Printf("Loading templates from: %s\n", templateBase)
	fmt.Printf("Layout: %s\n", layoutPath)
	fmt.Printf("Home: %s\n", homePath)
	fmt.Printf("Success: %s\n", successPath)

	// Parse layout + home
	homeTemplate, err = template.ParseFiles(layoutPath, homePath)
	if err != nil {
		log.Fatalf("Error parsing home template: %v", err)
	}

	// Parse layout + success
	successTemplate, err = template.ParseFiles(layoutPath, successPath)
	if err != nil {
		log.Fatalf("Error parsing success template: %v", err)
	}

	// Load configuration from environment
	config = Config{
		Port:         getEnv("PORT", "8091"),
		ClientID:     getEnv("CLIENT_ID", "demo-client"),
		ClientSecret: getEnv("CLIENT_SECRET", "demo-secret"),
		RedirectURI:  getEnv("REDIRECT_URI", "http://localhost:8091/success"),
		UsePKCE:      getEnv("USE_PKCE", "false") == "true",
	}

	clientType := "confidential (with secret)"
	if config.UsePKCE {
		clientType = "public (PKCE, no secret)"
	}

	fmt.Printf("Templates loaded successfully\n")
	fmt.Printf("Client Type: %s\n", clientType)
	fmt.Printf("Client ID: %s\n", config.ClientID)
	fmt.Printf("Redirect URI: %s\n", config.RedirectURI)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Generate PKCE code verifier
func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Generate PKCE code challenge from verifier
func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DEBUG: handleHome called - Path: %s\n", r.URL.Path)

	if r.URL.Path != "/" {
		fmt.Printf("DEBUG: Path is not /, returning 404\n")
		http.NotFound(w, r)
		return
	}

	state := fmt.Sprintf("state-%d", time.Now().Unix())

	params := url.Values{
		"client_id":     []string{config.ClientID},
		"response_type": []string{"code"},
		"scope":         []string{"openid profile email"},
		"redirect_uri":  []string{config.RedirectURI},
		"state":         []string{state},
	}

	// Add PKCE if using public client
	if config.UsePKCE {
		codeVerifier := generateCodeVerifier()
		codeChallenge := generateCodeChallenge(codeVerifier)
		codeVerifiers[state] = codeVerifier

		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")

		fmt.Printf("DEBUG: PKCE enabled - State: %s, Challenge: %s\n", state, codeChallenge)
	}

	loginURL := "http://localhost:4444/oauth2/auth?" + params.Encode()

	data := map[string]interface{}{
		"Title":    "Home",
		"LoginURL": loginURL,
	}

	fmt.Printf("DEBUG: Rendering home template\n")
	if err := homeTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Error executing home template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleSuccess(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("DEBUG: handleSuccess called - Path: %s\n", r.URL.Path)

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	fmt.Printf("DEBUG: Code: %s, State: %s\n", code, state)

	data := map[string]interface{}{
		"Title": "Success",
		"State": state,
		"Code":  code,
	}

	fmt.Printf("DEBUG: Rendering success template\n")
	if err := successTemplate.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Error executing success template: %v", err)
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
	state := r.FormValue("state")

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
		redirectURI = config.RedirectURI
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	// Handle based on client type
	if config.UsePKCE {
		// Public client - use PKCE
		data.Set("client_id", config.ClientID)

		codeVerifier, exists := codeVerifiers[state]
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "missing_verifier",
				"error_description": "Code verifier not found for state: " + state,
			})
			return
		}

		delete(codeVerifiers, state)
		data.Set("code_verifier", codeVerifier)

		fmt.Printf("Using PKCE - code_verifier: %s\n", codeVerifier)
	}

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

	// Only use client secret for confidential clients
	if !config.UsePKCE && config.ClientSecret != "" {
		req.SetBasicAuth(config.ClientID, config.ClientSecret)
		fmt.Printf("Using client_secret_basic auth\n")
	}

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

	// Use client auth for confidential clients
	if !config.UsePKCE && config.ClientSecret != "" {
		req.SetBasicAuth(config.ClientID, config.ClientSecret)
	}

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
	http.HandleFunc("/success", handleSuccess)
	http.HandleFunc("/exchange-token", handleTokenExchange)
	http.HandleFunc("/introspect-token", handleIntrospectToken)
	http.HandleFunc("/", handleHome)

	addr := ":" + config.Port
	clientType := "Confidential"
	if config.UsePKCE {
		clientType = "Public (PKCE)"
	}
	fmt.Printf("Relying Party (%s) running on http://localhost%s\n", clientType, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
