package ui

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
	"github.com/nduyhai/hydra-bridge/internal/plugins"
)

const userInfoCookie = "__bridge_user"

type Config struct {
	Addr         string
	HydraAdmin   string
	HydraPublic  string
	LoginAPIURL  string
	CookieAuth   string
	CookieEnc    string
	DefaultProv  string
	TemplatesDir string
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

func setShortCookie(w http.ResponseWriter, name, value string, maxAgeSec int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAgeSec,
	})
}

func deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:   name,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
