package output_port

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrNoVisibleSources   = errors.New("no visible sources")
	ErrReaderNeedsReissue = errors.New("reader client requires manual reissue")
)
