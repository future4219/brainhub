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

type RegisteredGBrainClient struct {
	ID     string
	Secret string
}

type GBrainAdmin interface {
	RegisterClient(context.Context, RegisterGBrainClientInput) (RegisteredGBrainClient, error)
	RevokeClient(context.Context, string) error
}
