package constructor

import (
	"errors"
	"regexp"

	"brainhub/domain/entity"
)

var (
	ErrInvalidSourceID = errors.New("invalid source ID")
	sourceIDPattern    = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)
)

func NewSourceID(value string) (entity.SourceID, error) {
	if !sourceIDPattern.MatchString(value) || value == "default" {
		return "", ErrInvalidSourceID
	}
	return entity.SourceID(value), nil
}
