package entconst

type BrainState string

const (
	BrainStateProvisioning BrainState = "provisioning"
	BrainStateReady        BrainState = "ready"
	BrainStateDegraded     BrainState = "degraded"
	BrainStateFailed       BrainState = "failed"
	BrainStateArchived     BrainState = "archived"
)

type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleReader Role = "reader"
)
