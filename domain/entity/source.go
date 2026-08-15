package entity

import "time"

// Source is the source metadata currently exposed as a Brain.
type Source struct {
	ID         string
	Name       string
	LocalPath  *string
	RemoteURL  *string
	Federated  bool
	PageCount  int
	LastSyncAt *time.Time
}
