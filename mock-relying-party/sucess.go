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
        pre { background: #f4f4f4; padding: 10px; border-radius: 4px; }
        button { 
            background: #007bff; 
            color: white; 
            padding: 10px 20px; 
            border: none; 
            border-radius: 4px; 
            cursor: pointer;
            font-size: 16px;
        }
        button:hover { background: #0056b3; }
        button:disabled { background: #6c757d; cursor: not-allowed; }
        #result { margin-top: 20px; }
    </style>
</head>
<body>
    <h2>Login successful</h2>
    <p><b>state</b>: {{.State}}</p>
    <p><b>authorization code</b>:</p>
    <pre>{{.Code}}</pre>

    <button id="exchangeBtn" onclick="exchangeToken()">Exchange Token</button>
    
    <div id="result"></div>

    <p>This page is for MVP/demo only.</p>

    <script>
        async function exchangeToken() {
            const code = '{{.Code}}';
            const resultDiv = document.getElementById('result');
            const btn = document.getElementById('exchangeBtn');
            
            btn.disabled = true;
            resultDiv.innerHTML = '<p>Exchanging token...</p>';

            try {
                const response = await fetch('/exchange-token', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded',
                    },
                    body: new URLSearchParams({
                        'code': code
                    })
                });

                const data = await response.json();
                
                if (response.ok) {
                    resultDiv.innerHTML = '<h3>Token Exchange Successful!</h3><pre>' + 
                        JSON.stringify(data, null, 2) + '</pre>';
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
