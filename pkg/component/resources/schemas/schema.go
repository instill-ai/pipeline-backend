package schemas

import (
	_ "embed"
)

//go:embed schema.yaml
var SchemaYAML []byte

//go:embed chat-schema.yaml
var ChatSchemaYAML []byte
