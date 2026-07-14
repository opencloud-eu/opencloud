package gofakeit

// Interjection will generate a random word expressing emotion
func Interjection() string { return interjection(GlobalFaker) }

// Interjection will generate a random word expressing emotion
func (f *Faker) Interjection() string { return interjection(f) }

func interjection(f *Faker) string { return getRandValue(f, []string{"word", "interjection"}) }

// LoremIpsumWord will generate a random word
func LoremIpsumWord() string { return loremIpsumWord(GlobalFaker) }

// LoremIpsumWord will generate a random word
func (f *Faker) LoremIpsumWord() string { return loremIpsumWord(f) }

func loremIpsumWord(f *Faker) string { return getRandValue(f, []string{"lorem", "word"}) }

func addWordMiscLookup() {
	AddFuncLookup("interjection", Info{
		Display:     "Interjection",
		Category:    "word",
		Description: "Word expressing emotion",
		Example:     "wow",
		Output:      "string",
		Aliases:     []string{"emotional expression", "feeling word", "reaction term", "exclamation element"},
		Keywords:    []string{"interjection", "emotion", "word", "expression", "feeling", "reaction", "exclamation", "utterance", "ejaculation", "emotional"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return interjection(f), nil
		},
	})

	AddFuncLookup("loremipsumword", Info{
		Display:     "Lorem Ipsum Word",
		Category:    "word",
		Description: "Word of the Lorem Ipsum placeholder text used in design and publishing",
		Example:     "quia",
		Output:      "string",
		Aliases: []string{
			"lorem word",
			"ipsum word",
			"placeholder word",
			"latin word",
		},
		Keywords: []string{
			"lorem", "ipsum", "word", "placeholder",
			"latin", "dummy", "filler", "text",
			"typography", "mockup",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return loremIpsumWord(f), nil
		},
	})
}
