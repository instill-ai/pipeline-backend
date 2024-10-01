package cmd

import (
	"flag"
	"os"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var updateScripts = flag.Bool("update", false, "update testdata/*.txt files with actual command output")

func TestExecute(t *testing.T) {
	testscript.Run(t, testscript.Params{
		TestWork:      true,
		Dir:           "testdata",
		UpdateScripts: *updateScripts,
	})
}

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"compogen": Execute,
	}))
}
