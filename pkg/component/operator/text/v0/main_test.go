package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/component/base" // Import the base package
	"google.golang.org/protobuf/types/known/structpb"
)

// Constants for test cases
const (
	taskDataCleansing = "TASK_CLEAN_DATA" // Remove from here if it's declared in main.go
)

// Test structure
type TestCase struct {
	name  string
	input *CleanDataInput
	want  *CleanDataOutput
}

// TestInit tests the Init function
func TestInit(t *testing.T) {
	c := quicktest.New(t)

	// Test initialization logic
	c.Run("Initialize Component", func(c *quicktest.C) {
		component := Init(base.Component{}) // Pass a base.Component here
		c.Assert(component, quicktest.IsNotNil)
	})
}

// TestCreateExecution tests the CreateExecution function
func TestCreateExecution(t *testing.T) {
	c := quicktest.New(t)

	// Test execution creation
	c.Run("Create Execution", func(c *quicktest.C) {
		component := Init(base.Component{}) // Pass a base.Component here
		execution, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      taskDataCleansing,
		})
		c.Assert(err, quicktest.IsNil)
		c.Assert(execution, quicktest.IsNotNil)
	})
}

// TestCleanData tests the CleanData function
func TestCleanData(t *testing.T) {
	c := quicktest.New(t)

	testCases := []TestCase{
		{
			name: "Valid Input",
			input: &CleanDataInput{
				Texts: []string{"Sample text 1.", "Sample text 2."},
				Setting: DataCleaningSetting{
					CleanMethod:    "Regex",
					ExcludePatterns: []string{"exclude this"},
				},
			},
			want: &CleanDataOutput{
				CleanedTexts: []string{"Sample text 1.", "Sample text 2."}, // Expected cleaned output
			},
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output := CleanData(*tc.input) // Use dereference for CleanData
			c.Assert(output, quicktest.DeepEquals, tc.want)
		})
	}
}

// TestRegexFunctionality tests the regex cleaning functions
func TestRegexFunctionality(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Clean Text Using Regex", func(c *quicktest.C) {
		input := []string{"Sample text with exclude this pattern."}
		expectedOutput := []string{"Sample text with  pattern."}

		output := clean
