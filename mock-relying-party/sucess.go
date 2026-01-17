package main

import (
	"net/http"
)

const successTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Login Successful</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        pre { background: #f4f4f4; padding: 10px; border-radius: 4px; overflow-x: auto; }
        button { 
            background: #007bff; 
            color: white; 
            padding: 10px 20px; 
            border: none; 
            border-radius: 4px; 
            cursor: pointer;
            font-size: 16px;
            margin-right: 10px;
        }
        button:hover { background: #0056b3; }
        button:disabled { background: #6c757d; cursor: not-allowed; }
        .success-btn { background: #28a745; }
        .success-btn:hover { background: #218838; }
        #result { margin-top: 20px; }
        #introspect-result { margin-top: 20px; }
    </style>
</head>
<body>
    <h2>Login successful</h2>
    <p><b>state</b>: {{.State}}</p>
    <p><b>authorization code</b>:</p>
    <pre>{{.Code}}</pre>

    <button id="exchangeBtn" onclick="exchangeToken()">Exchange Token</button>
    
    <div id="result"></div>
    <div id="introspect-result"></div>

    <p>This page is for MVP/demo only.</p>

    <script>
        let accessToken = null;

        async function exchangeToken() {
            const code = '{{.Code}}';
            const resultDiv = document.getElementById('result');
            const btn = document.getElementById('exchangeBtn');
            
            btn.disabled = true;
            resultDiv.innerHTML = '<p>Exchanging token...</p>';

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
                    resultDiv.innerHTML = '<h3>Token Exchange Successful!</h3><pre>' + 
                        JSON.stringify(data, null, 2) + '</pre>' +
                        '<button class="success-btn" onclick="introspectToken()">Introspect Token</button>';
                } else {
                    resultDiv.innerHTML = '<h3 style="color: red;">Error:</h3><pre>' + 
                        JSON.stringify(data, null, 2) + '</pre>';
                    btn.disabled = false;
                }
            } catch (error) {
                resultDiv.innerHTML = '<h3 style="color: red;">Error:</h3><p>' + 
                    error.message + '</p>';
                btn.disabled = false;
            }
        }

        async function introspectToken() {
            if (!accessToken) {
                alert('No access token available');
                return;
            }

            const introspectDiv = document.getElementById('introspect-result');
            introspectDiv.innerHTML = '<p>Introspecting token...</p>';

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
                    introspectDiv.innerHTML = '<h3>Token Introspection Result:</h3><pre>' + 
                        JSON.stringify(data, null, 2) + '</pre>';
                } else {
                    introspectDiv.innerHTML = '<h3 style="color: red;">Error:</h3><pre>' + 
                        JSON.stringify(data, null, 2) + '</pre>';
                }
            } catch (error) {
                introspectDiv.innerHTML = '<h3 style="color: red;">Error:</h3><p>' + 
                    error.message + '</p>';
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
