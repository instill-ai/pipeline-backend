//go:generate compogen readme ./config ./README.mdx
package text

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskDataCleansing string = "TASK_CLEAN_DATA" // Use this constant for the data cleansing task
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	once      sync.Once
	comp      *component
)

// Operator is the derived operator
type component struct {
	base.Component
}

// Execution is the derived execution
type execution struct {
	base.ComponentExecution
}

// Init initializes the operator
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to load component definition: %v", err))
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{ComponentExecution: x}, nil
}

// CleanDataInput defines the input structure for the data cleansing task
type CleanDataInput struct {
	Texts   []string            `json:"texts"`   // Array of text to be cleaned
	Setting DataCleaningSetting `json:"setting"` // Cleansing configuration
}

// CleanDataOutput defines the output structure for the data cleansing task
type CleanDataOutput struct {
	CleanedTexts []string `json:"texts"` // Array of cleaned text
}

// DataCleaningSetting defines the configuration for data cleansing
type DataCleaningSetting struct {
	CleanMethod     string   `json:"clean-method"` // "Regex" or "Substring"
	ExcludePatterns []string `json:"exclude-patterns,omitempty"`
	IncludePatterns []string `json:"include-patterns,omitempty"`
	ExcludeSubstrs  []string `json:"exclude-substrings,omitempty"`
	IncludeSubstrs  []string `json:"include-substrings,omitempty"`
	CaseSensitive   bool     `json:"case-sensitive,omitempty"`
}

// CleanData cleans the input texts based on the provided settings
func CleanData(input CleanDataInput) CleanDataOutput {
	var cleanedTexts []string

	switch input.Setting.CleanMethod {
	case "Regex":
		cleanedTexts = cleanTextUsingRegex(input.Texts, input.Setting)
	case "Substring":
		cleanedTexts = cleanTextUsingSubstring(input.Texts, input.Setting)
	default:
		// If no valid method is provided, return the original texts
		cleanedTexts = input.Texts
	}

	return CleanDataOutput{CleanedTexts: cleanedTexts}
}

// CleanChunkedData cleans the input texts in chunks based on the provided settings
func CleanChunkedData(input CleanDataInput, chunkSize int) []CleanDataOutput {
	var outputs []CleanDataOutput

	for i := 0; i < len(input.Texts); i += chunkSize {
		end := i + chunkSize
		if end > len(input.Texts) {
			end = len(input.Texts)
		}
		chunk := CleanDataInput{
			Texts:   input.Texts[i:end],
			Setting: input.Setting,
		}
		cleanedChunk := CleanData(chunk)
		outputs = append(outputs, cleanedChunk)
	}
	return outputs
}

// cleanTextUsingRegex cleans the input texts using regular expressions based on the given settings
func cleanTextUsingRegex(inputTexts []string, settings DataCleaningSetting) []string {
	var cleanedTexts []string

	// Precompile exclusion and inclusion patterns
	excludeRegexes := compileRegexPatterns(settings.ExcludePatterns)
	includeRegexes := compileRegexPatterns(settings.IncludePatterns)

	for _, text := range inputTexts {
		include := true

		// Exclude patterns
		for _, re := range excludeRegexes {
			if re.MatchString(text) {
				include = false
				break
			}
		}

		// Include patterns
		if include && len(includeRegexes) > 0 {
			include = false
			for _, re := range includeRegexes {
				if re.MatchString(text) {
					include = true
					break
				}
			}
		}

		if include {
			cleanedTexts = append(cleanedTexts, text)
		}
	}
	return cleanedTexts
}

// cleanTextUsingSubstring cleans the input texts using substrings based on the given settings
func cleanTextUsingSubstring(inputTexts []string, settings DataCleaningSetting) []string {
	var cleanedTexts []string

	for _, text := range inputTexts {
		include := true
		compareText := text
		if !settings.CaseSensitive {
			compareText = strings.ToLower(text)
		}

		// Exclude substrings
		for _, substr := range settings.ExcludeSubstrs {
			if !settings.CaseSensitive {
				substr = strings.ToLower(substr)
			}
			if strings.Contains(compareText, substr) {
				include = false
				break
			}
		}

		// Include substrings
		if include && len(settings.IncludeSubstrs) > 0 {
			include = false
			for _, substr := range settings.IncludeSubstrs {
				if !settings.CaseSensitive {
					substr = strings.ToLower(substr)
				}
				if strings.Contains(compareText, substr) {
					include = true
					break
				}
			}
		}

		if include {
			cleanedTexts = append(cleanedTexts, text)
		}
	}
	return cleanedTexts
}

// compileRegexPatterns compiles a list of regular expression patterns
func compileRegexPatterns(patterns []string) []*regexp.Regexp {
	var regexes []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Handle regex compilation errors appropriately
			continue // Skip this pattern if it fails
		}
		regexes = append(regexes, re)
	}
	return regexes
}

// FetchJSONInput reads JSON data from a file and unmarshals it into CleanDataInput
func FetchJSONInput(filePath string) (CleanDataInput, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return CleanDataInput{}, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return CleanDataInput{}, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var input CleanDataInput
	err = json.Unmarshal(bytes, &input)
	if err != nil {
		return CleanDataInput{}, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	return input, nil
}

// Execute executes the derived execution for the data cleansing task
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	for _, job := range jobs {
		if e.Task == taskDataCleansing {
			// Fetch JSON input from a specified file
			cleanDataInput, err := FetchJSONInput("path/to/your/input.json") // Replace with your actual file path
			if err != nil {
				job.Error.Error(ctx, fmt.Errorf("failed to fetch input data for cleansing: %w", err))
				continue
			}

			// Perform data cleansing
			cleanedDataOutput := CleanData(cleanDataInput)

			// Optionally, clean the data in chunks
			// Define a chunk size; adjust as needed based on your requirements
			chunkSize := 100 // Example chunk size
			chunkedOutputs := CleanChunkedData(cleanDataInput, chunkSize)

			// Write the cleaned output back to the job output
			err = job.Output.WriteData(ctx, cleanedDataOutput)
			if err != nil {
				job.Error.Error(ctx, fmt.Errorf("failed to write cleaned output data: %w", err))
				continue
			}

			// Optionally handle the chunked outputs if needed
			for _, chunk := range chunkedOutputs {
				err = job.Output.WriteData(ctx, chunk)
				if err != nil {
					job.Error.Error(ctx, fmt.Errorf("failed to write chunked cleaned output data: %w", err))
					continue
				}
			}
		} else {
			job.Error.Error(ctx, fmt.Errorf("not supported task: %s", e.Task))
			continue
		}
	}
	return nil
}
