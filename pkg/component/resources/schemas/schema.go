package schemas

import (
	_ "embed"
)

//go:embed schema.json
var SchemaJSON []byte
