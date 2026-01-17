package ui

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
	"github.com/nduyhai/hydra-bridge/internal/plugins"
)

type loginPageData struct {
	LoginChallenge string
	ClientID       string
	ClientName     string
	Provider       string
	CSRF           string
	Error          string
}

// Minimal session payload. You can add "sid", "iat", etc.
type bridgeSession struct {
	Sub    string                 `json:"sub"`
	Claims map[string]interface{} `json:"claims,omitempty"`
	Iat    int64                  `json:"iat"`
	Exp    int64                  `json:"exp"`
}

// --- Cookie helpers (HMAC-signed) ---
//
// You should store a SIGNED value in __bridge_session to prevent tampering.
// Below uses HMAC-SHA256 with base64url(payload) + "." + base64url(sig).
// If you already have a signing helper, reuse it.
func (s *Server) signCookieValue(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.CookieAuth))
	mac.Write(payload)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (s *Server) verifyCookieValue(v string) ([]byte, bool) {
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		return nil, false
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, err2 := base64.RawURLEncoding.DecodeString(parts[1])
	if err1 != nil || err2 != nil {
		return nil, false
	}

	mac := hmac.New(sha256.New, []byte(s.cfg.CookieAuth))
	mac.Write(payload)
	expect := mac.Sum(nil)
	if !hmac.Equal(sig, expect) {
		return nil, false
	}
	return payload, true
}

func (s *Server) readSessionFromRequest(r *http.Request) (*bridgeSession, bool) {
	c, err := r.Cookie(bridgeSessionCookie)
	if err != nil || c.Value == "" {
		return nil, false
	}
	payload, ok := s.verifyCookieValue(c.Value)
	if !ok {
		return nil, false
	}
	var sess bridgeSession
	if err := json.Unmarshal(payload, &sess); err != nil {
		return nil, false
	}
	// expiry check
	if sess.Exp > 0 && time.Now().Unix() > sess.Exp {
		return nil, false
	}
	if sess.Sub == "" {
		return nil, false
	}
	return &sess, true
}

// Optional: keep consent cookie fresh / keep short-lived claims cookie updated.
func (s *Server) setUserInfoCookie(w http.ResponseWriter, claims map[string]interface{}) {
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	s.setShortCookie(w, userInfoCookie, claimsB64, 1800) // 30 minutes
}

// ------------------- Updated handleLogin -------------------

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ch := r.URL.Query().Get("login_challenge")
	if ch == "" {
		http.Error(w, "missing login_challenge", http.StatusBadRequest)
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = s.cfg.DefaultProv
	}

	switch r.Method {
	case http.MethodGet:
		// Always fetch the login request first (for client info + redirect_to, skip, etc.)
		req, err := s.hyd.GetLoginRequest(ch)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// ----- SSO: if a bridge session cookie exists, auto-accept login -----
		if sess, ok := s.readSessionFromRequest(r); ok {
			// Keep a short-lived user-info cookie fresh for consent page rendering
			if sess.Claims != nil {
				s.setUserInfoCookie(w, sess.Claims)
			}

			ttl := s.cfg.SessionTTL()

			redir, err := s.hyd.AcceptLoginRequest(ch, hydra.AcceptLoginRequestBody{
				Subject:     sess.Sub,
				Remember:    true,
				RememberFor: int(ttl.Seconds()), // align with the bridge session
				Context:     sess.Claims,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, redir.RedirectTo, http.StatusFound)
			return
		}

		// No SSO session -> show login page
		data := loginPageData{
			LoginChallenge: ch,
			ClientID:       req.Client.ClientID,
			ClientName:     req.Client.ClientName,
			Provider:       provider,
			CSRF:           csrfToken(s.cfg.CookieAuth, ch),
		}
		if err := s.tmplLogin.ExecuteTemplate(w, "layout", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("csrf") != csrfToken(s.cfg.CookieAuth, ch) {
			http.Error(w, "csrf invalid", http.StatusForbidden)
			return
		}

		pluginName := r.Form.Get("provider")
		if pluginName == "" {
			pluginName = s.cfg.DefaultProv
		}
		p, err := s.reg.Get(pluginName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := s.ctx(r)
		defer cancel()

		res, err := p.Authenticate(ctx, plugins.Credentials{
			Username: r.Form.Get("username"),
			Password: r.Form.Get("password"),
		})
		if err != nil {
			req, _ := s.hyd.GetLoginRequest(ch)
			data := loginPageData{
				LoginChallenge: ch,
				ClientID:       req.Client.ClientID,
				ClientName:     req.Client.ClientName,
				Provider:       pluginName,
				CSRF:           csrfToken(s.cfg.CookieAuth, ch),
				Error:          "Invalid credentials",
			}
			w.WriteHeader(http.StatusUnauthorized)
			if err := s.tmplLogin.ExecuteTemplate(w, "layout", data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		// ----- Create / refresh Bridge SSO session cookie (SOURCE OF TRUTH) -----
		ttl := s.cfg.SessionTTL()
		if ttl <= 0 {
			ttl = time.Duration(bridgeSessionTTLDays) * 24 * time.Hour
		}
		now := time.Now().Unix()
		sess := bridgeSession{
			Sub:    res.Subject,
			Claims: res.Claims,
			Iat:    now,
			Exp:    now + int64(ttl.Seconds()),
		}
		payload, _ := json.Marshal(sess)
		signed := s.signCookieValue(payload)

		// name + value + ttl
		s.setSessionCookie(w, bridgeSessionCookie, signed, ttl)

		// Short-lived cookie for consent UI (optional but handy)
		s.setUserInfoCookie(w, res.Claims)

		// Accept login in Hydra
		redir, err := s.hyd.AcceptLoginRequest(ch, hydra.AcceptLoginRequestBody{
			Subject:     res.Subject, // OIDC sub
			Remember:    true,
			RememberFor: int(ttl.Seconds()),
			Context:     res.Claims,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, redir.RedirectTo, http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
