package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/nduyhai/hydra-bridge/internal/hydra"
	"github.com/nduyhai/hydra-bridge/internal/plugins"
	"github.com/nduyhai/hydra-bridge/internal/ui"
)

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing env %s", k)
	}
	return v
}

func mustEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid int env %s", key)
		}
		return i
	}
	return def
}

func mustEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("invalid bool env %s", key)
		}
		return b
	}
	return def
}
func main() {
	cfg := ui.Config{
		Addr:        mustEnv("BRIDGE_ADDR"),
		HydraAdmin:  mustEnv("HYDRA_ADMIN_URL"),
		HydraPublic: mustEnv("HYDRA_PUBLIC_URL"),
		LoginAPIURL: mustEnv("LOGIN_API_URL"),

		CookieAuth: mustEnv("COOKIE_AUTH_KEY"),
		CookieEnc:  mustEnv("COOKIE_ENC_KEY"),

		DefaultProv:  "internal",
		TemplatesDir: "web/templates",

		//  SSO / cookie settings
		SessionTTLSeconds: mustEnvInt("SESSION_TTL_SECONDS", 7*24*3600),
		CookieDomain:      mustEnvDefault("COOKIE_DOMAIN", ""),
		CookieSecure:      mustEnvBool("COOKIE_SECURE", false),
		CookieSameSite:    mustEnvDefault("COOKIE_SAMESITE", "lax"),
	}

	hc := hydra.NewAdminClient(cfg.HydraAdmin)

	reg := plugins.NewRegistry()
	reg.Register(plugins.NewInternalLoginPlugin(cfg.LoginAPIURL))

	app := ui.NewServer(cfg, hc, reg)

	log.Printf("bridge listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, app.Routes()); err != nil {
		log.Fatal(err)
	}
}
