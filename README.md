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
  participant User as User (Browser)
  participant RP as Relying Party<br/>:8091
  participant Hydra as Hydra<br/>:4444 (public)<br/>:4445 (admin)
  participant Bridge as Bridge<br/>:8081
  participant LoginAPI as Login API<br/>:8090
  participant DB as PostgreSQL<br/>:5432

  Note over Hydra,DB: Hydra uses PostgreSQL for storage

  User->>RP: Visit app and click "Login"
  RP->>Hydra: GET /oauth2/auth<br/>(redirect to authorization endpoint)
  Hydra-->>User: Set ory_hydra_login_csrf and<br/>ory_hydra_session cookies
  Hydra-->>User: Redirect to Bridge /login<br/>with login_challenge

  User->>Bridge: GET /login with login_challenge
  Bridge->>Hydra: GET Admin :4445<br/>login request by login_challenge
  Hydra-->>Bridge: Login request details (client_id)
  Bridge-->>User: Render login page

  User->>Bridge: POST /login<br/>(username, password, csrf)
  Bridge->>LoginAPI: POST :8090/login<br/>verify credentials
  LoginAPI-->>Bridge: Auth success<br/>(user_id, name, email)
  Bridge-->>User: Set cookie __bridge_user<br/>with claims (short TTL)
  Bridge->>Hydra: PUT Admin :4445<br/>accept login (subject, user_id, context claims)
  Hydra-->>Bridge: redirect_to
  Bridge-->>User: Redirect to Hydra

  Hydra-->>User: Set ory_hydra_consent_csrf cookie
  Hydra-->>User: Redirect to Bridge /consent<br/>with consent_challenge

  User->>Bridge: GET /consent with consent_challenge
  Bridge->>Hydra: GET Admin :4445<br/>consent request by consent_challenge
  Hydra-->>Bridge: Consent request<br/>(requested scopes, client_id)
  Bridge-->>User: Read __bridge_user and render<br/>consent page (name, email)

  User->>Bridge: POST /consent approve (csrf)
  Bridge->>Hydra: PUT Admin :4445<br/>accept consent (grant scopes)
  Note over Bridge,Hydra: Inject claims into id_token and<br/>access_token via session fields
  Hydra-->>Bridge: redirect_to
  Bridge-->>User: Delete __bridge_user<br/>and redirect to Hydra

  Hydra-->>User: Redirect to RP :8091/success<br/>with authorization code

  User->>RP: GET :8091/success<br/>with code and state
  RP-->>User: Render success page<br/>with "Exchange Token" button

  User->>RP: Click "Exchange Token"
  RP->>Hydra: POST :4444/oauth2/token<br/>with code (via host.docker.internal)
  Hydra-->>RP: access_token, id_token, refresh_token
  RP-->>User: Display tokens<br/>with "Introspect Token" button

  User->>RP: Click "Introspect Token"
  RP->>Hydra: POST :4445/oauth2/introspect<br/>with access_token (via host.docker.internal)
  Hydra-->>RP: Token introspection result<br/>(active, claims, expiry)
  RP-->>User: Display introspection result

```

## Run it

```bash
docker compose up -d
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
  -d "redirect_uri=http://localhost:8091/success"

```

* Introspect token

```bash
curl -sS -X POST http://localhost:4445/oauth2/introspect \
  -u demo-client:demo-secret \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=REPLACE_WITH_ACCESS_TOKEN" | jq

```

## Development

### Restart each app

```bash
docker compose restart bridge
docker compose restart relying-party
```