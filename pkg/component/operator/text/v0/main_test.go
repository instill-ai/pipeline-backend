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

			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up mock data reading
			ir.ReadDataMock.Set(func(ctx context.Context, v interface{}) error {
				*v.(*ChunkTextInput) = tc.input
				return nil
			})

			// Set up mock data writing and error handling
			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output interface{}) error {
				if tc.name == "error case" {
					c.Assert(output, quicktest.IsNil)
				}
				return nil
			})

			if tc.name == "error case" {
				ir.ReadDataMock.Optional()
			}

			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.name == "error case" {
					c.Assert(err, quicktest.ErrorMatches, "not supported task: FAKE_TASK")
				}
			})

			// Execute the task and assert no errors
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
			expected: CleanDataOutput{
				CleanedTexts: []string{"Hello World!", "This is a test."},
			},
			expectedError: false,
		},
		{
			name: "error case - empty input",
			input: CleanDataInput{
				Texts:   []string{},
				 },
			expected:      CleanDataOutput{},
			expectedError: true,
		},
		{
			name: "error case - nil input",
			input: nil,
			expected:      CleanDataOutput{},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		tc := tc // capture range variable
		c.Run(tc.name, func(c *quicktest.C) {
			output := CleanData(tc.input)
			c.Assert(output.CleanedTexts, quicktest.DeepEquals, tc.expected.CleanedTexts)
			if tc.expectedError {
				c.Assert(len(output.CleanedTexts), quicktest.Equals, 0)
			}
		})
	}
}
