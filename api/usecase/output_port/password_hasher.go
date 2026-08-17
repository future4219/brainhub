package output_port

type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(secretHash, password string) bool
}
