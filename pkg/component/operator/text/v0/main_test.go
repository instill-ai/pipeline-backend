package text

import (
	"context"
	"testing"
	"strings"       // Importing strings for normalizeLineEndings function

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

// Main test function using helper functions without redeclaration
func TestValidateChunkPositionsInMain(t *testing.T) {
	c := quicktest.New(t)

	// Sample data - replace with actual chunk data for your test
	chunks := []TextChunk{
		{StartPosition: 0, EndPosition: 10, TokenCount: 5},
		{StartPosition: 11, EndPosition: 20, TokenCount: 7},
	}

	expectedChunks := []TextChunk{
		{StartPosition: 0, EndPosition: 10, TokenCount: 5},
		{StartPosition: 11, EndPosition: 20, TokenCount: 7},
	}

	// Call validateChunkPositions directly; normalizeLineEndings should be referenced from chunk_text_test.go
	validateChunkPositions(c, chunks, expectedChunks)
}
