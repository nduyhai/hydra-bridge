# hydra-bridge

[![Go](https://img.shields.io/badge/go-1.25+-blue)](https://go.dev/)
[![License](https://img.shields.io/github/license/nduyhai/hydra-bridge)](LICENSE)

A GitHub Hydra Brid repository for bootstrapping a new OAuth2.

## MVP A architecture

* Hydra: OAuth2/OIDC + tokens
* Bridge: Login UI + Consent UI + Hydra Admin calls
* Plugins: authenticate users (no UI)
    * Internal plugin calls your existing login API
    * Later: oidc plugin does redirect/callback (still no UI except a “Continue with …” button rendered by Bridge)

## Features

* Flow (MVP)
* Client hits Hydra /oauth2/auth
* Hydra redirects to Bridge /login?login_challenge=...
* Bridge renders login page
* User submits username/password to Bridge POST /login
* Bridge calls internal plugin → plugin calls your Login API
* Bridge calls Hydra Admin accept login with subject = your_user_id
* Hydra redirects to Bridge /consent?consent_challenge=...
* Bridge renders consent page
* User approves → Bridge calls Hydra Admin accept consent
* Hydra returns code → client exchanges at /oauth2/token

## Flow diagram

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant hydra as hydra
    participant bridge as bridge
    participant login_api as login-api (legacy)
    participant postgres as postgres
    Note over hydra, postgres: hydra uses postgres for storage
    Client ->> hydra: GET /oauth2/auth
    hydra -->> Client: Set ory_hydra_login_csrf and ory_hydra_session cookies
    hydra -->> Client: Redirect to bridge /login with login_challenge
    Client ->> bridge: GET /login with login_challenge
    bridge ->> hydra: GET Admin login request by login_challenge
    hydra -->> bridge: Login request details client_id
    bridge -->> Client: Render login page
    Client ->> bridge: POST /login username password csrf
    bridge ->> login_api: POST /login verify credentials
    login_api -->> bridge: Auth success user_id name email
    bridge -->> Client: Set cookie __bridge_user with claims short ttl
    bridge ->> hydra: PUT Admin accept login subject user_id context claims
    hydra -->> bridge: redirect_to
    bridge -->> Client: Redirect to hydra
    hydra -->> Client: Set ory_hydra_consent_csrf cookie
    hydra -->> Client: Redirect to bridge /consent with consent_challenge
    Client ->> bridge: GET /consent with consent_challenge
    bridge ->> hydra: GET Admin consent request by consent_challenge
    hydra -->> bridge: Consent request requested scopes client_id
    bridge -->> Client: Read __bridge_user and render consent page with name email
    Client ->> bridge: POST /consent approve csrf
    bridge ->> hydra: PUT Admin accept consent grant scopes
    Note over bridge, hydra: Inject claims into id_token and access_token via session fields
    hydra -->> bridge: redirect_to
    bridge -->> Client: Delete __bridge_user and redirect to hydra
    hydra -->> Client: Redirect to redirect_uri with authorization code
    Client ->> hydra: POST /oauth2/token with code
    hydra -->> Client: access_token id_token refresh_token


```

## Run it

```bash
docker compose up -d

#retart app 
docker compose restart bridge
```

Create a client (example):

```bash
curl -sS -i -X POST http://localhost:4445/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "demo-client",
    "client_secret": "demo-secret",
    "grant_types": ["authorization_code","refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email offline_access",
    "redirect_uris": ["http://localhost:8091/success"],
    "token_endpoint_auth_method": "client_secret_basic"
  }'


```

Verify the client exists

```bash
curl -sS -i http://localhost:4445/clients/demo-client

```

Demo login credentials (from mock API):

```bash
curl -X POST http://localhost:8090/login \  -H "Content-Type: application/json" \
  -d '{"username":"hai","password":"123"}'
```

## Browser Flow

* Access UI http://localhost:8091
* Click Login (The browser will redirect to SSO)
* Then login with UI

```bash
username: hai
password: 123
```

* Click allow consent
* The browser will redirect to a client http://localhost:8091, We will get authorization code. To exchange token:

```bash
curl -X POST http://localhost:4444/oauth2/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u demo-client:demo-secret \
  -d "grant_type=authorization_code" \
  -d "code=REPLACE_WITH_CODE" \
  -d "redirect_uri=http://localhost:5555/callback"

```

* Introspect token

```bash
curl -sS -X POST http://localhost:4445/oauth2/introspect \
  -u demo-client:demo-secret \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=REPLACE_WITH_ACCESS_TOKEN" | jq

```
