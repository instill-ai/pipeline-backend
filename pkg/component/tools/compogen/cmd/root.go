package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "compogen",
	Short: "compogen is the Instill AI component schema generation tool",
	Long:  "compogen is the Instill AI component schema generation tool",

	// TODO jvallesm: this should be automatically set according to repo
	// releases.
	Version: "0.1.2",

	// We print errors ourselves rather than letting Cobra do it.
	// This lets us print usage information selectively.
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main(). It only needs to happen once
// to the rootCmd.
func Execute() int {
	cmd, err := rootCmd.ExecuteC()
	if err == nil {
		return 0
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	if _, ok := err.(*runError); !ok {
		// It's not an error from RunE, so it warrants usage information.
		fmt.Fprintf(os.Stderr, "Run '%v --help' for usage.\n", cmd.CommandPath())
	}

	return 1
}

type runError struct {
	error
}

type runFunc = func(cmd *cobra.Command, args []string) error

// wrapRun should be used by all registered commands to wrap the run errors so
// they don't get a usage message printed.
func wrapRun(f runFunc) runFunc {
	return func(cmd *cobra.Command, args []string) error {
		if err := f(cmd, args); err != nil {
			return &runError{err}
		}

		return nil
	}
}
