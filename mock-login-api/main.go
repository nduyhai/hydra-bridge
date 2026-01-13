package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResp struct {
	OK     bool              `json:"ok"`
	UserID string            `json:"user_id,omitempty"`
	Claims map[string]string `json:"claims,omitempty"`
	Error  string            `json:"error,omitempty"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req LoginReq
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Demo rule:
		// username: hai
		// password: 123
		if req.Username == "hai" && req.Password == "123" {
			_ = json.NewEncoder(w).Encode(LoginResp{
				OK:     true,
				UserID: "user-12345",
				Claims: map[string]string{
					"email": "hai@tripzy.local",
					"name":  "Nguyen Hai",
				},
			})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(LoginResp{OK: false, Error: "invalid credentials"})
	})

	log.Println("mock-login-api listening on :8090")
	_ = http.ListenAndServe(":8090", mux)
}
