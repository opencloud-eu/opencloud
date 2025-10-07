package gofakeit

// Preposition will generate a random preposition
func Preposition() string { return preposition(GlobalFaker) }

// Preposition will generate a random preposition
func (f *Faker) Preposition() string { return preposition(f) }

func preposition(f *Faker) string {
	var prepType = map[int]string{
		0: "preposition_simple",
		1: "preposition_double",
		2: "preposition_compound",
	}
	return getRandValue(f, []string{"word", prepType[number(f, 0, 2)]})
}

// PrepositionSimple will generate a random simple preposition
func PrepositionSimple() string { return prepositionSimple(GlobalFaker) }

// PrepositionSimple will generate a random simple preposition
func (f *Faker) PrepositionSimple() string { return prepositionSimple(f) }

func prepositionSimple(f *Faker) string {
	return getRandValue(f, []string{"word", "preposition_simple"})
}

// PrepositionDouble will generate a random double preposition
func PrepositionDouble() string { return prepositionDouble(GlobalFaker) }

// PrepositionDouble will generate a random double preposition
func (f *Faker) PrepositionDouble() string { return prepositionDouble(f) }

func prepositionDouble(f *Faker) string {
	return getRandValue(f, []string{"word", "preposition_double"})
}

// PrepositionCompound will generate a random compound preposition
func PrepositionCompound() string { return prepositionCompound(GlobalFaker) }

// PrepositionCompound will generate a random compound preposition
func (f *Faker) PrepositionCompound() string { return prepositionCompound(f) }

func prepositionCompound(f *Faker) string {
	return getRandValue(f, []string{"word", "preposition_compound"})
}

func addWordPrepositionLookup() {
	AddFuncLookup("preposition", Info{
		Display:     "Preposition",
		Category:    "word",
		Description: "Words used to express the relationship of a noun or pronoun to other words in a sentence",
		Example:     "other than",
		Output:      "string",
		Aliases:     []string{"relationship connector", "grammar link", "sentence bridge", "word connector"},
		Keywords:    []string{"preposition", "relationship", "noun", "pronoun", "sentence", "grammar", "express", "connector", "link", "relational"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return preposition(f), nil
		},
	})

	AddFuncLookup("prepositionsimple", Info{
		Display:     "Preposition Simple",
		Category:    "word",
		Description: "Single-word preposition showing relationships between 2 parts of a sentence",
		Example:     "out",
		Output:      "string",
		Aliases:     []string{"basic connector", "fundamental link", "single element", "grammar bridge"},
		Keywords:    []string{"preposition", "simple", "single-word", "relationships", "parts", "sentence", "grammar", "showing", "basic", "fundamental", "elementary"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return prepositionSimple(f), nil
		},
	})

	AddFuncLookup("prepositiondouble", Info{
		Display:     "Preposition Double",
		Category:    "word",
		Description: "Two-word combination preposition, indicating a complex relation",
		Example:     "before",
		Output:      "string",
		Aliases:     []string{"two-word connector", "complex relation", "combination element", "grammar bridge"},
		Keywords:    []string{"preposition", "double", "two-word", "combination", "complex", "relation", "grammar", "indicating", "compound", "multi-word", "paired"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return prepositionDouble(f), nil
		},
	})

	AddFuncLookup("prepositioncompound", Info{
		Display:     "Preposition Compound",
		Category:    "word",
		Description: "Preposition that can be formed by combining two or more prepositions",
		Example:     "according to",
		Output:      "string",
		Aliases:     []string{"multi-part connector", "complex combination", "formed element", "grammar bridge"},
		Keywords:    []string{"preposition", "compound", "combining", "two", "more", "prepositions", "grammar", "formed", "complex", "multi-part", "constructed"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return prepositionCompound(f), nil
		},
	})
}
