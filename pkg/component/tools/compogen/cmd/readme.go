package cmd

import (
	"github.com/spf13/cobra"

	"github.com/instill-ai/pipeline-backend/pkg/component/tools/compogen/pkg/gen"
)

func init() {
	var extraContentPaths map[string]string

	genReadmeCmd := &cobra.Command{
		Use:  "readme [config dir] [target file]",
		Args: cobra.ExactArgs(2),

		Short: "Generate component README",
		Long: `Generates a README.mdx file that describes the purpose and usage of the component.

The first argument specifies the path to the component config directory, i.e., the directory holding the component definitions.
The second argument allows users to specify the path to the generated README file.`,

		RunE: wrapRun(func(cmd *cobra.Command, args []string) error {
			return gen.NewREADMEGenerator(
				args[0],
				args[1],
				extraContentPaths,
			).Generate()
		}),
	}

	genReadmeCmd.Flags().StringToStringVar(
		&extraContentPaths,
		"extraContents",
		nil,
		`Paths to extra contents to be injected into the document.
It takes the form k=v, where k determines the section in or after which the content will be injected, and v is the path to the content.
The possible values of k are: intro, release, config, setup, [task ID], bottom.`,
	)

	rootCmd.AddCommand(genReadmeCmd)
}
