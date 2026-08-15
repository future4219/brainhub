package ulid

import oklogulid "github.com/oklog/ulid/v2"

type Generator struct{}

func (Generator) New() string { return oklogulid.Make().String() }
