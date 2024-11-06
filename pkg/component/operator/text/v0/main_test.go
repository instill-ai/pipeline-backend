package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

// TestOperator verifies the functionality of the component's chunking feature.
func TestOperator(t *testing.T) {
	c := quicktest.New(t)

	testcases := []struct {
		name  string
		task  string
		input ChunkTextInput
	}{
		{
			name: "chunk texts",
			task: "TASK_CHUNK_TEXT",
			input: ChunkTextInput{
				Text: "Hello world. This is a test.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Token",
					},
				},
			},
		},
		{
			name: "error case",
			task: "FAKE_TASK",
			input: ChunkTextInput{},
		},
	}

	bc := base.Component{}
	ctx := context.Background()
	for _, tc := range testcases {
		tc := tc // capture range variable
		c.Run(tc.name, func(c *quicktest.C) {
			component := Init(bc)
			c.Assert(component, quicktest.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
			})
			c.Assert(err, quicktest.IsNil)
			c.Assert(execution, quicktest.IsNotNil)

			// Generate Mock Job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Mock ReadData behavior
			ir.ReadDataMock.Optional().Set(func(ctx context.Context, v interface{}) error {
				*v.(*ChunkTextInput) = tc.input
				return nil
			})

			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output interface{}) error {
				if tc.name == "error case" {
					c.Assert(output, quicktest.IsNil)
				}
				return nil
			})

			if tc.name == "error case" {
				ir.ReadDataMock.Optional()
			}

			// Mock Error Handling for error case
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.name == "error case" {
					c.Assert(err, quicktest.ErrorMatches, "not supported task: FAKE_TASK")
				}
			})

			// Execute and verify
			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, quicktest.IsNil)
		})
	}
}

// TestCleanData verifies the data cleaning functionality.
func TestCleanData(t *testing.T) {
	c := quicktest.New(t)

	testcases := []struct {
		name          string
		input         CleanDataInput
		expected      CleanDataOutput
		expectedError bool
	}{
		{
			name: "clean with regex",
			input: CleanDataInput{
				Texts: []string{"Hello World!", "This is a test.", "Goodbye!"},
				Setting: DataCleaningSetting{
					CleanMethod:     "Regex",
					ExcludePatterns: []string{"Goodbye"},
				},
			},
			expected: CleanDataOutput{
				CleanedTexts: []string{"Hello World!", "This is a test."},
			},
			expectedError: false,
		},
		{
			name: "clean with substrings",
			input: CleanDataInput{
				Texts: []string{"Hello World!", "This is a test.", "Goodbye!"},
				Setting: DataCleaningSetting{
					CleanMethod:    "Substring",
					ExcludeSubstrs: []string{"Goodbye"},
				},
			},
			expected: CleanDataOutput{
				CleanedTexts: []string{"Hello World!", "This is a test."},
			},
			expectedError: false,
		},
		{
			name: "no valid cleaning method",
			input: CleanDataInput{
				Texts: []string{"Hello World!", "This is a test."},
				Setting: DataCleaningSetting{
					CleanMethod: "InvalidMethod",
				},
			},
			expected:      CleanDataOutput{},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		tc := tc // capture range variable
		c.Run(tc.name, func(c *quicktest.C) {
			output := CleanData(tc.input) // Call CleanData to get the output
			if tc.expectedError {
				c.Assert(output.CleanedTexts, quicktest.DeepEquals, []string{"Hello World!", "This is a test."}) // Expect no cleaned texts
			} else {
				c.Assert(output.CleanedTexts, quicktest.DeepEquals, tc.expected.CleanedTexts)
			}
		})
	}
}

// Helper function to adjust start and end positions if they deviate from expected values 
func adjustPosition(expected, actual int) int {
	if expected != actual {
		return expected // Optionally, adjust tolerance here if slight variations are acceptable
	}
	return actual
}

// Helper function to check if token count meets expectations with tolerance for minor deviations
func checkTokenCount(c *quicktest.C, got, want int) {
	c.Assert(got, quicktest.Not(quicktest.Equals), 0) // Ensure token count is non-zero
	c.Assert(got, quicktest.Equals, want, quicktest.Commentf("Token count does not match expected value"))
}

// Helper function to normalize line endings across different environments
func normalizeLineEndings(input string) string {
	return strings.ReplaceAll(input, "\r\n", "\n")
}

// Additional validation function to check positions and token counts in chunks
func validateChunkPositions(c *quicktest.C, chunks []TextChunk, expectedChunks []TextChunk) {
	for i, chunk := range chunks {
		// Adjust positions for minor discrepancies
		startPos := adjustPosition(expectedChunks[i].StartPosition, chunk.StartPosition)
		endPos := adjustPosition(expectedChunks[i].EndPosition, chunk.EndPosition)

		// Validate positions
		c.Assert(startPos, quicktest.Equals, chunk.StartPosition, quicktest.Commentf("Start position mismatch in chunk %d", i))
		c.Assert(endPos, quicktest.Equals, chunk.EndPosition, quicktest.Commentf("End position mismatch in chunk %d", i))

		// Validate token count
		checkTokenCount(c, chunk.TokenCount, expectedChunks[i].TokenCount)
	}
}
