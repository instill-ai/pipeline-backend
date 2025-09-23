package text

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/frankban/quicktest"
)

func TestChunkText_LongMD(t *testing.T) {
	c := quicktest.New(t)

	inputData, err := os.ReadFile("testdata/chunk-markdown-input.md")
	c.Assert(err, quicktest.IsNil)

	// Expectations are complex and hence stored in a file
	expectedData, err := os.ReadFile("testdata/chunk-markdown-output.json")
	c.Assert(err, quicktest.IsNil)

	var want ChunkTextOutput
	err = json.Unmarshal(expectedData, &want)
	c.Assert(err, quicktest.IsNil)

	input := ChunkTextInput{
		Text: string(inputData),
		Strategy: Strategy{
			Setting: Setting{
				ChunkMethod:  "Markdown",
				ModelName:    "gpt-4",
				ChunkSize:    1024,
				ChunkOverlap: 200,
			},
		},
	}

	// Run the algorithm
	got, err := chunkMarkdown(input)
	c.Assert(err, quicktest.IsNil)

	// Compare the results
	c.Check(got.ChunkNum, quicktest.Equals, want.ChunkNum)
	c.Check(got.TokenCount, quicktest.Equals, want.TokenCount)
	c.Check(got.ChunksTokenCount, quicktest.Equals, want.ChunksTokenCount)
	c.Check(len(got.TextChunks), quicktest.Equals, len(want.TextChunks))

	// Compare each chunk (ignoring the actual text content for performance)
	for i, chunk := range got.TextChunks {
		// NOTE: We don't compare the Text field as it can be large and the
		// positions/tokens are sufficient

		wantChunk := want.TextChunks[i]
		c.Check(chunk.StartPosition, quicktest.Equals, wantChunk.StartPosition)
		c.Check(chunk.EndPosition, quicktest.Equals, wantChunk.EndPosition)
		c.Check(chunk.TokenCount, quicktest.Equals, wantChunk.TokenCount)

	}
}

func TestChunkText(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name   string
		input  ChunkTextInput
		output ChunkTextOutput
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
		},
		{
			name: "chunk text by markdown",
			input: ChunkTextInput{
				Text: "Hello world.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod:  "Markdown",
						ModelName:    "gpt-3.5-turbo",
						ChunkSize:    5,
						ChunkOverlap: 2,
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "\nHello\nworld\n",
						StartPosition: 0,
						EndPosition:   11,
						TokenCount:    5,
					},
				},
				ChunkNum:         1,
				TokenCount:       3,
				ChunksTokenCount: 5,
			},
		},
		{
			name: "chunk text by recursive",
			input: ChunkTextInput{
				Text: "Hello world.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Recursive",
						ModelName:   "gpt-3.5-turbo",
						ChunkSize:   5,
						Separators:  []string{" ", "."},
					},
				},
			},
			output: ChunkTextOutput{
				TextChunks: []TextChunk{
					{
						Text:          "Hello",
						StartPosition: 0,
						EndPosition:   4,
						TokenCount:    1,
					},
					{
						Text:          "world",
						StartPosition: 6,
						EndPosition:   10,
						TokenCount:    1,
					},
				},
				ChunkNum:         2,
				TokenCount:       3,
				ChunksTokenCount: 2,
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			var output ChunkTextOutput
			err := error(nil)
			if tc.input.Strategy.Setting.ChunkMethod == "Markdown" {
				output, err = chunkMarkdown(tc.input)
			} else {
				output, err = chunkText(tc.input)
			}
			c.Assert(err, quicktest.IsNil)
			c.Check(output, quicktest.DeepEquals, tc.output)
		})
	}
}

func Test_ChunkPositionCalculator(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name                   string
		positionCalculatorType string
		rawTextFilePath        string
		chunkTextFilePath      string
		expectStartPosition    int
		expectEndPosition      int
	}{
		{
			name:                   "Chinese text with NOT Markdown Chunking 1",
			positionCalculatorType: "PositionCalculator",
			rawTextFilePath:        "testdata/chinese/text1.txt",
			chunkTextFilePath:      "testdata/chinese/chunk1_1.txt",
			expectStartPosition:    0,
			expectEndPosition:      35,
		},
		{
			name:                   "Chinese text with NOT Markdown Chunking 2",
			positionCalculatorType: "PositionCalculator",
			rawTextFilePath:        "testdata/chinese/text1.txt",
			chunkTextFilePath:      "testdata/chinese/chunk1_2.txt",
			expectStartPosition:    26,
			expectEndPosition:      46,
		},
		{
			name:                   "Chinese text with NOT Markdown Chunking 3",
			positionCalculatorType: "PositionCalculator",
			rawTextFilePath:        "testdata/chinese/text1.txt",
			chunkTextFilePath:      "testdata/chinese/chunk1_3.txt",
			expectStartPosition:    49,
			expectEndPosition:      80,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			var calculator ChunkPositionCalculator
			if tc.positionCalculatorType == "PositionCalculator" {
				calculator = PositionCalculator{}
			}
			rawTextBytes, err := os.ReadFile(tc.rawTextFilePath)
			c.Assert(err, quicktest.IsNil)
			rawTextRunes := []rune(string(rawTextBytes))

			chunkText, err := os.ReadFile(tc.chunkTextFilePath)
			c.Assert(err, quicktest.IsNil)

			chunkTextRunes := []rune(string(chunkText))

			startPosition, endPosition := calculator.getChunkPositions(rawTextRunes, chunkTextRunes, 0)

			c.Assert(startPosition, quicktest.Equals, tc.expectStartPosition)
			c.Assert(endPosition, quicktest.Equals, tc.expectEndPosition)

		})
	}
}

func Test_ChunkPositions(t *testing.T) {

	c := quicktest.New(t)

	testCases := []struct {
		name            string
		rawTextFilePath string
	}{
		{
			name:            "test",
			rawTextFilePath: "testdata/test.txt",
		},
	}

	for _, tc := range testCases {
		rawTextBytes, err := os.ReadFile(tc.rawTextFilePath)
		c.Assert(err, quicktest.IsNil)

		input := ChunkTextInput{
			Text: string(rawTextBytes),
			Strategy: Strategy{
				Setting: Setting{
					ChunkMethod:  "Recursive",
					ChunkSize:    800,
					ChunkOverlap: 200,
					ModelName:    "gpt-4",
				},
			},
		}

		output, err := chunkText(input)

		c.Assert(err, quicktest.IsNil)

		for i, chunk := range output.TextChunks {
			c.Assert(chunk.TokenCount, quicktest.Not(quicktest.Equals), 0)
			if i != 0 {
				c.Assert(chunk.StartPosition, quicktest.Not(quicktest.Equals), 0)
			}
			c.Assert(chunk.EndPosition, quicktest.Not(quicktest.Equals), 0)
			c.Assert(chunk.Text, quicktest.Not(quicktest.Equals), "")

			positionChecker := chunk.StartPosition < chunk.EndPosition
			c.Assert(positionChecker, quicktest.Equals, true)

			if i > 0 {
				increaseChecker := output.TextChunks[i].StartPosition > output.TextChunks[i-1].StartPosition
				c.Assert(increaseChecker, quicktest.Equals, true)
			}
		}

	}
}
