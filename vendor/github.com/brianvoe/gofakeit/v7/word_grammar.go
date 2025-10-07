package gofakeit

import (
	"unicode"
)

// SentenceSimple will generate a random simple sentence
func SentenceSimple() string { return sentenceSimple(GlobalFaker) }

// SentenceSimple will generate a random simple sentence
func (f *Faker) SentenceSimple() string { return sentenceSimple(f) }

func sentenceSimple(f *Faker) string {
	// simple sentence consists of a noun phrase and a verb phrase
	str := phraseNoun(f) + " " + phraseVerb(f) + "."

	// capitalize the first letter
	strR := []rune(str)
	strR[0] = unicode.ToUpper(strR[0])
	return string(strR)
}

func addWordGrammerLookup() {
	AddFuncLookup("sentencesimple", Info{
		Display:     "Simple Sentence",
		Category:    "word",
		Description: "Group of words that expresses a complete thought",
		Example:     "A tribe fly the lemony kitchen.",
		Output:      "string",
		Aliases:     []string{"basic statement", "complete thought", "independent clause", "grammar element"},
		Keywords:    []string{"sentence", "simple", "complete", "thought", "grammar", "basic", "expresses", "independent", "clause", "statement"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return sentenceSimple(f), nil
		},
	})
}
