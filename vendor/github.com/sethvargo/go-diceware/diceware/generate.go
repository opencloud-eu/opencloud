package diceware

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
)

// ErrNumWordsNegative is returned when a negative number of words is requested.
var ErrNumWordsNegative = fmt.Errorf("number of words cannot be negative")

// sides is the number of sides on a die.
var sides = big.NewInt(6)

var _ DicewareGenerator = (*Generator)(nil)

// Generator is the stateful generator which can be used to customize the word
// list and other generation options.
type Generator struct {
	wordList   WordList
	randReader io.Reader
}

// GeneratorInput is used as input to the NewGenerator function.
type GeneratorInput struct {
	// WordList is the word list to use. There are built-in word lists like
	// WordListEffBig (default), WordListEffSmall, and WordListOriginal. You can
	// also bring your own word list by implementing the WordList interface.
	WordList WordList

	// RandReader is an optional reader to use in place of the default
	// (crypto/rand.Reader), which can be used to generate repeatable sets of
	// words
	RandReader io.Reader
}

// NewGenerator creates a new Generator from the specified configuration. If no
// input is given, all the default values are used. This function is safe for
// concurrent use.
func NewGenerator(i *GeneratorInput) (*Generator, error) {
	if i == nil {
		i = new(GeneratorInput)
	}

	if i.WordList == nil {
		i.WordList = WordListEffLarge()
	}

	gen := &Generator{
		wordList:   i.WordList,
		randReader: i.RandReader,
	}

	if gen.randReader == nil {
		gen.randReader = rand.Reader
	}

	return gen, nil
}

// Generate generates a collection of diceware words, specified by the numWords
// parameter.
//
// The algorithm is fast, but it's not designed to be performant, favoring
// entropy over speed.
//
// This function is safe for concurrent use, but there is a possibility of
// concurrent invocations generating overlapping words. To generate multiple
// non-overlapping words, use a single invocation of the function and split the
// resulting string list.
func (g *Generator) Generate(numWords int) ([]string, error) {
	if numWords < 0 {
		return nil, ErrNumWordsNegative
	}

	if typ, ok := g.wordList.(WordListNumWordser); ok {
		if l := typ.NumWords(); numWords > l {
			return nil, fmt.Errorf("number of requested words (%d) cannot exceed the size of the wordlist (%d)",
				numWords, l)
		}
	}

	list := make([]string, 0, numWords)
	seen := make(map[string]struct{}, numWords)

	for i := 0; i < numWords; i++ {
		n, err := g.RollWord(g.wordList.Digits())
		if err != nil {
			return nil, err
		}

		word := g.wordList.WordAt(n)
		if _, ok := seen[word]; ok {
			i--
			continue
		}

		list = append(list, word)
		seen[word] = struct{}{}
	}

	return list, nil
}

// MustGenerate is the same as Generate, but panics on error.
func (g *Generator) MustGenerate(numWords int) []string {
	list, err := g.Generate(numWords)
	if err != nil {
		panic(err)
	}
	return list
}

// Generate - see Generator.Generate for usage.
func Generate(numWords int) ([]string, error) {
	gen, err := NewGenerator(nil)
	if err != nil {
		return nil, err
	}
	return gen.Generate(numWords)
}

// MustGenerate - see Generator.MustGenerate for usage.
func MustGenerate(numWords int) []string {
	gen, err := NewGenerator(nil)
	if err != nil {
		panic(err)
	}
	return gen.MustGenerate(numWords)
}

// GenerateWithWordList generates a list of the given number of words from the
// given word list.
func GenerateWithWordList(numWords int, wordList WordList) ([]string, error) {
	gen, err := NewGenerator(&GeneratorInput{
		WordList: wordList,
	})
	if err != nil {
		return nil, err
	}
	return gen.Generate(numWords)
}

// WordAt retrieves the word at the given index from EFF's large wordlist.
//
// Deprecated: Use WordList.WordAt instead.
func WordAt(i int) string {
	return WordListEffLarge().WordAt(i)
}

// RollDie rolls a single 6-sided die and returns a value between [1,6].
//
// Internally this creates a new Generator with a nil configuration and calls
// Generator.RollDie.
func RollDie() (int, error) {
	gen, err := NewGenerator(nil)
	if err != nil {
		return 0, err
	}
	return gen.RollDie()
}

// RollWord rolls and aggregates dice to represent one word in the list. The
// result is the index of the word in the list.
//
// Internally this creates a new Generator with a nil configuration and calls
// Generator.RollWord.
func RollWord(d int) (int, error) {
	gen, err := NewGenerator(nil)
	if err != nil {
		return 0, err
	}
	return gen.RollWord(d)
}

// RollDie rolls a single 6-sided die and returns a value between [1,6].
func (g *Generator) RollDie() (int, error) {
	r, err := rand.Int(g.randReader, sides)
	if err != nil {
		return 0, fmt.Errorf("failed to generate a random number: %w", err)
	}
	return int(r.Int64()) + 1, nil
}

// RollWord rolls and aggregates dice to represent one word in the list. The
// result is the index of the word in the list.
func (g *Generator) RollWord(d int) (int, error) {
	var final int

	for range d {
		res, err := g.RollDie()
		if err != nil {
			return 0, err
		}

		final = final*10 + res
	}

	return final, nil
}
