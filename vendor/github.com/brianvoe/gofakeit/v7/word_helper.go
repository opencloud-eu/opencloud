package gofakeit

import (
	"strings"
)

// This will look at a few things to determine what kind of article to use for the word
func getArticle(word string) string {
	// If nothing is passed return empty
	if word == "" {
		return ""
	}

	word = strings.ToLower(word)
	letters := strings.Split(word, "")
	firstLetter := ""
	secondLetter := ""
	if len(letters) > 0 {
		firstLetter = letters[0]
	}
	if len(letters) > 1 {
		secondLetter = letters[1]
	}

	// If the word starts with a, e, i, o, use an article
	if firstLetter == "a" || firstLetter == "e" || firstLetter == "i" || firstLetter == "o" {
		return "an"
	}

	// If the word starts with a u and n or l, use an article
	if firstLetter == "u" {
		if secondLetter == "n" || secondLetter == "l" {
			return "an"
		}
	}

	// If the word starts with a vowel, use an article
	if firstLetter == "h" {
		if secondLetter == "i" {
			return "an"
		}
	}

	return "a"
}
