package text

import (
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// About mergeChunks
// When the current chunk add the next chunk is smaller than the chunk size, we merge them.
// After merging, we need to check if the merged chunk add the next chunk is still smaller than the chunk size.
// If yes, we continue to merge with the next chunk.
// When the merged chunk add the next chunk is larger than the chunk size, we add the merged chunk to the result.
// And, we start from the unmerged chunk.

// About header
// If the current chunk and the next chunk have different headers, we merge them with the next chunk's header difference.
// e.g. currentChunk.PrependHeader = "## Header 1\n### Header 2"
// nextChunk.PrependHeader = "## Header 1\n### Header 3"
// headerDiff(currentChunk.PrependHeader, nextChunk.PrependHeader) = "\n### Header 3"

// About overlap
// When the previous merging chunk is only one chunk, it means the previous chunk is too long to combine, which means the current chunk will have overlap with the previous chunk.
// So, when the len of previous merging chunk is over 1, we need to add the overlap part to the current chunk.
// To add the overlap part, we have to make sure the position is correct. It means we need to add the overlap part one by one from the merging chunk's last chunk.
type mergingChunks struct {
	Chunks             []ContentChunk
	CollectedTokenSize int
}

type mergedChunk struct {
	// Chunk is the content of the chunk that includes the prepend header
	Chunk string
	// ContentStartPosition is the start position of the content in the raw text
	ContentStartPosition int
	// ContentEndPosition is the end position of the content in the raw text
	ContentEndPosition int
}

// mergeChunks groups chunks to have a token count under the specified chunk
// size.
func mergeChunks(chunks []ContentChunk, inputStruct ChunkTextInput, tkm *tiktoken.Tiktoken) []mergedChunk {
	if len(chunks) <= 1 {
		mergedChunks := []mergedChunk{}
		for _, chunk := range chunks {
			mergedChunks = append(mergedChunks, mergedChunk{
				Chunk:                chunk.PrependHeader + chunk.Chunk,
				ContentStartPosition: chunk.ContentStartPosition,
				ContentEndPosition:   chunk.ContentEndPosition,
			})
		}
		return mergedChunks
	}

	// The merging process is divided in 2 steps in order to compute the
	// positions of each chunk taking into account the overlap.
	mergingChunks := collectMergingChunks(chunks, inputStruct.Strategy.Setting, tkm)
	mergedChunks := processMergingChunks(mergingChunks, inputStruct, tkm)

	return mergedChunks
}

// collectMergingChunks combines the chunks so their token size is less than
// the chunk size in the settings. When adjacent chunks share part or all of
// their headers, the token count computes only the header diff.
func collectMergingChunks(chunks []ContentChunk, setting Setting, tkm *tiktoken.Tiktoken) []mergingChunks {
	var collectedMergingChunks []mergingChunks

	for i := 0; i < len(chunks); i++ {
		currentChunk := chunks[i]
		prependedChunk := currentChunk.PrependHeader + currentChunk.Chunk

		currentMergingChunk := mergingChunks{
			Chunks:             []ContentChunk{currentChunk},
			CollectedTokenSize: getTokenSize(prependedChunk, setting, tkm),
		}

		nextIndex := i + 1
		for nextIndex < len(chunks) {
			nextChunk := chunks[nextIndex]

			potentialSize := currentMergingChunk.CollectedTokenSize
			if currentChunk.PrependHeader != nextChunk.PrependHeader {
				diffHeader := headerDiff(currentChunk.PrependHeader, nextChunk.PrependHeader)
				addedChunk := diffHeader + nextChunk.Chunk
				potentialSize += getTokenSize(addedChunk, setting, tkm)
			} else {
				potentialSize += getTokenSize(nextChunk.Chunk, setting, tkm)
			}

			// We need to leave the overlap part for the next chunk
			cannotOverSize := setting.ChunkSize
			if len(collectedMergingChunks) > 0 {
				previousCollectedChunk := collectedMergingChunks[len(collectedMergingChunks)-1]
				if len(previousCollectedChunk.Chunks) > 1 {
					cannotOverSize = setting.ChunkSize - setting.ChunkOverlap
				}
			}

			if potentialSize > cannotOverSize {
				break
			}

			currentMergingChunk.Chunks = append(currentMergingChunk.Chunks, nextChunk)
			currentMergingChunk.CollectedTokenSize = potentialSize
			nextIndex++

			// If the next chunk has no header, we use the current chunk's
			// header to avoid the duplicate header.
			if nextChunk.PrependHeader == "" {
				nextChunk.PrependHeader = currentChunk.PrependHeader
			}

			currentChunk = nextChunk
		}

		collectedMergingChunks = append(collectedMergingChunks, currentMergingChunk)
		i = nextIndex - 1
	}

	return collectedMergingChunks
}

// ProcessMergingChunks

// About overlap
// When the previous merging chunk is only one chunk, it means the previous chunk is too long to combine, which means the current chunk will have overlap with the previous chunk.
// So, when the len of previous merging chunk is over 1, we need to add the overlap part to the current chunk.
// To add the overlap part, we have to make sure the position is correct. It means we need to add the overlap part one by one from the merging chunk's last chunk.
func processMergingChunks(mergingChunks []mergingChunks, inputStruct ChunkTextInput, tkm *tiktoken.Tiktoken) []mergedChunk {
	var mergedChunks []mergedChunk

	firstMergingChunk := mergingChunks[0]
	firstMergedChunk := mergeMergingChunks(firstMergingChunk)
	mergedChunks = append(mergedChunks, firstMergedChunk)

	if len(mergingChunks) < 2 {
		return mergedChunks
	}

	// Merge the other chunks taking into account the chunk overlap.
	for prevIdx, currentMergingChunk := range mergingChunks[1:] {
		previousMergingChunk := mergingChunks[prevIdx]

		if len(previousMergingChunk.Chunks) > 1 {
			overlapText, overlapPosition := getOverlapForSameHeader(previousMergingChunk, currentMergingChunk, inputStruct.Strategy.Setting, tkm)
			if overlapText != "" {
				currentMergingChunk.Chunks[0].Chunk = overlapText + currentMergingChunk.Chunks[0].Chunk
				currentMergingChunk.Chunks[0].ContentStartPosition = overlapPosition
			}
		}

		mergedChunk := mergeMergingChunks(currentMergingChunk)
		mergedChunks = append(mergedChunks, mergedChunk)
	}

	return mergedChunks
}

// mergeMergingChunks merges the merging chunks into a merged chunk
func mergeMergingChunks(mergingChunks mergingChunks) mergedChunk {
	mergedChunk := mergedChunk{}
	for i := range mergingChunks.Chunks {
		currentChunk := mergingChunks.Chunks[i]
		if i == 0 {
			mergedChunk.Chunk = currentChunk.PrependHeader + "\n" + currentChunk.Chunk
		} else {
			previousChunk := mergingChunks.Chunks[i-1]
			if currentChunk.PrependHeader != previousChunk.PrependHeader {
				diffHeader := headerDiff(previousChunk.PrependHeader, currentChunk.PrependHeader)
				mergedChunk.Chunk += "\n" + diffHeader + "\n" + currentChunk.Chunk
			} else {
				mergedChunk.Chunk += "\n" + currentChunk.Chunk
			}
		}
	}
	mergedChunk.ContentStartPosition = mergingChunks.Chunks[0].ContentStartPosition
	mergedChunk.ContentEndPosition = mergingChunks.Chunks[len(mergingChunks.Chunks)-1].ContentEndPosition

	return mergedChunk
}

// headerDiff returns the difference part of the headers
func headerDiff(currentChunkHeader, nextChunkHeader string) string {
	currentHeaders := strings.Split(strings.TrimSpace(currentChunkHeader), "\n")
	nextHeaders := strings.Split(strings.TrimSpace(nextChunkHeader), "\n")

	minLen := min(len(currentHeaders), len(nextHeaders))

	for i := range minLen {
		if currentHeaders[i] != nextHeaders[i] {
			return strings.Join(nextHeaders[i:], "\n")
		}
	}

	// If no difference found but next has more headers, return the remaining headers
	if len(nextHeaders) > len(currentHeaders) {
		return strings.Join(nextHeaders[len(currentHeaders):], "\n")
	}

	return ""
}

func getOverlapForSameHeader(previousMergingChunk mergingChunks, currentMergingChunks mergingChunks, setting Setting, tkm *tiktoken.Tiktoken) (string, int) {
	overlapText := ""
	overlapSize := setting.ChunkOverlap
	var overlapPosition int

	i := len(previousMergingChunk.Chunks) - 1
	currentMergingChunk := currentMergingChunks.Chunks[0]
	for i >= 0 {
		if previousMergingChunk.Chunks[i].PrependHeader == currentMergingChunk.PrependHeader {
			sizeChecker := previousMergingChunk.Chunks[i].Chunk + overlapText
			if getTokenSize(sizeChecker, setting, tkm) > overlapSize {
				return overlapText, overlapPosition
			}
			overlapText = previousMergingChunk.Chunks[i].Chunk + "\n" + overlapText
			overlapPosition = previousMergingChunk.Chunks[i].ContentStartPosition
		}
		if overlapText == "" {
			return "", 0
		}
		i--
	}
	return overlapText, overlapPosition
}

// getTokenSize returns the token size of the text
func getTokenSize(text string, setting Setting, tkm *tiktoken.Tiktoken) int {
	return len(tkm.Encode(text, setting.AllowedSpecial, setting.DisallowedSpecial))
}
