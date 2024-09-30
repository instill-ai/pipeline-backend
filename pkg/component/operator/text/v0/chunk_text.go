package text

import (
	"fmt"
	"reflect"

	"github.com/tmc/langchaingo/textsplitter"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

type ChunkTextInput struct {
	Text     string   `json:"text"`
	Strategy Strategy `json:"strategy"`
}

type Strategy struct {
	Setting Setting `json:"setting"`
}

type Setting struct {
	ChunkMethod       string   `json:"chunk-method,omitempty"`
	ChunkSize         int      `json:"chunk-size,omitempty"`
	ChunkOverlap      int      `json:"chunk-overlap,omitempty"`
	ModelName         string   `json:"model-name,omitempty"`
	AllowedSpecial    []string `json:"allowed-special,omitempty"`
	DisallowedSpecial []string `json:"disallowed-special,omitempty"`
	Separators        []string `json:"separators,omitempty"`
	KeepSeparator     bool     `json:"keep-separator,omitempty"`
	CodeBlocks        bool     `json:"code-blocks,omitempty"`
}

type ChunkTextOutput struct {
	ChunkNum         int         `json:"chunk-num"`
	TextChunks       []TextChunk `json:"text-chunks"`
	TokenCount       int         `json:"token-count"`
	ChunksTokenCount int         `json:"chunks-token-count"`
}

type TextChunk struct {
	Text          string `json:"text"`
	StartPosition int    `json:"start-position"`
	EndPosition   int    `json:"end-position"`
	TokenCount    int    `json:"token-count"`
}

func (s *Setting) SetDefault() {
	if s.ChunkSize == 0 {
		s.ChunkSize = 512
	}
	if s.ChunkOverlap == 0 {
		s.ChunkOverlap = 100
	}
	if s.ModelName == "" {
		s.ModelName = "gpt-3.5-turbo"
	}
	if s.AllowedSpecial == nil {
		s.AllowedSpecial = []string{}
	}
	if s.DisallowedSpecial == nil {
		s.DisallowedSpecial = []string{"all"}
	}
	if s.Separators == nil {
		s.Separators = []string{"\n\n", "\n", " ", ""}
	}
}

type TextSplitter interface {
	SplitText(text string) ([]string, error)
}

func chunkText(input ChunkTextInput) (ChunkTextOutput, error) {
	var split TextSplitter
	setting := input.Strategy.Setting
	setting.SetDefault()

	var output ChunkTextOutput
	var positionCalculator ChunkPositionCalculator

	switch setting.ChunkMethod {
	case "Token":
		positionCalculator = PositionCalculator{}
		if setting.ChunkOverlap >= setting.ChunkSize {
			err := fmt.Errorf("ChunkOverlap must be less than ChunkSize when using Token method")
			return output, err
		}

		split = textsplitter.NewTokenSplitter(
			textsplitter.WithChunkSize(setting.ChunkSize),
			textsplitter.WithChunkOverlap(setting.ChunkOverlap),
			textsplitter.WithModelName(setting.ModelName),
			textsplitter.WithAllowedSpecial(setting.AllowedSpecial),
			textsplitter.WithDisallowedSpecial(setting.DisallowedSpecial),
		)
	case "Recursive":
		positionCalculator = PositionCalculator{}
		split = textsplitter.NewRecursiveCharacter(
			textsplitter.WithSeparators(setting.Separators),
			textsplitter.WithChunkSize(setting.ChunkSize),
			textsplitter.WithChunkOverlap(setting.ChunkOverlap),
			textsplitter.WithKeepSeparator(setting.KeepSeparator),
		)
	}

	chunks, err := split.SplitText(input.Text)
	if err != nil {
		return output, err
	}
	output.ChunkNum = len(chunks)

	tkm, err := tiktoken.EncodingForModel(setting.ModelName)
	if err != nil {
		return output, err
	}

	totalTokenCount := 0
	startScanPosition := 0
	rawRunes := []rune(input.Text)
	for _, chunk := range chunks {
		chunkRunes := []rune(chunk)

		startPosition, endPosition := positionCalculator.getChunkPositions(rawRunes, chunkRunes, startScanPosition)

		if shouldScanRawTextFromPreviousChunk(startPosition, endPosition) {
			previousChunkIndex := len(output.TextChunks) - 1
			previousChunk := output.TextChunks[previousChunkIndex]
			startPosition, endPosition = positionCalculator.getChunkPositions(rawRunes, chunkRunes, previousChunk.StartPosition+1)
		}

		if startPosition == endPosition {
			continue
		}

		token := tkm.Encode(chunk, setting.AllowedSpecial, setting.DisallowedSpecial)

		output.TextChunks = append(output.TextChunks, TextChunk{
			Text:          chunk,
			StartPosition: startPosition,
			EndPosition:   endPosition,
			TokenCount:    len(token),
		})
		totalTokenCount += len(token)
		startScanPosition = startPosition + 1
	}

	if len(output.TextChunks) == 0 {
		token := tkm.Encode(input.Text, setting.AllowedSpecial, setting.DisallowedSpecial)

		output.TextChunks = append(output.TextChunks, TextChunk{
			Text:          input.Text,
			StartPosition: 0,
			EndPosition:   len(rawRunes) - 1,
			TokenCount:    len(token),
		})
		output.ChunkNum = 1
		totalTokenCount = len(token)
	}

	originalTextToken := tkm.Encode(input.Text, setting.AllowedSpecial, setting.DisallowedSpecial)
	output.TokenCount = len(originalTextToken)
	output.ChunksTokenCount = totalTokenCount

	return output, nil
}

func chunkMarkdown(input ChunkTextInput) (ChunkTextOutput, error) {
	var output ChunkTextOutput
	setting := input.Strategy.Setting
	setting.SetDefault()

	sp := NewMarkdownTextSplitter(setting.ChunkSize, setting.ChunkOverlap, input.Text)

	err := sp.Validate()

	if err != nil {
		return output, fmt.Errorf("failed to validate MarkdownTextSplitter: %w", err)
	}

	chunks, err := sp.SplitText()

	if err != nil {
		return output, fmt.Errorf("failed to split text: %w", err)
	}

	tkm, err := tiktoken.EncodingForModel(setting.ModelName)

	if err != nil {
		return output, fmt.Errorf("failed to get encoding for model: %w", err)
	}

	totalTokenCount := 0
	for _, chunk := range chunks {
		token := tkm.Encode(chunk.Chunk, setting.AllowedSpecial, setting.DisallowedSpecial)

		output.TextChunks = append(output.TextChunks, TextChunk{
			Text:          chunk.Chunk,
			StartPosition: chunk.ContentStartPosition,
			EndPosition:   chunk.ContentEndPosition,
			TokenCount:    len(token),
		})
		totalTokenCount += len(token)
	}

	if len(output.TextChunks) == 0 {
		token := tkm.Encode(input.Text, setting.AllowedSpecial, setting.DisallowedSpecial)

		output.TextChunks = append(output.TextChunks, TextChunk{
			Text:          input.Text,
			StartPosition: 0,
			EndPosition:   len([]rune(input.Text)) - 1,
			TokenCount:    len(token),
		})
		output.ChunkNum = 1
		totalTokenCount = len(token)
	}

	originalTextToken := tkm.Encode(input.Text, setting.AllowedSpecial, setting.DisallowedSpecial)
	output.TokenCount = len(originalTextToken)
	output.ChunksTokenCount = totalTokenCount
	output.ChunkNum = len(output.TextChunks)

	return output, nil
}

func shouldScanRawTextFromPreviousChunk(startPosition, endPosition int) bool {
	return startPosition == 0 && endPosition == 0
}

type ChunkPositionCalculator interface {
	getChunkPositions(rawText, chunk []rune, startScanPosition int) (startPosition int, endPosition int)
}

type PositionCalculator struct{}

func (PositionCalculator) getChunkPositions(rawText, chunk []rune, startScanPosition int) (startPosition int, endPosition int) {

	for i := startScanPosition; i < len(rawText); i++ {
		if rawText[i] == chunk[0] {

			if i+len(chunk) > len(rawText) {
				break
			}

			if reflect.DeepEqual(rawText[i:i+len(chunk)], chunk) {
				startPosition = i
				endPosition = len(chunk) + i - 1
				break
			}
		}
	}
	return startPosition, endPosition
}
