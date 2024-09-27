package text

import (
	"strconv"
	"strings"
)

// Document Implementation
const (
	ListStarters = "-*+"
)

type MarkdownDocument struct {
	Headers  []Header
	Contents []Content
}

type Header struct {
	Level int
	Text  string
	Size  int
}

type Content struct {
	Type      string
	PlainText string
	Table     Table
	// All lists in the content with all levels in order
	Lists              []List
	BlockStartPosition int
	BlockEndPosition   int
}

type Table struct {
	HeaderText     string
	TableSeparator string
	HeaderRow      string
	Rows           []string
}

// List includes bullet points and numbered lists
type List struct {
	// HeaderText is the text before the list starts
	HeaderText        string
	PreviousLevelList *List
	Text              string
	StartPosition     int
	EndPosition       int
	NextLevelLists    []List
	NextList          *List
	PreviousList      *List
	indentation       int
	isNumeric         bool
}

func buildDocuments(rawRunes []rune) ([]MarkdownDocument, error) {
	var documents []MarkdownDocument

	var previousDocument *MarkdownDocument
	var currentPosition int

	for currentPosition < len(rawRunes) {
		var endPosition int
		var doc MarkdownDocument

		// Build document
		doc, endPosition = buildDocument(rawRunes, previousDocument, currentPosition)
		if len(doc.Contents) > 0 { // Ensure the document has content
			documents = append(documents, doc)
		}

		// Move to the next section
		currentPosition = endPosition
		if len(documents) > 0 {
			previousDocument = &documents[len(documents)-1]
		}
	}

	return documents, nil
}

func buildDocument(rawRunes []rune, previousDocument *MarkdownDocument, startPosition int) (doc MarkdownDocument, endPosition int) {
	var (
		currentPosition    = startPosition
		currentContent     Content
		headers            = make([]*Header, 6)
		currentHeaderLevel int
		end                bool
	)

	// Copy lower-level headers from previousDocument
	if previousDocument != nil {
		for _, prevHeader := range previousDocument.Headers {
			if prevHeader.Level > 0 && prevHeader.Level <= len(headers) {
				headers[prevHeader.Level-1] = &prevHeader
				currentHeaderLevel = prevHeader.Level
			}
		}
	}

	for currentPosition < len(rawRunes) && !end {
		currentContent = Content{}
		block := readBlock(rawRunes, &currentPosition)
		trimmedBlock := strings.TrimSpace(block)

		if len(trimmedBlock) == 0 {
			continue
		}

		if isTable(block) {
			currentContent.Type = "table"
			currentContent.Table = parseTableFromBlock(block)
			currentContent.BlockStartPosition = currentPosition - sizeOfString(block) - 1
			currentContent.BlockEndPosition = currentPosition
			doc.Contents = append(doc.Contents, currentContent)
		} else if isList(block) {
			currentContent.Type = "list"
			currentContent.Lists = parseListFromBlock(block, currentPosition)
			currentContent.BlockStartPosition = currentPosition - sizeOfString(block) - 1
			currentContent.BlockEndPosition = currentPosition
			doc.Contents = append(doc.Contents, currentContent)
		} else {
			if containsHeader(block) {
				var paragraph string
				endPositionOfBlock := currentPosition
				currentPosition -= sizeOfString(block) + 1
				if currentPosition < 0 {
					currentPosition = 0
				}
				currentContent.Type = "plaintext"
				currentContent.BlockStartPosition = currentPosition
				currentContent.BlockEndPosition = currentPosition
				meetHeaderTimes := 0
				for currentPosition < endPositionOfBlock {

					line := readLine(rawRunes, &currentPosition)
					currentContent.BlockEndPosition += sizeOfString(line) + 1

					if isHeader(line) {
						meetHeaderTimes++

						if meetHeaderTimes > 0 && len(paragraph) > 0 {
							currentContent.PlainText = paragraph
							doc.Contents = append(doc.Contents, currentContent)
						}
						if endOfDocument(doc) {
							currentPosition -= sizeOfString(line) + 1
							currentContent.PlainText = paragraph
							if len(currentContent.PlainText) > 0 {
								doc.Contents = append(doc.Contents, currentContent)
							}
							end = true
							break
						}
						header := parseHeader(line)
						currentHeaderLevel = header.Level
						headers[header.Level-1] = &header
					} else {
						if len(line) > 0 {
							paragraph += line + "\n"
						}
					}
				}
				currentContent.PlainText = paragraph
				if len(currentContent.PlainText) > 0 {
					doc.Contents = append(doc.Contents, currentContent)
				}
			} else {
				currentContent.Type = "plaintext"
				currentContent.PlainText = block

				currentContent.BlockStartPosition = currentPosition - sizeOfString(block) - 1
				currentContent.BlockEndPosition = currentPosition
				doc.Contents = append(doc.Contents, currentContent)
			}
		}
	}

	// clear higher level headers that is higher than currentHeaderLevel
	for i := currentHeaderLevel; i < len(headers); i++ {
		headers[i] = nil
	}

	for i := range headers {
		if headers[i] != nil {
			doc.Headers = append(doc.Headers, *headers[i])
			headers[i] = nil // Clear the header after it's added
		}
	}

	return doc, currentPosition
}

// Reader Section //
// Helper functions to read and parse markdown content
// Helper function to read a block of text separated by double newlines
func readBlock(rawRunes []rune, currentPosition *int) string {
	var block []rune
	isEmptyLineCount := 0

	for *currentPosition < len(rawRunes) {
		line := readLine(rawRunes, currentPosition)

		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) == 0 {
			isEmptyLineCount++
			if isEmptyLineCount >= 1 && len(block) > 0 {
				break
			}
		} else {
			isEmptyLineCount = 0
		}

		block = append(block, []rune(line)...)

		if len(line) > 0 {
			block = append(block, '\n')
		}
	}
	return string(block)
}

func readLine(rawRunes []rune, currentPosition *int) string {
	var line []rune
	for *currentPosition < len(rawRunes) && rawRunes[*currentPosition] != '\n' {
		line = append(line, rawRunes[*currentPosition])
		*currentPosition++
	}
	*currentPosition++
	return string(line)
}

// Table Section //
// Helper function to determine if a block is a table
func isTable(block string) bool {
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) == 0 {
			continue
		}
		if !isRow(line) {
			return false
		}
	}
	return true
}

// Helper function to determine if a line starts a table
func isRow(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	if len(trimmedLine) == 0 {
		return false
	}

	// Count occurrences of vertical bars
	barCount := strings.Count(trimmedLine, "|")

	// Check if the line contains at least two vertical bars (indicating multiple cells)
	return barCount >= 1
}

// Helper function to determine if a line is a table separator
func isTableSeparator(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return strings.Contains(trimmedLine, "-|")
}

// Function to parse a table from a block of text
func parseTableFromBlock(block string) Table {
	var table Table
	lines := strings.Split(block, "\n")
	var rows []string
	var headerRow string
	var headerText string

	inHeader := true

	for i, line := range lines {
		// Preserve the original line without trimming spaces

		if i == 0 && !isRow(line) {
			// The first line is the header text if it's not a table row
			headerText = line
			continue
		}

		if isTableSeparator(line) {
			table.TableSeparator = line
			inHeader = false
		} else if isRow(line) {
			// Process table header or data row
			if inHeader {
				headerRow = line
				inHeader = false // Ensure we don't overwrite header row with data
			} else {
				rows = append(rows, line)
			}
		}
	}

	if len(headerRow) > 0 {
		table.HeaderRow = headerRow
	} else if len(rows) > 0 {
		table.HeaderRow = rows[0]
		rows = rows[1:] // Remove the header row from the rows
	}

	table.HeaderText = headerText
	table.Rows = rows

	return table
}

// List Section //
// Helper function to determine if a block is a list
func isList(block string) bool {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		if i < 5 && isListStart(line) {
			return true
		}
	}
	return false
}

// Function to parse a list from a block of text
func parseListFromBlock(block string, currentPosition int) []List {
	headerText := ""
	currentPosition -= sizeOfString(block) + 1
	lines := strings.Split(block, "\n")

	// Accumulate header text until we hit the first list item
	for _, line := range lines {
		if isListStart(line) {
			break
		}
		headerText += line + "\n"
	}

	var lists []List
	for _, line := range lines {
		if isListStart(line) {
			indentLevel := countIndent(line)
			listItem := List{
				HeaderText:     headerText,
				Text:           line,
				NextLevelLists: []List{},
				StartPosition:  currentPosition,
				EndPosition:    currentPosition + sizeOfString(line) - 1,
				indentation:    indentLevel,
				isNumeric:      isNumericList(line),
			}
			lists = append(lists, listItem)

		} else {
			if len(lists) > 0 {
				lists[len(lists)-1].Text += "\n" + line
			}
		}
		currentPosition += sizeOfString(line) + 1
	}

	// Set NextList & PreviousList & NextLevelList & PreviousLevelList
	var lastListAtLevel = make(map[int]*List)
	var indentStack []int
	for i := range lists {
		indent := lists[i].indentation

		// Link to the previous list at the same level
		if prev, ok := lastListAtLevel[indent]; ok {
			if prev.isNumeric == lists[i].isNumeric {
				lists[i].PreviousList = prev
				prev.NextList = &lists[i]
			}
		}

		var prevIndent int
		for j := len(indentStack) - 1; j >= 0; j-- {
			if indentStack[j] < indent {
				prevIndent = indentStack[j]
				break
			}
		}

		// Link to the previous level list if exists
		if prevLevelList, ok := lastListAtLevel[prevIndent]; ok {
			if prevLevelList.indentation != lists[i].indentation {
				lists[i].PreviousLevelList = prevLevelList
				prevLevelList.NextLevelLists = append(prevLevelList.NextLevelLists, lists[i])
			}
		}

		// Update the last list at this level
		lastListAtLevel[indent] = &lists[i]

		if len(indentStack) == 0 || indentStack[len(indentStack)-1] != indent {
			indentStack = append(indentStack, indent)
		}
	}

	return lists
}

// Helper function to count indentation level
func countIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// Helper function to determine if a line starts a list
func isListStart(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return len(trimmedLine) > 0 && (strings.Contains(ListStarters, string(trimmedLine[0])) || isNumericList(trimmedLine))
}

// Helper function to determine if a line starts a numeric list (e.g., "1. Item")
func isNumericList(line string) bool {
	parts := strings.SplitN(line, ".", 2)
	if len(parts) < 2 {
		return false
	}
	_, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	return err == nil
}

// Header Section //
func isHeader(line string) bool {
	trimmedLine := strings.TrimLeft(line, " \t") // Remove leading whitespace
	return len(trimmedLine) > 0 && trimmedLine[0] == '#'
}

// Helper function to parse a header line into a Header struct
func parseHeader(line string) Header {
	trimmedLine := strings.TrimLeft(line, " \t") // Remove leading whitespace
	level := 0
	for _, char := range trimmedLine {
		if char == '#' {
			level++
		} else {
			break
		}
	}
	return Header{Level: level, Text: line, Size: sizeOfString(line)}
}

func containsHeader(block string) bool {
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		if isHeader(line) {
			return true
		}
	}
	return false
}

// Helper function to determine if content should be finalized based on its type
func endOfDocument(currentDoc MarkdownDocument) bool {
	return len(currentDoc.Contents) > 0
}

func sizeOfString(text string) int {
	return len([]rune(text))
}
