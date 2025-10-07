package gofakeit

// Pronoun will generate a random pronoun
func Pronoun() string { return pronoun(GlobalFaker) }

// Pronoun will generate a random pronoun
func (f *Faker) Pronoun() string { return pronoun(f) }

func pronoun(f *Faker) string {
	var pronounType = map[int]string{
		0: "pronoun_personal",
		1: "pronoun_object",
		2: "pronoun_possessive",
		3: "pronoun_reflective",
		4: "pronoun_indefinite",
		5: "pronoun_demonstrative",
		6: "pronoun_interrogative",
		7: "pronoun_relative",
	}
	return getRandValue(f, []string{"word", pronounType[number(f, 0, 7)]})
}

// PronounPersonal will generate a random personal pronoun
func PronounPersonal() string { return pronounPersonal(GlobalFaker) }

// PronounPersonal will generate a random personal pronoun
func (f *Faker) PronounPersonal() string { return pronounPersonal(f) }

func pronounPersonal(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_personal"})
}

// PronounObject will generate a random object pronoun
func PronounObject() string { return pronounObject(GlobalFaker) }

// PronounObject will generate a random object pronoun
func (f *Faker) PronounObject() string { return pronounObject(f) }

func pronounObject(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_object"})
}

// PronounPossessive will generate a random possessive pronoun
func PronounPossessive() string { return pronounPossessive(GlobalFaker) }

// PronounPossessive will generate a random possessive pronoun
func (f *Faker) PronounPossessive() string { return pronounPossessive(f) }

func pronounPossessive(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_possessive"})
}

// PronounReflective will generate a random reflective pronoun
func PronounReflective() string { return pronounReflective(GlobalFaker) }

// PronounReflective will generate a random reflective pronoun
func (f *Faker) PronounReflective() string { return pronounReflective(f) }

func pronounReflective(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_reflective"})
}

// PronounIndefinite will generate a random indefinite pronoun
func PronounIndefinite() string { return pronounIndefinite(GlobalFaker) }

// PronounIndefinite will generate a random indefinite pronoun
func (f *Faker) PronounIndefinite() string { return pronounIndefinite(f) }

func pronounIndefinite(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_indefinite"})
}

// PronounDemonstrative will generate a random demonstrative pronoun
func PronounDemonstrative() string { return pronounDemonstrative(GlobalFaker) }

// PronounDemonstrative will generate a random demonstrative pronoun
func (f *Faker) PronounDemonstrative() string { return pronounDemonstrative(f) }

func pronounDemonstrative(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_demonstrative"})
}

// PronounInterrogative will generate a random interrogative pronoun
func PronounInterrogative() string { return pronounInterrogative(GlobalFaker) }

// PronounInterrogative will generate a random interrogative pronoun
func (f *Faker) PronounInterrogative() string { return pronounInterrogative(f) }

func pronounInterrogative(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_interrogative"})
}

// PronounRelative will generate a random relative pronoun
func PronounRelative() string { return pronounRelative(GlobalFaker) }

// PronounRelative will generate a random relative pronoun
func (f *Faker) PronounRelative() string { return pronounRelative(f) }

func pronounRelative(f *Faker) string {
	return getRandValue(f, []string{"word", "pronoun_relative"})
}

func addWordPronounLookup() {
	AddFuncLookup("pronoun", Info{
		Display:     "Pronoun",
		Category:    "word",
		Description: "Word used in place of a noun to avoid repetition",
		Example:     "me",
		Output:      "string",
		Aliases:     []string{"noun substitute", "word replacement", "grammar element", "reference word"},
		Keywords:    []string{"pronoun", "noun", "replacement", "grammar", "repetition", "substitute", "reference", "avoidance"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronoun(f), nil
		},
	})

	AddFuncLookup("pronounpersonal", Info{
		Display:     "Pronoun Personal",
		Category:    "word",
		Description: "Pronoun referring to a specific persons or things",
		Example:     "it",
		Output:      "string",
		Aliases:     []string{"personal reference", "specific entity", "individual pronoun", "grammar element"},
		Keywords:    []string{"pronoun", "personal", "specific", "person", "thing", "grammar", "referring", "individual", "entity", "identity"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounPersonal(f), nil
		},
	})

	AddFuncLookup("pronounobject", Info{
		Display:     "Pronoun Object",
		Category:    "word",
		Description: "Pronoun used as the object of a verb or preposition",
		Example:     "it",
		Output:      "string",
		Aliases:     []string{"object reference", "verb object", "preposition object", "grammar function"},
		Keywords:    []string{"pronoun", "verb", "preposition", "grammar", "used", "objective", "case", "receiver"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounObject(f), nil
		},
	})

	AddFuncLookup("pronounpossessive", Info{
		Display:     "Pronoun Possessive",
		Category:    "word",
		Description: "Pronoun indicating ownership or belonging",
		Example:     "mine",
		Output:      "string",
		Aliases:     []string{"ownership indicator", "belonging reference", "possession word", "grammar element"},
		Keywords:    []string{"pronoun", "possessive", "ownership", "belonging", "grammar", "indicating", "possession", "property"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounPossessive(f), nil
		},
	})

	AddFuncLookup("pronounreflective", Info{
		Display:     "Pronoun Reflective",
		Category:    "word",
		Description: "Pronoun referring back to the subject of the sentence",
		Example:     "myself",
		Output:      "string",
		Aliases:     []string{"self reference", "subject reflection", "backward reference", "grammar element"},
		Keywords:    []string{"pronoun", "reflective", "subject", "sentence", "grammar", "referring", "reflexive", "self", "mirror"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounReflective(f), nil
		},
	})

	AddFuncLookup("pronounindefinite", Info{
		Display:     "Pronoun Indefinite",
		Category:    "word",
		Description: "Pronoun that does not refer to a specific person or thing",
		Example:     "few",
		Output:      "string",
		Aliases:     []string{"vague reference", "general pronoun", "unspecific word", "grammar element"},
		Keywords:    []string{"pronoun", "indefinite", "specific", "person", "grammar", "refer", "vague", "general", "unspecified"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounIndefinite(f), nil
		},
	})

	AddFuncLookup("pronoundemonstrative", Info{
		Display:     "Pronoun Demonstrative",
		Category:    "word",
		Description: "Pronoun that points out specific people or things",
		Example:     "this",
		Output:      "string",
		Aliases:     []string{"pointing reference", "specific indicator", "demonstration word", "grammar element"},
		Keywords:    []string{"pronoun", "demonstrative", "specific", "people", "grammar", "points", "indicate", "reference", "pointing"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounDemonstrative(f), nil
		},
	})

	AddFuncLookup("pronouninterrogative", Info{
		Display:     "Pronoun Interrogative",
		Category:    "word",
		Description: "Pronoun used to ask questions",
		Example:     "what",
		Output:      "string",
		Aliases:     []string{"question word", "inquiry reference", "interrogation element", "grammar function"},
		Keywords:    []string{"pronoun", "interrogative", "question", "ask", "grammar", "used", "inquiry", "wh-word", "questioning"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounInterrogative(f), nil
		},
	})

	AddFuncLookup("pronounrelative", Info{
		Display:     "Pronoun Relative",
		Category:    "word",
		Description: "Pronoun that introduces a clause, referring back to a noun or pronoun",
		Example:     "as",
		Output:      "string",
		Aliases:     []string{"backward reference", "linking pronoun", "grammar element"},
		Keywords:    []string{"pronoun", "relative", "clause", "noun", "grammar", "introduces", "referring", "connector", "link"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return pronounRelative(f), nil
		},
	})
}
