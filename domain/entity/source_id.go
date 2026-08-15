package entity

// SourceID is GBrain's immutable citation key for a source.
type SourceID string

func (s SourceID) String() string { return string(s) }
