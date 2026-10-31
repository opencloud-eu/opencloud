package gofakeit

// Language will return a random language
func Language() string { return language(GlobalFaker) }

// Language will return a random language
func (f *Faker) Language() string { return language(f) }

func language(f *Faker) string { return getRandValue(f, []string{"language", "long"}) }

// LanguageAbbreviation will return a random language abbreviation
func LanguageAbbreviation() string { return languageAbbreviation(GlobalFaker) }

// LanguageAbbreviation will return a random language abbreviation
func (f *Faker) LanguageAbbreviation() string { return languageAbbreviation(f) }

func languageAbbreviation(f *Faker) string { return getRandValue(f, []string{"language", "short"}) }

// LanguageBCP will return a random language BCP (Best Current Practices)
func LanguageBCP() string { return languageBCP(GlobalFaker) }

// LanguageBCP will return a random language BCP (Best Current Practices)
func (f *Faker) LanguageBCP() string { return languageBCP(f) }

func languageBCP(f *Faker) string { return getRandValue(f, []string{"language", "bcp"}) }

// ProgrammingLanguage will return a random programming language
func ProgrammingLanguage() string { return programmingLanguage(GlobalFaker) }

// ProgrammingLanguage will return a random programming language
func (f *Faker) ProgrammingLanguage() string { return programmingLanguage(f) }

func programmingLanguage(f *Faker) string {
	return getRandValue(f, []string{"language", "programming"})
}

func addLanguagesLookup() {
	AddFuncLookup("language", Info{
		Display:     "Language",
		Category:    "language",
		Description: "System of communication using symbols, words, and grammar to convey meaning between individuals",
		Example:     "Kazakh",
		Output:      "string",
		Aliases:     []string{"spoken tongue", "dialect name", "native language", "speech form", "linguistic system"},
		Keywords:    []string{"language", "communication", "symbols", "words", "grammar", "meaning", "system", "convey", "individuals"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return language(f), nil
		},
	})

	AddFuncLookup("languageabbreviation", Info{
		Display:     "Language Abbreviation",
		Category:    "language",
		Description: "Shortened form of a language's name",
		Example:     "kk",
		Output:      "string",
		Aliases:     []string{"language code", "iso code", "locale code", "short form", "abbreviated tag"},
		Keywords:    []string{"abbreviation", "identifier", "shortened", "representation", "two-letter", "three-letter", "standard", "locale"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return languageAbbreviation(f), nil
		},
	})

	AddFuncLookup("languagebcp", Info{
		Display:     "Language BCP",
		Category:    "language",
		Description: "Set of guidelines and standards for identifying and representing languages in computing and internet protocols",
		Example:     "en-US",
		Output:      "string",
		Aliases:     []string{"bcp47 tag", "language tag", "locale identifier", "regional code", "protocol language code"},
		Keywords:    []string{"guidelines", "standards", "rfc", "internet", "protocols", "representation", "locale", "region", "country"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return languageBCP(f), nil
		},
	})

	AddFuncLookup("programminglanguage", Info{
		Display:     "Programming Language",
		Category:    "language",
		Description: "Formal system of instructions used to create software and perform computational tasks",
		Example:     "Go",
		Output:      "string",
		Aliases:     []string{"coding language", "scripting language", "software language", "development language", "computer language"},
		Keywords:    []string{"programming", "instructions", "formal", "system", "tasks", "development", "compilation", "execution"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return programmingLanguage(f), nil
		},
	})
}
