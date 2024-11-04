//go:generate compogen readme ./config ./README.mdx
package text

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskChunkText     string = "TASK_CHUNK_TEXT"
	taskDataCleansing string = "TASK_CLEAN_DATA" // Ensure this matches your requirement
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
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
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

// FetchDefinition fetches and parses the definition JSON
func FetchDefinition() (map[string]interface{}, error) {
	var definition map[string]interface{}
	err := json.Unmarshal(definitionJSON, &definition)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal definition JSON: %w", err)
	}
	return definition, nil
}

// FetchTasks fetches and parses the tasks JSON
func FetchTasks() (map[string]interface{}, error) {
	var tasks map[string]interface{}
	err := json.Unmarshal(tasksJSON, &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks JSON: %w", err)
	}
	return tasks, nil
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

	if len(cleanedTexts) == 0 {
		return CleanDataOutput{CleanedTexts: nil} // Return nil if there are no cleaned texts
	}

	return CleanDataOutput{CleanedTexts: cleanedTexts}
}

// cleanTextUsingRegex cleans the input texts using regular expressions based on the given settings
func cleanTextUsingRegex(inputTexts []string, settings DataCleaningSetting) []string {
	var cleanedTexts []string

	for _, text := range inputTexts {
		include := true

		// Exclude patterns
		for _, pattern := range settings.ExcludePatterns {
			re := regexp.MustCompile(pattern)
			if re.MatchString(text) {
				include = false
				break
			}
		}

		// Include patterns
		if include && len(settings.IncludePatterns) > 0 {
			include = false
			for _, pattern := range settings.IncludePatterns {
				re := regexp.MustCompile(pattern)
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

// Execute executes the derived execution
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	for _, job := range jobs {
		switch e.Task {
		case taskChunkText:
			inputStruct := ChunkTextInput{}

			err := job.Input.ReadData(ctx, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			var outputStruct ChunkTextOutput
			if inputStruct.Strategy.Setting.ChunkMethod == "Markdown" {
				outputStruct, err = chunkMarkdown(inputStruct)
			} else {
				outputStruct, err = chunkText(inputStruct)
			}

			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			err = job.Output.WriteData(ctx, outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case taskDataCleansing: // Use the correct task constant
			cleanDataInput := CleanDataInput{}
			// Read the data from job input into cleanDataInput
			err := job.Input.ReadData(ctx, &cleanDataInput)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			// Perform data cleansing
			cleanedDataOutput := CleanData(cleanDataInput)
			// Convert output to Structpb format
			output, err := base.ConvertToStructpb(cleanedDataOutput)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			// Write the output back to the job output
			err = job.Output.Write(ctx, output)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		default:
			job.Error.Error(ctx, fmt.Errorf("not supported task: %s", e.Task))
			continue
		}
	}
	return nil
}
