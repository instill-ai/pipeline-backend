package example

import (
	"os"
	"strings"
	"testing"
)

// TestMainFile only confirms the client code runs without errors
func TestMainFile(t *testing.T) {

	files := []string{
		"test_data_with_lists.md",
		"test_data_with_table_and_lists.md",
	}

	os.Args = []string{
		"example",
		"-file_paths", strings.Join(files, ","),
		"-chunksize", "800",
		"-chunkoverlap", "200",
	}

	// Run the main function
	main()

}
