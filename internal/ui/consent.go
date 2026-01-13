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
	ch := r.URL.Query().Get("consent_challenge")
	if ch == "" {
		http.Error(w, "missing consent_challenge", http.StatusBadRequest)
		return
	}

	// Read user claims from a cookie (set after login)
	userClaims := map[string]interface{}{}
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
				IDToken:     userClaims,
				AccessToken: userClaims,
			},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Clean up cookie after consent is done
		deleteCookie(w, userInfoCookie)

		http.Redirect(w, r, redir.RedirectTo, http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
