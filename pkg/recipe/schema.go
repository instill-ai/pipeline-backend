package recipe

import (
	_ "embed"
)

//go:embed schema.json
var RecipeSchema []byte
