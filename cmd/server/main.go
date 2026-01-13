package main

import (
	"log"
	"net/http"
	"os"

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

func main() {
	cfg := ui.Config{
		Addr:         mustEnv("BRIDGE_ADDR"),
		HydraAdmin:   mustEnv("HYDRA_ADMIN_URL"),
		HydraPublic:  mustEnv("HYDRA_PUBLIC_URL"),
		LoginAPIURL:  mustEnv("LOGIN_API_URL"),
		CookieAuth:   mustEnv("COOKIE_AUTH_KEY"),
		CookieEnc:    mustEnv("COOKIE_ENC_KEY"),
		DefaultProv:  "internal",
		TemplatesDir: "web/templates",
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
