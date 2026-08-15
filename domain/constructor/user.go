package constructor

import (
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/domain/validation"
)

func NewPasswordUser(userID, identityID, email, name, secretHash string, now time.Time) (entity.User, entity.AuthIdentity, error) {
	normalizedEmail, err := validation.NormalizeEmail(email)
	if err != nil {
		return entity.User{}, entity.AuthIdentity{}, err
	}
	normalizedName, err := validation.NormalizeName(name)
	if err != nil {
		return entity.User{}, entity.AuthIdentity{}, err
	}

	user := entity.User{
		ID:        userID,
		Email:     normalizedEmail,
		Name:      normalizedName,
		State:     entconst.UserStateActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	identity := entity.AuthIdentity{
		ID:         identityID,
		UserID:     userID,
		Provider:   entconst.AuthProviderPassword,
		Identifier: normalizedEmail,
		SecretHash: &secretHash,
		CreatedAt:  now,
	}
	return user, identity, nil
}
