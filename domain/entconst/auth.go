package entconst

type UserState string

const (
	UserStateActive    UserState = "active"
	UserStateSuspended UserState = "suspended"
)

type AuthProvider string

const (
	AuthProviderPassword AuthProvider = "password"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderGitHub   AuthProvider = "github"
)
