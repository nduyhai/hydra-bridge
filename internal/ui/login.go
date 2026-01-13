package ui

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

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
		req, err := s.hyd.GetLoginRequest(ch)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		data := loginPageData{
			LoginChallenge: ch,
			ClientID:       req.Client.ClientID,
			ClientName:     req.Client.ClientName,
			Provider:       provider,
			CSRF:           csrfToken(s.cfg.CookieAuth, ch),
		}
		_ = s.tmpl.ExecuteTemplate(w, "login.html", data)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", 400)
			return
		}
		if r.Form.Get("csrf") != csrfToken(s.cfg.CookieAuth, ch) {
			http.Error(w, "csrf invalid", 403)
			return
		}

		pluginName := r.Form.Get("provider")
		if pluginName == "" {
			pluginName = s.cfg.DefaultProv
		}
		p, err := s.reg.Get(pluginName)
		if err != nil {
			http.Error(w, err.Error(), 400)
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
			_ = s.tmpl.ExecuteTemplate(w, "login.html", data)
			return
		}

		// Save user claims for consent UI + token claims injection (short-lived)
		claimsJSON, _ := json.Marshal(res.Claims)
		claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
		setShortCookie(w, userInfoCookie, claimsB64, 300) // 5 minutes

		redir, err := s.hyd.AcceptLoginRequest(ch, hydra.AcceptLoginRequestBody{
			Subject:     res.Subject, // becomes OIDC sub
			Remember:    true,
			RememberFor: 3600,
			Context:     res.Claims,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		http.Redirect(w, r, redir.RedirectTo, http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
