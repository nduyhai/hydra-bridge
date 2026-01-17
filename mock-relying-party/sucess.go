package main

import (
	"net/http"
)

const successTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login Successful</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.2);
            max-width: 800px;
            width: 100%;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            text-align: center;
        }
        h2 {
            color: #667eea;
            margin-top: 30px;
            margin-bottom: 15px;
            font-size: 20px;
        }
        h3 {
            color: #333;
            margin-top: 20px;
            margin-bottom: 10px;
            font-size: 18px;
        }
        .info-label {
            color: #666;
            font-weight: bold;
            margin-top: 15px;
            margin-bottom: 5px;
        }
        pre {
            background: #f4f4f4;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            border-left: 4px solid #667eea;
            margin: 10px 0;
            font-size: 13px;
            line-height: 1.5;
        }
        button {
            background: #667eea;
            color: white;
            border: none;
            padding: 12px 30px;
            font-size: 16px;
            border-radius: 5px;
            cursor: pointer;
            transition: background 0.3s;
            margin-right: 10px;
            margin-top: 10px;
        }
        button:hover {
            background: #5568d3;
        }
        button:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .success-btn {
            background: #28a745;
        }
        .success-btn:hover {
            background: #218838;
        }
        .button-group {
            margin-top: 20px;
            text-align: center;
        }
        #result, #introspect-result {
            margin-top: 20px;
        }
        .error {
            color: #dc3545;
        }
        .demo-note {
            text-align: center;
            color: #999;
            font-size: 14px;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #eee;
        }
        .loading {
            color: #667eea;
            font-style: italic;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎉 Login Successful!</h1>
        
        <div class="info-label">State:</div>
        <pre>{{.State}}</pre>

        <div class="info-label">Authorization Code:</div>
        <pre>{{.Code}}</pre>

        <div class="button-group">
            <button id="exchangeBtn" onclick="exchangeToken()">Exchange Token</button>
        </div>
        
        <div id="result"></div>
        <div id="introspect-result"></div>

        <p class="demo-note">This page is for MVP/demo purposes only.</p>
    </div>

    <script>
        let accessToken = null;

        async function exchangeToken() {
            const code = '{{.Code}}';
            const resultDiv = document.getElementById('result');
            const btn = document.getElementById('exchangeBtn');
            
            btn.disabled = true;
            resultDiv.innerHTML = '<p class="loading">⏳ Exchanging token...</p>';

            const currentURL = window.location.origin + window.location.pathname;

            try {
                const response = await fetch('/exchange-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: new URLSearchParams({
                        'code': code,
                        'redirect_uri': currentURL
                    })
                });

                const data = await response.json();
                
                if (response.ok) {
                    accessToken = data.access_token;
                    resultDiv.innerHTML = 
                        '<h2>✅ Token Exchange Successful!</h2>' +
                        '<pre>' + JSON.stringify(data, null, 2) + '</pre>' +
                        '<div class="button-group">' +
                        '<button class="success-btn" onclick="introspectToken()">Introspect Token</button>' +
                        '</div>';
                } else {
                    resultDiv.innerHTML = 
                        '<h3 class="error">❌ Error</h3>' +
                        '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
                    btn.disabled = false;
                }
            } catch (error) {
                resultDiv.innerHTML = 
                    '<h3 class="error">❌ Error</h3>' +
                    '<p class="error">' + error.message + '</p>';
                btn.disabled = false;
            }
        }

        async function introspectToken() {
            if (!accessToken) {
                alert('No access token available');
                return;
            }

            const introspectDiv = document.getElementById('introspect-result');
            introspectDiv.innerHTML = '<p class="loading">⏳ Introspecting token...</p>';

            try {
                const response = await fetch('/introspect-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: new URLSearchParams({
                        'token': accessToken
                    })
                });

                const data = await response.json();
                
                if (response.ok) {
                    introspectDiv.innerHTML = 
                        '<h2>🔍 Token Introspection Result</h2>' +
                        '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
                } else {
                    introspectDiv.innerHTML = 
                        '<h3 class="error">❌ Error</h3>' +
                        '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
                }
            } catch (error) {
                introspectDiv.innerHTML = 
                    '<h3 class="error">❌ Error</h3>' +
                    '<p class="error">' + error.message + '</p>';
            }
        }
    </script>
</body>
</html>
`

func handleSuccess(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	w.Header().Set("Content-Type", "text/html")

	data := struct {
		State string
		Code  string
	}{
		State: state,
		Code:  code,
	}

	_ = successTmpl.Execute(w, data)
}
