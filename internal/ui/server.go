package ui

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
	"github.com/nduyhai/hydra-bridge/internal/plugins"
)

const (
	userInfoCookie       = "__bridge_user"    // short-lived claims for consent UI
	bridgeSessionCookie  = "__bridge_session" // long-lived SSO session cookie
	bridgeSessionTTLDays = 7                  // example only
)

type Config struct {
	Addr        string
	HydraAdmin  string
	HydraPublic string
	LoginAPIURL string

	// Secrets
	CookieAuth string // HMAC signing secret (tamper-proof cookies, CSRF token)
	CookieEnc  string // (optional) encryption secret if you encrypt userInfo cookie

	DefaultProv  string
	TemplatesDir string

	// Bridge session (SSO) cookie settings
	SessionTTLSeconds int    // e.g. 604800 (7 days)
	CookieDomain      string // "" = host-only, or ".tripzy.com"
	CookieSecure      bool   // true in prod (https)
	CookieSameSite    string // "lax" (default), "strict", "none"
}

func (c Config) SameSiteMode() http.SameSite {
	switch strings.ToLower(strings.TrimSpace(c.CookieSameSite)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		// NOTE: SameSite=None requires Secure=true in modern browsers.
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (c Config) SessionTTL() time.Duration {
	if c.SessionTTLSeconds <= 0 {
		return 7 * 24 * time.Hour
	}
	return time.Duration(c.SessionTTLSeconds) * time.Second
}

type Server struct {
	cfg         Config
	hyd         *hydra.AdminClient
	reg         *plugins.Registry
	tmplLogin   *template.Template
	tmplConsent *template.Template
}

func NewServer(cfg Config, hyd *hydra.AdminClient, reg *plugins.Registry) *Server {
	tmplLogin := template.Must(template.ParseFiles(
		"/app/web/templates/layout.html",
		"/app/web/templates/login.html",
	))

	tmplConsent := template.Must(template.ParseFiles(
		"/app/web/templates/layout.html",
		"/app/web/templates/consent.html",
	))
	return &Server{cfg: cfg, hyd: hyd, reg: reg, tmplConsent: tmplConsent, tmplLogin: tmplLogin}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/consent", s.handleConsent)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	return mux
}

func (s *Server) ctx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 15*time.Second)
}

func csrfToken(secret, challenge string) string {
	h := sha256.Sum256([]byte(secret + ":" + challenge))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func (s *Server) setShortCookie(
	w http.ResponseWriter,
	name, value string,
	maxAgeSec int,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   s.cfg.CookieDomain, // "" = host-only (good for localhost)
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,   // true in prod
		SameSite: s.cfg.SameSiteMode(), // Lax by default
		MaxAge:   maxAgeSec,
		Expires:  time.Now().Add(time.Duration(maxAgeSec) * time.Second),
	})
}

// setSessionCookie: for __bridge_session (SSO source of truth)
func (s *Server) setSessionCookie(
	w http.ResponseWriter,
	name, value string,
	ttl time.Duration,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: s.cfg.SameSiteMode(),
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func (s *Server) deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: s.cfg.SameSiteMode(),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
