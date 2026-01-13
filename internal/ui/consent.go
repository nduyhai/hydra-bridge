package ui

import (
	"net/http"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
)

type consentPageData struct {
	ConsentChallenge string
	ClientID         string
	ClientName       string
	RequestedScope   []string
	CSRF             string
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	ch := r.URL.Query().Get("consent_challenge")
	if ch == "" {
		http.Error(w, "missing consent_challenge", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		req, err := s.hyd.GetConsentRequest(ch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		data := consentPageData{
			ConsentChallenge: ch,
			ClientID:         req.Client.ClientID,
			ClientName:       req.Client.ClientName,
			RequestedScope:   req.RequestedScope,
			CSRF:             csrfToken(s.cfg.CookieAuth, ch),
		}
		_ = s.tmpl.ExecuteTemplate(w, "consent.html", data)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", 400)
			return
		}
		if r.Form.Get("csrf") != csrfToken(s.cfg.CookieAuth, ch) {
			http.Error(w, "csrf invalid", 403)
			return
		}

		ctx, cancel := s.ctx(r)
		defer cancel()

		req, err := s.hyd.GetConsentRequest(ch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// MVP: grant all requested scopes (you can restrict later)
		redir, err := s.hyd.AcceptConsentRequest(ch, hydra.AcceptConsentRequestBody{
			GrantScope:  req.RequestedScope,
			Remember:    true,
			RememberFor: 86400,
			Session: hydra.ConsentSession{
				// You can inject claims here if needed
				IDToken:     map[string]interface{}{},
				AccessToken: map[string]interface{}{},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		_ = ctx // keep ctx for future enhancements
		http.Redirect(w, r, redir.RedirectTo, http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
