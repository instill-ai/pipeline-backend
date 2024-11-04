package text

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Mocking base.Job for testing
type MockJob struct {
	Input  CleanDataInput
	Output MockOutput
	Error  MockError
}

type MockOutput struct {
	Data []CleanDataOutput
}

func (m *MockOutput) WriteData(ctx context.Context, data CleanDataOutput) error {
	m.Data = append(m.Data, data)
	return nil // Simulate successful write
}

type MockError struct {
	Errors []error
}

func (m *MockError) Error(ctx context.Context, err error) {
	m.Errors = append(m.Errors, err)
}

// Test for CleanData function
func TestCleanData(t *testing.T) {
	input := CleanDataInput{
		Texts: []string{"Hello World", "This is a test.", "Another line."},
		Setting: DataCleaningSetting{
			CleanMethod:     "Regex",
			ExcludePatterns: []string{"World"},
			IncludePatterns: []string{},
		},
	}

	expectedOutput := CleanDataOutput{
		CleanedTexts: []string{"This is a test.", "Another line."},
	}

	result := CleanData(input)
	if len(result.CleanedTexts) != len(expectedOutput.CleanedTexts) {
		t.Errorf("Expected %d cleaned texts, got %d", len(expectedOutput.CleanedTexts), len(result.CleanedTexts))
	}

	for i, text := range result.CleanedTexts {
		if text != expectedOutput.CleanedTexts[i] {
			t.Errorf("Expected cleaned text '%s', got '%s'", expectedOutput.CleanedTexts[i], text)
		}
	}
}

// Test for CleanChunkedData function
func TestCleanChunkedData(t *testing.T) {
	input := CleanDataInput{
		Texts: []string{"Hello World", "This is a test.", "Another line."},
		Setting: DataCleaningSetting{
			CleanMethod:     "Substring",
			ExcludeSubstrs:  []string{"World"},
			IncludeSubstrs:  []string{},
		},
	}

	expectedOutput := []CleanDataOutput{
		{CleanedTexts: []string{"This is a test.", "Another line."}},
	}

	result := CleanChunkedData(input, 2)

	if len(result) != len(expectedOutput) {
		t.Errorf("Expected %d chunked outputs, got %d", len(expectedOutput), len(result))
	}

	for i, chunk := range result {
		if len(chunk.CleanedTexts) != len(expectedOutput[i].CleanedTexts) {
			t.Errorf("Expected %d cleaned texts in chunk, got %d", len(expectedOutput[i].CleanedTexts), len(chunk.CleanedTexts))
		}

		for j, text := range chunk.CleanedTexts {
			if text != expectedOutput[i].CleanedTexts[j] {
				t.Errorf("Expected cleaned text '%s', got '%s'", expectedOutput[i].CleanedTexts[j], text)
			}
		}
	}
}

// Test for FetchJSONInput function with valid JSON
func TestFetchJSONInput_ValidJSON(t *testing.T) {
	// Create a temporary JSON file
	jsonData := `{"texts": ["Sample text"], "setting": {"clean-method": "Regex"}}`
	tempFile, err := ioutil.TempFile("", "input.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(jsonData)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	expected := CleanDataInput{
		Texts: []string{"Sample text"},
		Setting: DataCleaningSetting{
			CleanMethod: "Regex",
		},
	}

	result, err := FetchJSONInput(tempFile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != expected {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

// Test for FetchJSONInput function with an invalid JSON file
func TestFetchJSONInput_InvalidJSON(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "invalid.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write invalid JSON data
	if _, err := tempFile.Write([]byte("{invalid json}")); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tempFile.Close()

	_, err = FetchJSONInput(tempFile.Name())
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

// Test for Execute function
func TestExecute(t *testing.T) {
	ctx := context.Background()
	mockJob := &MockJob{}
	exec := execution{}

	// Prepare a valid job with the cleansing task
	mockJob.Input = CleanDataInput{
		Texts: []string{"Hello World", "Goodbye World"},
		Setting: DataCleaningSetting{
			CleanMethod:     "Regex",
			ExcludePatterns: []string{"World"},
		},
	}

	jobs := []*base.Job{mockJob}

	// Call the Execute method
	err := exec.Execute(ctx, jobs)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check if the output has cleaned texts
	if len(mockJob.Output.Data) == 0 {
		t.Errorf("expected cleaned output, got none")
	}
}
