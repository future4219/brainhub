package migrations

import (
	"embed"
)

// Files contains the versioned brainhub database migrations.
//
//go:embed *.sql
var Files embed.FS
