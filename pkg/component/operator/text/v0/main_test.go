package text

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestInit tests the Init function
func TestInit(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Initialize Component", func(c *quicktest.C) {
		component := Init(base.Component{}) // Pass a base.Component instance
		c.Assert(component, quicktest.IsNotNil)
	})
}

// TestCleanData tests the CleanData function
func TestCleanData(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name  string
		input CleanDataInput
		want  CleanDataOutput
	}{
		{
			name: "Valid Regex Exclusion",
			input: CleanDataInput{
				Texts: []string{"Keep this text", "Remove this text"},
				Setting: DataCleaningSetting{
					CleanMethod:     "Regex",
					ExcludePatterns: []string{"Remove"},
				},
			},
			want: CleanDataOutput{
				CleanedTexts: []string{"Keep this text"},
			},
		},
		{
			name: "Valid Substring Exclusion",
			input: CleanDataInput{
				Texts: []string{"Keep this text", "Remove this text"},
				Setting: DataCleaningSetting{
					CleanMethod:     "Substring",
					ExcludeSubstrs:  []string{"Remove"},
					IncludeSubstrs:  []string{"Keep"},
					CaseSensitive:    false,
				},
			},
			want: CleanDataOutput{
				CleanedTexts: []string{"Keep this text"},
			},
		},
		{
			name: "No Exclusion",
			input: CleanDataInput{
				Texts: []string{"Text without exclusions"},
				Setting: DataCleaningSetting{
					CleanMethod: "Regex",
				},
			},
			want: CleanDataOutput{
				CleanedTexts: []string{"Text without exclusions"},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output := CleanData(tc.input)
			c.Assert(output, quicktest.DeepEquals, tc.want)
		})
	}
}

// TestFetchJSONInput tests the FetchJSONInput function
func TestFetchJSONInput(t *testing.T) {
	c := quicktest.New(t)

	// Create a temporary JSON file for testing
	tempFile, err := os.CreateTemp("", "test_input.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up

	// Write test data to the file
	testData := CleanDataInput{
		Texts: []string{"Sample text 1.", "Sample text 2."},
		Setting: DataCleaningSetting{
			CleanMethod:     "Regex",
			ExcludePatterns: []string{"exclude this"},
		},
	}
	data, _ := json.Marshal(testData)
	if _, err := tempFile.Write(data); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	// Test FetchJSONInput
	c.Run("Fetch JSON Input", func(c *quicktest.C) {
		input, err := FetchJSONInput(tempFile.Name())
		c.Assert(err, quicktest.IsNil)
		c.Assert(input, quicktest.DeepEquals, testData)
	})
}

// TestExecute tests the Execute function of the execution struct
func TestExecute(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Execute Task", func(c *quicktest.C) {
		component := Init(base.Component{}) // Pass a base.Component instance
		execution, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      taskDataCleansing,
		})
		c.Assert(err, quicktest.IsNil)

		// Create a mock job
		mockJob := &base.Job{
			Output: &MockOutput{}, // Implement MockOutput to simulate job output
			Error:  &MockError{},  // Implement MockError to simulate error handling
		}

		err = execution.Execute(context.Background(), []*base.Job{mockJob})
		c.Assert(err, quicktest.IsNil)
	})
}

// TestChunkTextFunctionality tests the chunkText function
func TestChunkTextFunctionality(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name  string
		input ChunkTextInput
		want  ChunkTextOutput
	}{
		{
			name: "Valid Token Chunking",
			input: ChunkTextInput{
				Text: "This is a sample text for chunking.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Token",
						ChunkSize:   10,
						ModelName:  "gpt-3.5-turbo",
					},
				},
			},
			want: ChunkTextOutput{
				ChunkNum:         2,
				TextChunks: []TextChunk{
					{
						Text:          "This is a sample",
						StartPosition: 0,
						EndPosition:   13,
						TokenCount:    5,
					},
					{
						Text:          "text for chunking.",
						StartPosition: 14,
						EndPosition:   29,
						TokenCount:    4,
					},
				},
				TokenCount:       9,
				ChunksTokenCount: 9,
			},
		},
		{
			name: "Valid Recursive Chunking",
			input: ChunkTextInput{
				Text: "This is a sample text for chunking.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Recursive",
						ChunkSize:   10,
						Separators:  []string{" ", "\n"},
					},
				},
			},
			want: ChunkTextOutput{
				ChunkNum:         2,
				TextChunks: []TextChunk{
					{
						Text:          "This is a sample",
						StartPosition: 0,
						EndPosition:   13,
						TokenCount:    4,
					},
					{
						Text:          "text for chunking.",
						StartPosition: 14,
						EndPosition:   29,
						TokenCount:    4,
					},
				},
				TokenCount:       8,
				ChunksTokenCount: 8,
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output, err := chunkText(tc.input)
			c.Assert(err, quicktest.IsNil)
			c.Assert(output, quicktest.DeepEquals, tc.want)
		})
	}
}

// TestChunkMarkdown tests the chunkMarkdown function
func TestChunkMarkdown(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name  string
		input ChunkTextInput
		want  ChunkTextOutput
	}{
		{
			name: "Valid Markdown Chunking",
			input: ChunkTextInput{
				Text: "This is a sample text for chunking.\n\nAnother paragraph.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Markdown",
						ChunkSize:   10,
						ModelName:  "gpt-3.5-turbo",
					},
				},
			},
			want: ChunkTextOutput{
				ChunkNum:         2,
				TextChunks: []TextChunk{
					{
						Text:          "This is a sample text for chunking.",
						StartPosition: 0,
						EndPosition:   29,
						TokenCount:    7,
					},
					{
						Text:          "Another paragraph.",
						StartPosition: 30,
						EndPosition:   47,
						TokenCount:    2,
					},
				},
				TokenCount:       9,
				ChunksTokenCount: 9,
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output, err := chunkMarkdown(tc.input)
			c.Assert(err, quicktest.IsNil)
			c.Assert(output, quicktest.DeepEquals, tc.want)
		})
	}
}

// MockOutput simulates the output for testing
type MockOutput struct {
	data []*structpb.Struct
}

func (m *MockOutput) Write(ctx context.Context, data *structpb.Struct) error {
	m.data = append(m.data, data)
	return nil
}

// MockError simulates an error for testing
type MockError struct {
	err error
}

// HandleError handles an error for testing, implementing the ErrorHandler interface
func (m *MockError) HandleError(ctx context.Context, err error) {
	m.err = err
}

// Error returns the stored error, implementing the ErrorHandler interface
func (m *MockError) Error(ctx context.Context, err error) {
	m.err = err
}



