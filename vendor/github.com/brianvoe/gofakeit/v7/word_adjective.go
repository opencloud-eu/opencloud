package gofakeit

// Adjective will generate a random adjective
func Adjective() string { return adjective(GlobalFaker) }

// Adjective will generate a random adjective
func (f *Faker) Adjective() string { return adjective(f) }

func adjective(f *Faker) string {
	var adjType = map[int]string{
		0: "adjective_descriptive",
		1: "adjective_quantitative",
		2: "adjective_proper",
		3: "adjective_demonstrative",
		4: "adjective_possessive",
		5: "adjective_interrogative",
		6: "adjective_indefinite",
	}
	return getRandValue(f, []string{"word", adjType[number(f, 0, 6)]})
}

// AdjectiveDescriptive will generate a random descriptive adjective
func AdjectiveDescriptive() string { return adjectiveDescriptive(GlobalFaker) }

// AdjectiveDescriptive will generate a random descriptive adjective
func (f *Faker) AdjectiveDescriptive() string { return adjectiveDescriptive(f) }

func adjectiveDescriptive(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_descriptive"})
}

// AdjectiveQuantitative will generate a random quantitative adjective
func AdjectiveQuantitative() string { return adjectiveQuantitative(GlobalFaker) }

// AdjectiveQuantitative will generate a random quantitative adjective
func (f *Faker) AdjectiveQuantitative() string { return adjectiveQuantitative(f) }

func adjectiveQuantitative(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_quantitative"})
}

// AdjectiveProper will generate a random proper adjective
func AdjectiveProper() string { return adjectiveProper(GlobalFaker) }

// AdjectiveProper will generate a random proper adjective
func (f *Faker) AdjectiveProper() string { return adjectiveProper(f) }

func adjectiveProper(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_proper"})
}

// AdjectiveDemonstrative will generate a random demonstrative adjective
func AdjectiveDemonstrative() string { return adjectiveDemonstrative(GlobalFaker) }

// AdjectiveDemonstrative will generate a random demonstrative adjective
func (f *Faker) AdjectiveDemonstrative() string { return adjectiveDemonstrative(f) }

func adjectiveDemonstrative(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_demonstrative"})
}

// AdjectivePossessive will generate a random possessive adjective
func AdjectivePossessive() string { return adjectivePossessive(GlobalFaker) }

// AdjectivePossessive will generate a random possessive adjective
func (f *Faker) AdjectivePossessive() string { return adjectivePossessive(f) }

func adjectivePossessive(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_possessive"})
}

// AdjectiveInterrogative will generate a random interrogative adjective
func AdjectiveInterrogative() string { return adjectiveInterrogative(GlobalFaker) }

// AdjectiveInterrogative will generate a random interrogative adjective
func (f *Faker) AdjectiveInterrogative() string { return adjectiveInterrogative(f) }

func adjectiveInterrogative(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_interrogative"})
}

// AdjectiveIndefinite will generate a random indefinite adjective
func AdjectiveIndefinite() string { return adjectiveIndefinite(GlobalFaker) }

// AdjectiveIndefinite will generate a random indefinite adjective
func (f *Faker) AdjectiveIndefinite() string { return adjectiveIndefinite(f) }

func adjectiveIndefinite(f *Faker) string {
	return getRandValue(f, []string{"word", "adjective_indefinite"})
}

func addWordAdjectiveLookup() {
	AddFuncLookup("adjective", Info{
		Display:     "Adjective",
		Category:    "word",
		Description: "Word describing or modifying a noun",
		Example:     "genuine",
		Output:      "string",
		Aliases: []string{
			"descriptor term",
			"qualifying modifier",
			"attribute marker",
			"descriptive label",
			"noun qualifier",
		},
		Keywords: []string{
			"adjective", "noun", "speech", "quality", "attribute",
			"characteristic", "property", "trait", "descriptive", "modifier",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjective(f), nil
		},
	})

	AddFuncLookup("adjectivedescriptive", Info{
		Display:     "Descriptive Adjective",
		Category:    "word",
		Description: "Adjective that provides detailed characteristics about a noun",
		Example:     "brave",
		Output:      "string",
		Aliases: []string{
			"qualitative adjective",
			"detail-rich modifier",
			"characterizing term",
			"specific descriptor",
			"noun enhancer",
		},
		Keywords: []string{
			"adjective", "word", "describing", "modifying", "attribute",
			"property", "trait", "feature", "aspect", "detailed", "characteristics",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveDescriptive(f), nil
		},
	})

	AddFuncLookup("adjectivequantitative", Info{
		Display:     "Quantitative Adjective",
		Category:    "word",
		Description: "Adjective that indicates the quantity or amount of something",
		Example:     "a little",
		Output:      "string",
		Aliases: []string{
			"numeric descriptor",
			"cardinal qualifier",
			"quantifier adjective",
			"how many indicator",
			"magnitude marker",
		},
		Keywords: []string{
			"adjective", "quantitative", "word", "describing", "modifying",
			"count", "volume", "extent", "degree", "magnitude", "quantity", "amount",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveQuantitative(f), nil
		},
	})

	AddFuncLookup("adjectiveproper", Info{
		Display:     "Proper Adjective",
		Category:    "word",
		Description: "Adjective derived from a proper noun, often used to describe nationality or origin",
		Example:     "Afghan",
		Output:      "string",
		Aliases: []string{
			"nationality adjective",
			"eponym-derived",
			"proper-noun based",
			"demonym adjective",
			"origin descriptor",
		},
		Keywords: []string{
			"adjective", "noun", "word", "describing",
			"cultural", "regional", "ethnic", "linguistic", "heritage",
			"proper", "nationality",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveProper(f), nil
		},
	})

	AddFuncLookup("adjectivedemonstrative", Info{
		Display:     "Demonstrative Adjective",
		Category:    "word",
		Description: "Adjective used to point out specific things",
		Example:     "this",
		Output:      "string",
		Aliases: []string{
			"demonstrative adjective",
			"pointing adjective",
			"deictic adjective",
			"proximal distal adjective",
			"reference adjective",
		},
		Keywords: []string{
			"adjective", "demonstrative", "deictic",
			"this", "that", "these", "those",
			"proximal", "distal", "near", "far",
			"pointer", "reference", "specific", "grammar",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveDemonstrative(f), nil
		},
	})

	AddFuncLookup("adjectivepossessive", Info{
		Display:     "Possessive Adjective",
		Category:    "word",
		Description: "Adjective indicating ownership or possession",
		Example:     "my",
		Output:      "string",
		Aliases: []string{
			"ownership adjective",
			"owners descriptor",
			"possessive determiner",
			"belonging indicator",
			"proprietary modifier",
		},
		Keywords: []string{
			"adjective", "word", "grammar",
			"my", "your", "his", "her", "its", "our", "their",
			"belong", "possessive", "ownership",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectivePossessive(f), nil
		},
	})

	AddFuncLookup("adjectiveinterrogative", Info{
		Display:     "Interrogative Adjective",
		Category:    "word",
		Description: "Adjective used to ask questions",
		Example:     "what",
		Output:      "string",
		Aliases: []string{
			"interrogative adjective",
			"question word",
			"asking adjective",
			"inquiry word",
			"grammar adjective",
		},
		Keywords: []string{
			"adjective", "word", "grammar", "what", "which", "whose",
			"question", "inquiry", "interrogation", "interrogative", "ask",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveInterrogative(f), nil
		},
	})

	AddFuncLookup("adjectiveindefinite", Info{
		Display:     "Indefinite Adjective",
		Category:    "word",
		Description: "Adjective describing a non-specific noun",
		Example:     "few",
		Output:      "string",
		Aliases: []string{
			"unspecified adjective",
			"quantifier-like",
			"noncount marker",
			"broad determiner",
			"approximate amount",
		},
		Keywords: []string{
			"adjective", "noun", "word", "grammar",
			"some", "any", "many", "few", "several", "various", "certain",
			"indefinite", "non-specific",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return adjectiveIndefinite(f), nil
		},
	})

}
