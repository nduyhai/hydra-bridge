package plugins

import "context"

type AuthResult struct {
	Subject string
	Claims  map[string]interface{}
}

type Credentials struct {
	Username string
	Password string
}

type AuthPlugin interface {
	Name() string
	Authenticate(ctx context.Context, cred Credentials) (*AuthResult, error)
}
