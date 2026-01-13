package hydra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type AdminClient struct {
	base string
	hc   *http.Client
}

func NewAdminClient(base string) *AdminClient {
	return &AdminClient{
		base: base,
		hc: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type LoginRequest struct {
	Challenge  string `json:"challenge"`
	Client     Client `json:"client"`
	Skip       bool   `json:"skip"`
	Subject    string `json:"subject"`
	RequestURL string `json:"request_url"`
}

type ConsentRequest struct {
	Challenge      string   `json:"challenge"`
	Client         Client   `json:"client"`
	RequestedScope []string `json:"requested_scope"`
	Skip           bool     `json:"skip"`
	Subject        string   `json:"subject"`
}

type Client struct {
	ClientID   string `json:"client_id"`
	ClientName string `json:"client_name"`
}

type AcceptLoginRequestBody struct {
	Subject     string                 `json:"subject"`
	Remember    bool                   `json:"remember"`
	RememberFor int                    `json:"remember_for"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

type AcceptConsentRequestBody struct {
	GrantScope  []string       `json:"grant_scope"`
	Remember    bool           `json:"remember"`
	RememberFor int            `json:"remember_for"`
	Session     ConsentSession `json:"session,omitempty"`
}

type ConsentSession struct {
	IDToken     map[string]interface{} `json:"id_token,omitempty"`
	AccessToken map[string]interface{} `json:"access_token,omitempty"`
}

type RedirectResponse struct {
	RedirectTo string `json:"redirect_to"`
}

func (c *AdminClient) GetLoginRequest(loginChallenge string) (*LoginRequest, error) {
	u := fmt.Sprintf("%s/oauth2/auth/requests/login?login_challenge=%s", c.base, url.QueryEscape(loginChallenge))
	var out LoginRequest
	if err := c.getJSON(u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) AcceptLoginRequest(loginChallenge string, body AcceptLoginRequestBody) (*RedirectResponse, error) {
	u := fmt.Sprintf("%s/oauth2/auth/requests/login/accept?login_challenge=%s", c.base, url.QueryEscape(loginChallenge))
	var out RedirectResponse
	if err := c.putJSON(u, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) GetConsentRequest(consentChallenge string) (*ConsentRequest, error) {
	u := fmt.Sprintf("%s/oauth2/auth/requests/consent?consent_challenge=%s", c.base, url.QueryEscape(consentChallenge))
	var out ConsentRequest
	if err := c.getJSON(u, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) AcceptConsentRequest(consentChallenge string, body AcceptConsentRequestBody) (*RedirectResponse, error) {
	u := fmt.Sprintf("%s/oauth2/auth/requests/consent/accept?consent_challenge=%s", c.base, url.QueryEscape(consentChallenge))
	var out RedirectResponse
	if err := c.putJSON(u, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) getJSON(u string, out any) error {
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Accept", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("hydra admin %s: %s", res.Status, string(b))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *AdminClient) putJSON(u string, in any, out any) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodPut, u, buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("hydra admin %s: %s", res.Status, string(b))
	}
	return json.NewDecoder(res.Body).Decode(out)
}
