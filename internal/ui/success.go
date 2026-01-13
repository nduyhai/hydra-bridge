package ui

import (
	"fmt"
	"net/http"
)

func (s *Server) handleSuccess(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	w.Header().Set("Content-Type", "text/html")

	fmt.Fprintf(w, `
		<h2>Login successful 🎉</h2>
		<p><b>state</b>: %s</p>
		<p><b>authorization code</b>:</p>
		<pre>%s</pre>

		<p>This page is for MVP/demo only.</p>
	`, state, code)
}
