package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
)

type consentPageData struct {
	ConsentChallenge string
	ClientID         string
	ClientName       string
	RequestedScope   []string
	Name             string
	Email            string
	CSRF             string
}

func (s *Server) handleConsent(w http.ResponseWriter, r *http.Request) {
	// Challenge comes from query on GET, from form on POST
	ch := r.URL.Query().Get("consent_challenge")
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if ch == "" {
			ch = r.Form.Get("consent_challenge")
		}
	}

	if ch == "" {
		http.Error(w, "missing consent_challenge", http.StatusBadRequest)
		return
	}

	// Read user claims from cookie (set after login)
	userClaims := map[string]any{}
	if c, err := r.Cookie(userInfoCookie); err == nil {
		if raw, err := base64.RawURLEncoding.DecodeString(c.Value); err == nil {
			_ = json.Unmarshal(raw, &userClaims)
		}
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
			Name:             fmt.Sprint(userClaims["name"]),
			Email:            fmt.Sprint(userClaims["email"]),
			CSRF:             csrfToken(s.cfg.CookieAuth, ch),
		}

		if err := s.tmplConsent.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, "template render error: "+err.Error(), 500)
			return
		}

	case http.MethodPost:
		if r.Form.Get("csrf") != csrfToken(s.cfg.CookieAuth, ch) {
			http.Error(w, "csrf invalid", 403)
			return
		}

		req, err := s.hyd.GetConsentRequest(ch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Inject claims into tokens (id_token + access_token)
		redir, err := s.hyd.AcceptConsentRequest(ch, hydra.AcceptConsentRequestBody{
			GrantScope:  req.RequestedScope,
			Remember:    true,
			RememberFor: 86400,
			Session: hydra.ConsentSession{
				IDToken:     userClaims, // add extra fields here
				AccessToken: userClaims,
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Clean up cookie after consent is done
		s.deleteCookie(w, userInfoCookie)

		http.Redirect(w, r, redir.RedirectTo, http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
