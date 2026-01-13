package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type internalLoginPlugin struct {
	loginAPI string
	hc       *http.Client
}

func NewInternalLoginPlugin(loginAPI string) AuthPlugin {
	return &internalLoginPlugin{
		loginAPI: loginAPI,
		hc: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (p *internalLoginPlugin) Name() string { return "internal" }

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	OK     bool                   `json:"ok"`
	UserID string                 `json:"user_id"`
	Claims map[string]interface{} `json:"claims"`
	Error  string                 `json:"error"`
}

func (p *internalLoginPlugin) Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error) {
	u := p.loginAPI + "/login"
	body := loginReq{Username: cred.Username, Password: cred.Password}
	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, buf)
	req.Header.Set("Content-Type", "application/json")

	res, err := p.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("login api %s: %s", res.Status, string(b))
	}

	var out loginResp
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if !out.OK || out.UserID == "" {
		return nil, fmt.Errorf("invalid credentials")
	}

	// IMPORTANT: subject becomes OIDC sub
	return &AuthResult{
		Subject: out.UserID,
		Claims:  out.Claims,
	}, nil
}
