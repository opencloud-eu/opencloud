package gofakeit

// Verb will generate a random verb
func Verb() string { return verb(GlobalFaker) }

// Verb will generate a random verb
func (f *Faker) Verb() string { return verb(f) }

func verb(f *Faker) string {
	var verbType = map[int]string{
		0: "verb_action",
		1: "verb_linking",
		2: "verb_helping",
	}
	return getRandValue(f, []string{"word", verbType[number(f, 0, 2)]})
}

// VerbAction will generate a random action verb
func VerbAction() string { return verbAction(GlobalFaker) }

// VerbAction will generate a random action verb
func (f *Faker) VerbAction() string { return verbAction(f) }

func verbAction(f *Faker) string { return getRandValue(f, []string{"word", "verb_action"}) }

// VerbTransitive will generate a random transitive verb
func VerbTransitive() string { return verbTransitive(GlobalFaker) }

// VerbTransitive will generate a random transitive verb
func (f *Faker) VerbTransitive() string { return verbTransitive(f) }

func verbTransitive(f *Faker) string { return getRandValue(f, []string{"word", "verb_transitive"}) }

// VerbIntransitive will generate a random intransitive verb
func VerbIntransitive() string { return verbIntransitive(GlobalFaker) }

// VerbIntransitive will generate a random intransitive verb
func (f *Faker) VerbIntransitive() string { return verbIntransitive(f) }

func verbIntransitive(f *Faker) string {
	return getRandValue(f, []string{"word", "verb_intransitive"})
}

// VerbLinking will generate a random linking verb
func VerbLinking() string { return verbLinking(GlobalFaker) }

// VerbLinking will generate a random linking verb
func (f *Faker) VerbLinking() string { return verbLinking(f) }

func verbLinking(f *Faker) string { return getRandValue(f, []string{"word", "verb_linking"}) }

// VerbHelping will generate a random helping verb
func VerbHelping() string { return verbHelping(GlobalFaker) }

// VerbHelping will generate a random helping verb
func (f *Faker) VerbHelping() string { return verbHelping(f) }

func verbHelping(f *Faker) string { return getRandValue(f, []string{"word", "verb_helping"}) }

func addWordVerbLookup() {
	AddFuncLookup("verb", Info{
		Display:     "Verb",
		Category:    "word",
		Description: "Word expressing an action, event or state",
		Example:     "release",
		Output:      "string",
		Aliases: []string{
			"action word", "doing word", "predicate word", "verb form", "process word",
		},
		Keywords: []string{
			"movement", "change", "existence", "process", "condition", "happening", "expression", "statement", "activity", "function",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verb(f), nil
		},
	})

	AddFuncLookup("verbaction", Info{
		Display:     "Action Verb",
		Category:    "word",
		Description: "Verb indicating a physical or mental action",
		Example:     "close",
		Output:      "string",
		Aliases: []string{
			"movement word", "doing action", "behavior word", "mental action", "physical action",
		},
		Keywords: []string{
			"activity", "task", "operation", "motion", "effort", "performance", "gesture", "response", "execution", "behavior",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verbAction(f), nil
		},
	})

	AddFuncLookup("verbtransitive", Info{
		Display:     "Transitive Verb",
		Category:    "word",
		Description: "Verb that requires a direct object to complete its meaning",
		Example:     "follow",
		Output:      "string",
		Aliases: []string{
			"object verb", "requires object", "dependent verb", "object-linked", "receiver word",
		},
		Keywords: []string{
			"direct", "receiver", "transfer", "target", "completion", "relation", "dependent", "object-based", "action-transfer", "link",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verbTransitive(f), nil
		},
	})

	AddFuncLookup("verbintransitive", Info{
		Display:     "Intransitive Verb",
		Category:    "word",
		Description: "Verb that does not require a direct object to complete its meaning",
		Example:     "laugh",
		Output:      "string",
		Aliases: []string{
			"standalone verb", "independent word", "no object verb", "complete action", "self-contained verb",
		},
		Keywords: []string{
			"autonomous", "independent", "non-transfer", "self-complete", "expression", "state", "behavior", "occur", "perform", "action-only",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verbIntransitive(f), nil
		},
	})

	AddFuncLookup("verblinking", Info{
		Display:     "Linking Verb",
		Category:    "word",
		Description: "Verb that connects the subject of a sentence to a subject complement",
		Example:     "was",
		Output:      "string",
		Aliases: []string{
			"connecting verb", "copular verb", "bridge word", "link word", "equating verb",
		},
		Keywords: []string{
			"relation", "connection", "equivalence", "identification", "state", "being", "subject-link", "copula", "connector", "description",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verbLinking(f), nil
		},
	})

	AddFuncLookup("verbhelping", Info{
		Display:     "Helping Verb",
		Category:    "word",
		Description: "Auxiliary verb that helps the main verb complete the sentence",
		Example:     "be",
		Output:      "string",
		Aliases: []string{
			"auxiliary verb", "supporting verb", "assisting word", "helper verb", "modal verb",
		},
		Keywords: []string{
			"tense", "mood", "voice", "aspect", "support", "structure",
			"compound", "formation", "assistance",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return verbHelping(f), nil
		},
	})

}
