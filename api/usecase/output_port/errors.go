package output_port

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrTokenExpired       = errors.New("token expired")
	ErrNoVisibleSources   = errors.New("no visible sources")
	ErrReaderNeedsReissue = errors.New("reader client requires manual reissue")
)
