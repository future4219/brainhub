package authentication

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct {
	cost      int
	dummyHash []byte
}

func NewBcrypt(cost int) (*Bcrypt, error) {
	if cost < 12 {
		return nil, errors.New("bcrypt cost must be at least 12")
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("brainhub-dummy-password"), cost)
	if err != nil {
		return nil, err
	}
	return &Bcrypt{cost: cost, dummyHash: dummyHash}, nil
}

func (b *Bcrypt) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	return string(hash), err
}

func (b *Bcrypt) Matches(secretHash, password string) bool {
	hash := []byte(secretHash)
	if secretHash == "" {
		hash = b.dummyHash
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}
