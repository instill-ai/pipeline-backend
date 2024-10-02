// compogen is a generation tool for Instill AI component schemas. It is a
// command line application that should guide users through the usage,
// documentation and maintenance of Instill AI components.
package main

import (
	"os"

	"github.com/instill-ai/pipeline-backend/pkg/component/tools/compogen/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
