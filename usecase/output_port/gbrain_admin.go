package output_port

import "context"

type RegisterGBrainClientInput struct {
	Name                    string
	Scopes                  []string
	Source                  *string
	FederatedRead           []string
	GrantTypes              []string
	RedirectURIs            []string
	TokenEndpointAuthMethod string
}

type GBrainAdmin interface {
	RegisterClient(context.Context, RegisterGBrainClientInput) (string, error)
	RevokeClient(context.Context, string) error
}
