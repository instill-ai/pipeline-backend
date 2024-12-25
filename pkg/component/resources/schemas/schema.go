package schemas

import (
	_ "embed"
)

//go:embed schema.yaml
var SchemaYAML []byte
