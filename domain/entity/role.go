package entity

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleReader Role = "reader"
)

// GBrainScopes is the only mapping from a brainhub role to GBrain OAuth scopes.
func (r Role) GBrainScopes() []string {
	switch r {
	case RoleOwner, RoleEditor:
		return []string{"read", "write"}
	default:
		return []string{"read"}
	}
}

func (r Role) IsValid() bool {
	return r == RoleOwner || r == RoleEditor || r == RoleReader
}
