package text

import (
	"fmt"
	"testing"

	"github.com/frankban/quicktest"
)

func TestChunkText(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name   string
		input  ChunkTextInput
		output ChunkTextOutput
		err    error
	}{
		{
			name: "chunk text by token",
			input: ChunkTextInput{
				Text: "Hello world.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Token",
						ChunkSize:   512,
						ModelName:   "gpt-3.5-turbo",
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "Hello world.",
						StartPosition: 0,
						EndPosition:   11,
						TokenCount:    3,
					},
				},
				ChunkNum:         1,
				TokenCount:       3,
				ChunksTokenCount: 3,
			},
			err: nil,
		},
		{
			name: "chunk text by recursive method",
			input: ChunkTextInput{
				Text: "This is a simple test for chunking text into pieces.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Recursive",
						ChunkSize:    50,
						ChunkOverlap: 10,
						ModelName:    "gpt-3.5-turbo",
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "This is a simple test for chunking text into pi",
						StartPosition: 0,
						EndPosition:   47,
						TokenCount:    10,
					},
					{
						Text:          "text into pieces.",
						StartPosition: 10,
						EndPosition:   27,
						TokenCount:    5,
					},
				},
				ChunkNum:         2,
				TokenCount:       15,
				ChunksTokenCount: 15,
			},
			err: nil,
		},
		{
			name: "chunk overlap must be less than chunk size",
			input: ChunkTextInput{
				Text: "Hello world.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Token",
						ChunkSize:    5,
						ChunkOverlap: 5,
						ModelName:    "gpt-3.5-turbo",
					},
				},
			},
			output: ChunkTextOutput{},
			err:    fmt.Errorf("ChunkOverlap must be less than ChunkSize when using Token method"),
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output, err := chunkText(tc.input)
			if tc.err != nil {
				c.Assert(err, quicktest.ErrorIs, tc.err)
			} else {
				c.Assert(err, quicktest.IsNil)
				c.Check(output, quicktest.DeepEquals, tc.output)
			}
		})
	}
}

func TestChunkMarkdown(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name   string
		input  ChunkTextInput
		output ChunkTextOutput
		err    error
	}{
		{
			name: "chunk markdown text",
			input: ChunkTextInput{
				Text: "## Heading\n\nThis is a test paragraph. It has multiple sentences.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ChunkSize:    50,
						ChunkOverlap: 5,
						ModelName:    "gpt-3.5-turbo",
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "## Heading\n\nThis is a test paragraph.",
						StartPosition: 0,
						EndPosition:   36,
						TokenCount:    8,
					},
					{
						Text:          " It has multiple sentences.",
						StartPosition: 31,
						EndPosition:   57,
						TokenCount:    5,
					},
				},
				ChunkNum:         2,
				TokenCount:       13,
				ChunksTokenCount: 13,
			},
			err: nil,
		},
		{
			name: "empty markdown input",
			input: ChunkTextInput{
				Text: "",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ChunkSize:    100,
						ChunkOverlap: 0,
						ModelName:    "gpt-3.5-turbo",
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "",
						StartPosition: 0,
						EndPosition:   0,
						TokenCount:    0,
					},
				},
				ChunkNum:         1,
				TokenCount:       0,
				ChunksTokenCount: 0,
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			output, err := chunkMarkdown(tc.input)
			if tc.err != nil {
				c.Assert(err, quicktest.ErrorIs, tc.err)
			} else {
				c.Assert(err, quicktest.IsNil)
				c.Check(output, quicktest.DeepEquals, tc.output)
			}
		})
	}
}

func TestSetDefault(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name     string
		input    Setting
		expected Setting
	}{
		{
			name: "all defaults",
			input: Setting{
				ChunkSize:    0,
				ChunkOverlap: 0,
				ModelName:    "",
			},
			expected: Setting{
				ChunkSize:    512,
				ChunkOverlap: 100,
				ModelName:    "gpt-3.5-turbo",
				AllowedSpecial:    []string{},
				DisallowedSpecial: []string{"all"},
				Separators:        []string{"\n\n", "\n", " ", ""},
			},
		},
		{
			name: "some defaults",
			input: Setting{
				ChunkSize:    256,
				ChunkOverlap: 0,
				ModelName:    "gpt-4",
			},
			expected: Setting{
				ChunkSize:    256,
				ChunkOverlap: 100,
				ModelName:    "gpt-4",
				AllowedSpecial:    []string{},
				DisallowedSpecial: []string{"all"},
				Separators:        []string{"\n\n", "\n", " ", ""},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			tc.input.SetDefault()
			c.Check(tc.input, quicktest.DeepEquals, tc.expected)
		})
	}
}

func TestGetChunkPositions(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name           string
		rawText       string
		chunkText     string
		expectedStart int
		expectedEnd   int
	}{
		{
			name:           "simple case",
			rawText:       "Hello, this is a test.",
			chunkText:     "this is a test.",
			expectedStart: 7,
			expectedEnd:   24,
		},
		{
			name:           "no match",
			rawText:       "Goodbye, see you later.",
			chunkText:     "not present",
			expectedStart: 0,
			expectedEnd:   0,
		},
	}

	calculator := PositionCalculator{}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			rawRunes := []rune(tc.rawText)
			chunkRunes := []rune(tc.chunkText)

			startPosition, endPosition := calculator.getChunkPositions(rawRunes, chunkRunes, 0)

			c.Check(startPosition, quicktest.Equals, tc.expectedStart)
			c.Check(endPosition, quicktest.Equals, tc.expectedEnd)
		})
	}
}
