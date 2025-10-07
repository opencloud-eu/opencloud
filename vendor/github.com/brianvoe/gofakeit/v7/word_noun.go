package gofakeit

// Noun will generate a random noun
func Noun() string { return noun(GlobalFaker) }

// Noun will generate a random noun
func (f *Faker) Noun() string { return noun(f) }

func noun(f *Faker) string {
	var nounType = map[int]string{
		0: "noun_common",
		1: "noun_concrete",
		2: "noun_abstract",
		3: "noun_collective_people",
		4: "noun_collective_animal",
		5: "noun_collective_thing",
		6: "noun_countable",
		7: "noun_uncountable",
	}
	return getRandValue(f, []string{"word", nounType[number(f, 0, 7)]})
}

// NounCommon will generate a random common noun
func NounCommon() string { return nounCommon(GlobalFaker) }

// NounCommon will generate a random common noun
func (f *Faker) NounCommon() string { return nounCommon(f) }

func nounCommon(f *Faker) string { return getRandValue(f, []string{"word", "noun_common"}) }

// NounConcrete will generate a random concrete noun
func NounConcrete() string { return nounConcrete(GlobalFaker) }

// NounConcrete will generate a random concrete noun
func (f *Faker) NounConcrete() string { return nounConcrete(f) }

func nounConcrete(f *Faker) string { return getRandValue(f, []string{"word", "noun_concrete"}) }

// NounAbstract will generate a random abstract noun
func NounAbstract() string { return nounAbstract(GlobalFaker) }

// NounAbstract will generate a random abstract noun
func (f *Faker) NounAbstract() string { return nounAbstract(f) }

func nounAbstract(f *Faker) string { return getRandValue(f, []string{"word", "noun_abstract"}) }

// NounCollectivePeople will generate a random collective noun person
func NounCollectivePeople() string { return nounCollectivePeople(GlobalFaker) }

// NounCollectivePeople will generate a random collective noun person
func (f *Faker) NounCollectivePeople() string { return nounCollectivePeople(f) }

func nounCollectivePeople(f *Faker) string {
	return getRandValue(f, []string{"word", "noun_collective_people"})
}

// NounCollectiveAnimal will generate a random collective noun animal
func NounCollectiveAnimal() string { return nounCollectiveAnimal(GlobalFaker) }

// NounCollectiveAnimal will generate a random collective noun animal
func (f *Faker) NounCollectiveAnimal() string { return nounCollectiveAnimal(f) }

func nounCollectiveAnimal(f *Faker) string {
	return getRandValue(f, []string{"word", "noun_collective_animal"})
}

// NounCollectiveThing will generate a random collective noun thing
func NounCollectiveThing() string { return nounCollectiveThing(GlobalFaker) }

// NounCollectiveThing will generate a random collective noun thing
func (f *Faker) NounCollectiveThing() string { return nounCollectiveThing(f) }

func nounCollectiveThing(f *Faker) string {
	return getRandValue(f, []string{"word", "noun_collective_thing"})
}

// NounCountable will generate a random countable noun
func NounCountable() string { return nounCountable(GlobalFaker) }

// NounCountable will generate a random countable noun
func (f *Faker) NounCountable() string { return nounCountable(f) }

func nounCountable(f *Faker) string { return getRandValue(f, []string{"word", "noun_countable"}) }

// NounUncountable will generate a random uncountable noun
func NounUncountable() string { return nounUncountable(GlobalFaker) }

// NounUncountable will generate a random uncountable noun
func (f *Faker) NounUncountable() string { return nounUncountable(f) }

func nounUncountable(f *Faker) string {
	return getRandValue(f, []string{"word", "noun_uncountable"})
}

// NounProper will generate a random proper noun
func NounProper() string { return nounProper(GlobalFaker) }

// NounProper will generate a random proper noun
func (f *Faker) NounProper() string { return nounProper(f) }

func nounProper(f *Faker) string {
	switch randInt := randIntRange(f, 1, 3); randInt {
	case 1:
		return getRandValue(f, []string{"celebrity", "actor"})
	case 2:
		genStr, _ := generate(f, getRandValue(f, []string{"address", "city"}))
		return genStr
	}

	return getRandValue(f, []string{"person", "first"})
}

// NounDeterminer will generate a random noun determiner
func NounDeterminer() string { return nounDeterminer(GlobalFaker) }

// NounDeterminer will generate a random noun determiner
func (f *Faker) NounDeterminer() string { return nounDeterminer(f) }

func nounDeterminer(f *Faker) string { return getRandValue(f, []string{"word", "noun_determiner"}) }

func addWordNounLookup() {
	AddFuncLookup("noun", Info{
		Display:     "Noun",
		Category:    "word",
		Description: "Person, place, thing, or idea, named or referred to in a sentence",
		Example:     "aunt",
		Output:      "string",
		Aliases: []string{
			"random noun", "grammar noun", "word type", "part speech", "naming word", "lexical noun", "nominal word",
		},
		Keywords: []string{
			"noun", "person", "place", "idea", "sentence", "grammar", "named", "referred", "subject", "object", "entity", "concept", "term", "substantive",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return noun(f), nil
		},
	})

	AddFuncLookup("nouncommon", Info{
		Display:     "Noun Common",
		Category:    "word",
		Description: "General name for people, places, or things, not specific or unique",
		Example:     "part",
		Output:      "string",
		Aliases: []string{
			"common noun", "general noun", "generic name", "basic noun", "ordinary noun", "regular noun", "everyday noun",
		},
		Keywords: []string{
			"common", "general", "name", "people", "places", "generic", "basic", "ordinary", "standard", "typical", "regular", "normal",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounCommon(f), nil
		},
	})

	AddFuncLookup("nounconcrete", Info{
		Display:     "Noun Concrete",
		Category:    "word",
		Description: "Names for physical entities experienced through senses like sight, touch, smell, or taste",
		Example:     "snowman",
		Output:      "string",
		Aliases: []string{
			"concrete noun", "physical noun", "tangible noun", "material noun", "sensory noun", "real noun", "perceptible noun",
		},
		Keywords: []string{
			"concrete", "physical", "entities", "senses", "sight", "touch", "smell", "taste", "tangible", "material", "solid", "real", "visible", "touchable", "observable",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounConcrete(f), nil
		},
	})

	AddFuncLookup("nounabstract", Info{
		Display:     "Noun Abstract",
		Category:    "word",
		Description: "Ideas, qualities, or states that cannot be perceived with the five senses",
		Example:     "confusion",
		Output:      "string",
		Aliases: []string{
			"abstract noun", "concept noun", "idea noun", "intangible noun", "mental noun", "notional noun", "theoretical noun",
		},
		Keywords: []string{
			"abstract", "ideas", "qualities", "states", "senses", "concept", "intangible", "mental", "theoretical", "emotional", "spiritual", "intellectual", "philosophical", "metaphysical",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounAbstract(f), nil
		},
	})

	AddFuncLookup("nouncollectivepeople", Info{
		Display:     "Noun Collective People",
		Category:    "word",
		Description: "Group of people or things regarded as a unit",
		Example:     "body",
		Output:      "string",
		Aliases: []string{
			"collective noun", "group noun", "people group", "crowd noun", "assembly noun", "community noun", "societal noun",
		},
		Keywords: []string{
			"collective", "people", "group", "unit", "regarded", "crowd", "assembly", "gathering", "team", "committee", "audience", "class", "family", "society",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounCollectivePeople(f), nil
		},
	})

	AddFuncLookup("nouncollectiveanimal", Info{
		Display:     "Noun Collective Animal",
		Category:    "word",
		Description: "Group of animals, like a 'pack' of wolves or a 'flock' of birds",
		Example:     "party",
		Output:      "string",
		Aliases: []string{
			"animal collective", "pack noun", "flock noun", "herd noun", "swarm noun", "colony noun", "pride noun",
		},
		Keywords: []string{
			"collective", "animal", "group", "pack", "flock", "animals", "herd", "swarm", "pride", "school", "colony", "pod", "gaggle", "murder", "exaltation",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounCollectiveAnimal(f), nil
		},
	})

	AddFuncLookup("nouncollectivething", Info{
		Display:     "Noun Collective Thing",
		Category:    "word",
		Description: "Group of objects or items, such as a 'bundle' of sticks or a 'cluster' of grapes",
		Example:     "hand",
		Output:      "string",
		Aliases: []string{
			"object collective", "bundle noun", "cluster noun", "collection noun", "set noun", "batch noun", "pile noun",
		},
		Keywords: []string{
			"collective", "thing", "group", "objects", "items", "bundle", "cluster", "collection", "set", "batch", "stack", "pile", "heap", "bunch", "array", "assortment",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounCollectiveThing(f), nil
		},
	})

	AddFuncLookup("nouncountable", Info{
		Display:     "Noun Countable",
		Category:    "word",
		Description: "Items that can be counted individually",
		Example:     "neck",
		Output:      "string",
		Aliases: []string{
			"countable noun", "count noun", "discrete item", "enumerable noun", "plural noun", "numerical noun", "measurable noun",
		},
		Keywords: []string{
			"countable", "items", "counted", "individually", "discrete", "enumerable", "plural", "many", "few", "number", "objects", "things", "units", "pieces",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounCountable(f), nil
		},
	})

	AddFuncLookup("noununcountable", Info{
		Display:     "Noun Uncountable",
		Category:    "word",
		Description: "Items that can't be counted individually",
		Example:     "seafood",
		Output:      "string",
		Aliases: []string{
			"uncountable noun", "mass noun", "non-count noun", "bulk noun", "substance noun", "continuous noun", "material noun",
		},
		Keywords: []string{
			"uncountable", "items", "counted", "individually", "mass", "bulk", "substance", "material", "liquid", "powder", "grain", "continuous", "indivisible", "measurement",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounUncountable(f), nil
		},
	})

	AddFuncLookup("nounproper", Info{
		Display:     "Noun Proper",
		Category:    "word",
		Description: "Specific name for a particular person, place, or organization",
		Example:     "John",
		Output:      "string",
		Aliases: []string{
			"proper noun", "specific name", "person name", "place name", "organization name", "capitalized noun", "unique name",
		},
		Keywords: []string{
			"proper", "specific", "name", "person", "place", "organization", "capitalized", "title", "brand", "company", "city", "country", "individual", "entity", "designation",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounProper(f), nil
		},
	})

	AddFuncLookup("noundeterminer", Info{
		Display:     "Noun Determiner",
		Category:    "word",
		Description: "Word that introduces a noun and identifies it as a noun",
		Example:     "your",
		Output:      "string",
		Aliases: []string{
			"determiner word", "article word", "noun introducer", "specifier word", "modifier word", "defining word", "introductory word",
		},
		Keywords: []string{
			"determiner", "word", "introduces", "identifies", "article", "specifier", "modifier",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return nounDeterminer(f), nil
		},
	})

}
