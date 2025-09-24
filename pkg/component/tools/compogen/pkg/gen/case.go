package gen

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// List of words to keep unchanged.
var unchangedWords = map[string]bool{
	"WhatsApp": true,
	"BigQuery": true,
	"OAuth":    true,
}

// List of words to keep in uppercase
var uppercaseWords = map[string]bool{
	"id":   true, // Identifier
	"uid":  true, // Unique Identifier
	"url":  true, // Uniform resource locator
	"uri":  true, // Unifor resource identifier
	"api":  true, // Application programming interface
	"html": true, // Hyper-text markup language
	"json": true, // JavaScript object notation
	"sql":  true, // Structured query language
	"ssl":  true, // Secure sockets layer
	"tls":  true, // Transport layer security
	"cfg":  true, // Classifier-free guidance
	"ocr":  true, // Optical character recognition
	"cc":   true, // Carbon copy
	"bcc":  true, // Blind carbon copy
	"pr":   true, // Pull request
	"sha":  true, // Secure hashing algorithm
	"pdf":  true, // Portable document format
	"fps":  true, // Frames per second
	"ttl":  true, // Time to live
}

// List of words to keep in lowercase (articles, conjunctions, prepositions)
var lowercaseWords = map[string]bool{
	// Following Title Case as defined on
	// [Wikipedia](https://en.wikipedia.org/wiki/Letter_case#Headings_and_publication_titles).

	// Articles
	"a":   true,
	"an":  true,
	"the": true,
	// Short prepositions
	"at":    true,
	"by":    true,
	"in":    true,
	"on":    true,
	"to":    true,
	"of":    true,
	"up":    true,
	"for":   true,
	"off":   true,
	"out":   true,
	"from":  true,
	"with":  true,
	"over":  true,
	"under": true,
	"into":  true,
	"per":   true,
	// Conjunctions
	"and": true,
	"nor": true,
	"but": true,
	"or":  true,
	"yet": true,
	"so":  true,
	"if":  true,
}

// titleCase keeps certain words (articles, conjunctions, prepositions) lowercase.
func titleCase(s string) string {
	// Create a Title case transformer
	titleCaser := cases.Title(language.English)

	// Split the string into words
	words := strings.Fields(s)

	// Apply title case to each word
	for i, word := range words {
		lower := strings.ToLower(word)
		lowerSingular := strings.TrimSuffix(lower, "s")

		switch {
		case unchangedWords[word]:
			continue
		case uppercaseWords[lower]:
			words[i] = strings.ToUpper(word)
		case uppercaseWords[lowerSingular]:
			// If the singular version of the word is in the uppercaseWords
			// list, keep the singular version uppercased.
			words[i] = strings.ToUpper(lowerSingular) + "s"
		case i != 0 && (i != len(words)-1) && lowercaseWords[lower]:
			// Keep the word lowercase if it's not the first or last word and
			// is in the lowercaseWords list.
			words[i] = lower
		default:
			words[i] = titleCaser.String(word)
		}
	}

	// Join the words with spaces to maintain spaces between words.
	return strings.Join(words, " ")
}
