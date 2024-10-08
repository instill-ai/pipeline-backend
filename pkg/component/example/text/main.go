package example

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/operator/text/v0"
)

func main() {

	// Parse command-line arguments
	filePaths := flag.String("file_paths", "", "Comma-separated list of file paths")
	chunkSize := flag.Int("chunksize", 800, "Size of each chunk")
	chunkOverlap := flag.Int("chunkoverlap", 300, "Size of overlap between chunks")

	flag.Parse()

	// Check if file paths are provided
	if *filePaths == "" {
		return
	}

	files := strings.Split(*filePaths, ",")

	for _, file := range files {
		b, err := os.ReadFile(file)

		if err != nil {
			return
		}

		rawText := string(b)

		input := text.ChunkTextInput{
			Text: rawText,
			Strategy: text.Strategy{
				Setting: text.Setting{
					ChunkMethod:  "Markdown",
					ChunkSize:    *chunkSize,
					ChunkOverlap: *chunkOverlap,
					ModelName:    "gpt-4",
				},
			},
		}

		output, err := text.ChunkMarkdown(input)

		if err != nil {
			return
		}

		for i, chunk := range output.TextChunks {
			fmt.Printf("\n\nChunk %d:\n %s\n\n\n", i, chunk.Text)
		}

	}
}
