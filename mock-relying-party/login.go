package main

import (
	"log"
	"net/http"
)

const loginTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login Page</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.2);
            text-align: center;
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
        }
        .login-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 15px 40px;
            font-size: 16px;
            border-radius: 5px;
            cursor: pointer;
            transition: background 0.3s;
        }
        .login-btn:hover {
            background: #5568d3;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome</h1>
        <button class="login-btn" onclick="window.location.href='{{.LoginURL}}'">
            Login
        </button>
    </div>
</body>
</html>
`

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Get the URL from query parameter, default to xxx.com
	loginURL := r.URL.Query().Get("url")
	if loginURL == "" {
		loginURL = "http://localhost:4444/oauth2/auth?response_type=code&client_id=demo-client&redirect_uri=http://localhost:8091/success&scope=openid%20profile%20email&state=mvpthelongnightstage"
	}

	data := PageData{
		LoginURL: loginURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := loginTmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
	}
}
