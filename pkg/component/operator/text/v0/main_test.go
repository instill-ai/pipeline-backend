package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"
)

// Constants for test cases
const (
	taskDataCleansing = "TASK_CLEAN_DATA"
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
		component := Init()
		c.Assert(component, quicktest.IsNotNil)
	})
}

// TestCreateExecution tests the CreateExecution function
func TestCreateExecution(t *testing.T) {
	c := quicktest.New(t)

	// Test execution creation
	c.Run("Create Execution", func(c *quicktest.C) {
		component := Init()
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

// TestCleanChunkedData tests the CleanChunkedData function
func TestCleanChunkedData(t *testing.T) {
	c := quicktest.New(t)

	// Add test cases for CleanChunkedData
	c.Run("Clean Chunked Data", func(c *quicktest.C) {
		// Define test inputs and expected outputs
		// Example: output := CleanChunkedData(...)
		// c.Assert(output, quicktest.DeepEquals, expectedOutput)
	})
}

// TestRegexFunctionality tests the regex cleaning functions
func TestRegexFunctionality(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Clean Text Using Regex", func(c *quicktest.C) {
		input := []string{"Sample text with exclude this pattern."} // Change to []string
		expectedOutput := []string{"Sample text with  pattern."}    // Expected output after cleaning

		output := cleanTextUsingRegex(input, []string{"exclude this"}) // Ensure the first argument is []string
		c.Assert(output, quicktest.DeepEquals, expectedOutput)         // Match expected output type
	})

	c.Run("Clean Text Using Substring", func(c *quicktest.C) {
		input := []string{"Sample text without any exclusion."} // Change to []string
		expectedOutput := []string{"Sample text without any exclusion."}

		output := cleanTextUsingSubstring(input, "exclude") // Ensure correct parameters are passed
		c.Assert(output, quicktest.DeepEquals, expectedOutput)
	})
}

// TestCompileRegexPatterns tests the compileRegexPatterns function
func TestCompileRegexPatterns(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Compile Patterns", func(c *quicktest.C) {
		patterns := []string{"exclude this"}
		compiled, err := compileRegexPatterns(patterns) // Ensure you're capturing all return values
		c.Assert(err, quicktest.IsNil)                  // Check for error
		c.Assert(len(compiled), quicktest.Equals, 1)    // Expect one compiled pattern
	})
}

// TestFetchJSONInput tests the FetchJSONInput function
func TestFetchJSONInput(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Fetch JSON Input", func(c *quicktest.C) {
		expected := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"key": {Kind: &structpb.Value_StringValue{StringValue: "value"}},
			},
		}

		output, err := FetchJSONInput("some-input-source") // Adjust input as necessary
		c.Assert(err, quicktest.IsNil)                     // Check for error
		c.Assert(output, quicktest.DeepEquals, expected)
	})
}

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	c := quicktest.New(t)

	c.Run("Execute Task", func(c *quicktest.C) {
		component := Init()
		execution, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      taskDataCleansing,
		})
		c.Assert(err, quicktest.IsNil)

		err = execution.Execute(context.Background(), nil) // Adjust as necessary
		c.Assert(err, quicktest.IsNil)
	})
}
