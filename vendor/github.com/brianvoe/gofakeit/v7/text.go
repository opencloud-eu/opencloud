package gofakeit

import (
	"bytes"
	"errors"
	"strings"
	"unicode"
)

type paragrapOptions struct {
	paragraphCount int
	sentenceCount  int
	wordCount      int
	separator      string
}

const bytesPerWordEstimation = 6

type sentenceGenerator func(f *Faker, wordCount int) string
type wordGenerator func(f *Faker) string

// Comment will generate a random statement or remark expressing an opinion, observation, or reaction
func Comment() string { return comment(GlobalFaker) }

// Comment will generate a random statement or remark expressing an opinion, observation, or reaction
func (f *Faker) Comment() string { return comment(f) }

func comment(f *Faker) string {
	structures := [][]string{
		{"interjection", "adjective", "noun", "verb", "adverb"},
		{"noun", "verb", "preposition", "determiner", "adjective", "noun"},
		{"noun", "verb", "adverb"},
		{"adjective", "noun", "verb"},
		{"noun", "verb", "preposition", "noun"},
	}

	// Randomly select a structure
	structure := structures[number(f, 0, len(structures)-1)]

	// Build the sentence
	var commentParts []string
	for _, wordType := range structure {
		switch wordType {
		case "noun":
			commentParts = append(commentParts, noun(f))
		case "verb":
			commentParts = append(commentParts, verb(f))
		case "adjective":
			commentParts = append(commentParts, adjective(f))
		case "adverb":
			commentParts = append(commentParts, adverb(f))
		case "interjection":
			commentParts = append(commentParts, interjection(f))
		case "preposition":
			commentParts = append(commentParts, preposition(f))
		case "determiner":
			commentParts = append(commentParts, nounDeterminer(f))
		default:
			// Should never hit
			panic("Invalid word type")
		}
	}

	// Combine the words into a sentence
	sentence := strings.Join(commentParts, " ")

	// Capitalize the first letter
	sentence = title(sentence)

	// Add a period to the end of the sentence
	sentence = sentence + "."

	return sentence
}

// Phrase will return a random phrase
func Phrase() string { return phrase(GlobalFaker) }

// Phrase will return a random phrase
func (f *Faker) Phrase() string { return phrase(f) }

func phrase(f *Faker) string { return getRandValue(f, []string{"sentence", "phrase"}) }

// PhraseNoun will return a random noun phrase
func PhraseNoun() string { return phraseNoun(GlobalFaker) }

// PhraseNoun will return a random noun phrase
func (f *Faker) PhraseNoun() string { return phraseNoun(f) }

func phraseNoun(f *Faker) string {
	str := ""

	// You may also want to add an adjective to describe the noun
	if boolFunc(f) {
		str = adjectiveDescriptive(f) + " " + noun(f)
	} else {
		str = noun(f)
	}

	// Add determiner from weighted list
	prob, _ := weighted(f, []any{1, 2, 3}, []float32{2, 1.5, 1})
	if prob == 1 {
		str = getArticle(str) + " " + str
	} else if prob == 2 {
		str = "the " + str
	}

	return str
}

// PhraseVerb will return a random preposition phrase
func PhraseVerb() string { return phraseVerb(GlobalFaker) }

// PhraseVerb will return a random preposition phrase
func (f *Faker) PhraseVerb() string { return phraseVerb(f) }

func phraseVerb(f *Faker) string {
	// Put together a string builder
	sb := []string{}

	// You may have an adverb phrase
	if boolFunc(f) {
		sb = append(sb, phraseAdverb(f))
	}

	// Lets add the primary verb
	sb = append(sb, verbAction(f))

	// You may have a noun phrase
	if boolFunc(f) {
		sb = append(sb, phraseNoun(f))
	}

	// You may have an adverb phrase
	if boolFunc(f) {
		sb = append(sb, phraseAdverb(f))

		// You may also have a preposition phrase
		if boolFunc(f) {
			sb = append(sb, phrasePreposition(f))
		}

		// You may also hae an adverb phrase
		if boolFunc(f) {
			sb = append(sb, phraseAdverb(f))
		}
	}

	return strings.Join(sb, " ")
}

// PhraseAdverb will return a random adverb phrase
func PhraseAdverb() string { return phraseAdverb(GlobalFaker) }

// PhraseAdverb will return a random adverb phrase
func (f *Faker) PhraseAdverb() string { return phraseAdverb(f) }

func phraseAdverb(f *Faker) string {
	if boolFunc(f) {
		return adverbDegree(f) + " " + adverbManner(f)
	}

	return adverbManner(f)
}

// PhrasePreposition will return a random preposition phrase
func PhrasePreposition() string { return phrasePreposition(GlobalFaker) }

// PhrasePreposition will return a random preposition phrase
func (f *Faker) PhrasePreposition() string { return phrasePreposition(f) }

func phrasePreposition(f *Faker) string {
	return prepositionSimple(f) + " " + phraseNoun(f)
}

// Sentence will generate a random sentence
func Sentence(wordCount int) string { return sentence(GlobalFaker, wordCount) }

// Sentence will generate a random sentence
func (f *Faker) Sentence(wordCount int) string { return sentence(f, wordCount) }

func sentence(f *Faker, wordCount int) string {
	return sentenceGen(f, wordCount, word)
}

// Paragraph will generate a random paragraphGenerator
func Paragraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return paragraph(GlobalFaker, paragraphCount, sentenceCount, wordCount, separator)
}

// Paragraph will generate a random paragraphGenerator
func (f *Faker) Paragraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return paragraph(f, paragraphCount, sentenceCount, wordCount, separator)
}

func paragraph(f *Faker, paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return paragraphGen(f, paragrapOptions{paragraphCount, sentenceCount, wordCount, separator}, sentence)
}

func sentenceGen(f *Faker, wordCount int, word wordGenerator) string {
	if wordCount <= 0 {
		return ""
	}

	wordSeparator := ' '
	sentence := bytes.Buffer{}
	sentence.Grow(wordCount * bytesPerWordEstimation)

	for i := 0; i < wordCount; i++ {
		word := word(f)
		if i == 0 {
			runes := []rune(word)
			runes[0] = unicode.ToTitle(runes[0])
			word = string(runes)
		}
		sentence.WriteString(word)
		if i < wordCount-1 {
			sentence.WriteRune(wordSeparator)
		}
	}
	sentence.WriteRune('.')
	return sentence.String()
}

func paragraphGen(f *Faker, opts paragrapOptions, sentecer sentenceGenerator) string {
	if opts.paragraphCount <= 0 || opts.sentenceCount <= 0 || opts.wordCount <= 0 {
		return ""
	}

	//to avoid making Go 1.10 dependency, we cannot use strings.Builder
	paragraphs := bytes.Buffer{}
	//we presume the length
	paragraphs.Grow(opts.paragraphCount * opts.sentenceCount * opts.wordCount * bytesPerWordEstimation)
	wordSeparator := ' '

	for i := 0; i < opts.paragraphCount; i++ {
		for e := 0; e < opts.sentenceCount; e++ {
			paragraphs.WriteString(sentecer(f, opts.wordCount))
			if e < opts.sentenceCount-1 {
				paragraphs.WriteRune(wordSeparator)
			}
		}

		if i < opts.paragraphCount-1 {
			paragraphs.WriteString(opts.separator)
		}
	}

	return paragraphs.String()
}

// Question will return a random question
func Question() string {
	return question(GlobalFaker)
}

// Question will return a random question
func (f *Faker) Question() string {
	return question(f)
}

func question(f *Faker) string {
	return strings.Replace(hipsterSentence(f, number(f, 3, 10)), ".", "?", 1)
}

// Quote will return a random quote from a random person
func Quote() string { return quote(GlobalFaker) }

// Quote will return a random quote from a random person
func (f *Faker) Quote() string { return quote(f) }

func quote(f *Faker) string {
	return `"` + hipsterSentence(f, number(f, 3, 10)) + `" - ` + firstName(f) + " " + lastName(f)
}

// LoremIpsumSentence will generate a random sentence
func LoremIpsumSentence(wordCount int) string {
	return loremIpsumSentence(GlobalFaker, wordCount)
}

// LoremIpsumSentence will generate a random sentence
func (f *Faker) LoremIpsumSentence(wordCount int) string {
	return loremIpsumSentence(f, wordCount)
}

func loremIpsumSentence(f *Faker, wordCount int) string {
	return sentenceGen(f, wordCount, loremIpsumWord)
}

// LoremIpsumParagraph will generate a random paragraphGenerator
func LoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return loremIpsumParagraph(GlobalFaker, paragraphCount, sentenceCount, wordCount, separator)
}

// LoremIpsumParagraph will generate a random paragraphGenerator
func (f *Faker) LoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return loremIpsumParagraph(f, paragraphCount, sentenceCount, wordCount, separator)
}

func loremIpsumParagraph(f *Faker, paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return paragraphGen(f, paragrapOptions{paragraphCount, sentenceCount, wordCount, separator}, loremIpsumSentence)
}

func addTextLookup() {
	AddFuncLookup("comment", Info{
		Display:     "Comment",
		Category:    "text",
		Description: "Statement or remark expressing an opinion, observation, or reaction",
		Example:     "wow",
		Output:      "string",
		Aliases: []string{
			"verbal statement", "expressed thought", "spoken remark", "communication element", "casual note",
		},
		Keywords: []string{
			"opinion", "observation", "reaction", "response", "feedback", "critique", "interpretation", "perspective", "reflection", "discussion",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return comment(f), nil
		},
	})

	AddFuncLookup("phrase", Info{
		Display:     "Phrase",
		Category:    "text",
		Description: "A small group of words standing together",
		Example:     "time will tell",
		Output:      "string",
		Aliases:     []string{"word group", "language unit", "text element", "expression block"},
		Keywords:    []string{"phrase", "words", "group", "sentence", "text", "language", "grammar", "expression", "unit", "collection"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return phrase(f), nil
		},
	})

	AddFuncLookup("phrasenoun", Info{
		Display:     "Noun Phrase",
		Category:    "text",
		Description: "Phrase with a noun as its head, functions within sentence like a noun",
		Example:     "a tribe",
		Output:      "string",
		Aliases:     []string{"nominal phrase", "substantive element", "subject phrase", "object phrase"},
		Keywords:    []string{"phrase", "noun", "grammar", "subject", "object", "head", "sentence", "nominal", "substantive", "entity"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return phraseNoun(f), nil
		},
	})

	AddFuncLookup("phraseverb", Info{
		Display:     "Verb Phrase",
		Category:    "text",
		Description: "Phrase that Consists of a verb and its modifiers, expressing an action or state",
		Example:     "a tribe",
		Output:      "string",
		Aliases:     []string{"predicate phrase", "verbal element", "action phrase", "state phrase"},
		Keywords:    []string{"phrase", "verb", "grammar", "action", "state", "modifiers", "sentence", "predicate", "verbal", "behavior"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return phraseVerb(f), nil
		},
	})

	AddFuncLookup("phraseadverb", Info{
		Display:     "Adverb Phrase",
		Category:    "text",
		Description: "Phrase that modifies a verb, adjective, or another adverb, providing additional information.",
		Example:     "fully gladly",
		Output:      "string",
		Aliases:     []string{"adverbial phrase", "qualifier element", "modifier phrase", "description phrase"},
		Keywords:    []string{"phrase", "adverb", "grammar", "modifier", "description", "information", "adverbial", "qualifier", "modification"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return phraseAdverb(f), nil
		},
	})

	AddFuncLookup("phrasepreposition", Info{
		Display:     "Preposition Phrase",
		Category:    "text",
		Description: "Phrase starting with a preposition, showing relation between elements in a sentence.",
		Example:     "out the black thing",
		Output:      "string",
		Aliases:     []string{"prepositional phrase", "relational element", "connection phrase", "grammar bridge"},
		Keywords:    []string{"phrase", "preposition", "grammar", "relation", "connection", "sentence", "prepositional", "relational", "linking"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return phrasePreposition(f), nil
		},
	})

	AddFuncLookup("sentence", Info{
		Display:     "Sentence",
		Category:    "text",
		Description: "Set of words expressing a statement, question, exclamation, or command",
		Example:     "Interpret context record river mind.",
		Output:      "string",
		Aliases:     []string{"complete thought", "grammatical unit", "word group", "linguistic element"},
		Keywords:    []string{"sentence", "complete", "thought", "grammatical", "unit", "word", "group", "expression", "clause", "utterance"},
		Params: []Param{
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			return sentence(f, wordCount), nil
		},
	})

	AddFuncLookup("paragraph", Info{
		Display:     "Paragraph",
		Category:    "text",
		Description: "Distinct section of writing covering a single theme, composed of multiple sentences",
		Example:     "Interpret context record river mind press self should compare property outcome divide. Combine approach sustain consult discover explanation direct address church husband seek army. Begin own act welfare replace press suspect stay link place manchester specialist. Arrive price satisfy sign force application hair train provide basis right pay. Close mark teacher strengthen information attempt head touch aim iron tv take.",
		Output:      "string",
		Aliases:     []string{"text block", "writing section", "thematic unit", "content block"},
		Keywords:    []string{"paragraph", "text", "block", "writing", "section", "theme", "sentences", "composition", "distinct", "passage", "content"},
		Params: []Param{
			{Field: "paragraphcount", Display: "Paragraph Count", Type: "int", Default: "2", Description: "Number of paragraphs"},
			{Field: "sentencecount", Display: "Sentence Count", Type: "int", Default: "2", Description: "Number of sentences in a paragraph"},
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
			{Field: "paragraphseparator", Display: "Paragraph Separator", Type: "string", Default: "<br />", Description: "String value to add between paragraphs"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			paragraphCount, err := info.GetInt(m, "paragraphcount")
			if err != nil {
				return nil, err
			}
			if paragraphCount <= 0 || paragraphCount > 20 {
				return nil, errors.New("invalid paragraph count, must be greater than 0, less than 20")
			}

			sentenceCount, err := info.GetInt(m, "sentencecount")
			if err != nil {
				return nil, err
			}
			if sentenceCount <= 0 || sentenceCount > 20 {
				return nil, errors.New("invalid sentence count, must be greater than 0, less than 20")
			}

			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			paragraphSeparator, err := info.GetString(m, "paragraphseparator")
			if err != nil {
				return nil, err
			}

			return paragraph(f, paragraphCount, sentenceCount, wordCount, paragraphSeparator), nil
		},
	})

	AddFuncLookup("question", Info{
		Display:     "Question",
		Category:    "text",
		Description: "Statement formulated to inquire or seek clarification",
		Example:     "Roof chia echo?",
		Output:      "string",
		Aliases: []string{
			"interrogative sentence",
			"information request",
			"asking phrase",
			"query prompt",
			"clarifying ask",
		},
		Keywords: []string{
			"question", "inquiry", "clarification", "interrogative",
			"ask", "who", "what", "when", "where", "why", "how", "mark",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return question(f), nil
		},
	})

	AddFuncLookup("quote", Info{
		Display:     "Quote",
		Category:    "text",
		Description: "Direct repetition of someone else's words",
		Example:     `"Roof chia echo." - Lura Lockman`,
		Output:      "string",
		Aliases: []string{
			"direct speech",
			"verbatim line",
			"cited passage",
			"attributed text",
			"pulled excerpt",
		},
		Keywords: []string{
			"quote", "quotation", "citation", "reference", "excerpt",
			"epigraph", "saying", "maxim", "attribution", "blockquote",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return quote(f), nil
		},
	})

	AddFuncLookup("loremipsumsentence", Info{
		Display:     "Lorem Ipsum Sentence",
		Category:    "text",
		Description: "Sentence of the Lorem Ipsum placeholder text used in design and publishing",
		Example:     "Quia quae repellat consequatur quidem.",
		Output:      "string",
		Aliases: []string{
			"lorem sentence",
			"ipsum sentence",
			"placeholder sentence",
			"latin sentence",
		},
		Keywords: []string{
			"lorem", "ipsum", "sentence", "placeholder",
			"latin", "dummy", "filler", "text",
			"typography", "mockup",
		},
		Params: []Param{
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			return loremIpsumSentence(f, wordCount), nil
		},
	})

	AddFuncLookup("loremipsumparagraph", Info{
		Display:     "Lorem Ipsum Paragraph",
		Category:    "text",
		Description: "Paragraph of the Lorem Ipsum placeholder text used in design and publishing",
		Example: `Quia quae repellat consequatur quidem nisi quo qui voluptatum accusantium quisquam amet. Quas et ut non dolorem ipsam aut enim assumenda mollitia harum ut. Dicta similique veniam nulla voluptas at excepturi non ad maxime at non. Eaque hic repellat praesentium voluptatem qui consequuntur dolor iusto autem velit aut. Fugit tempore exercitationem harum consequatur voluptatum modi minima aut eaque et et.

Aut ea voluptatem dignissimos expedita odit tempore quod aut beatae ipsam iste. Minus voluptatibus dolorem maiores eius sed nihil vel enim odio voluptatem accusamus. Natus quibusdam temporibus tenetur cumque sint necessitatibus dolorem ex ducimus iusto ex. Voluptatem neque dicta explicabo officiis et ducimus sit ut ut praesentium pariatur. Illum molestias nisi at dolore ut voluptatem accusantium et fugiat et ut.

Explicabo incidunt reprehenderit non quia dignissimos recusandae vitae soluta quia et quia. Aut veniam voluptas consequatur placeat sapiente non eveniet voluptatibus magni velit eum. Nobis vel repellendus sed est qui autem laudantium quidem quam ullam consequatur. Aut iusto ut commodi similique quae voluptatem atque qui fugiat eum aut. Quis distinctio consequatur voluptatem vel aliquid aut laborum facere officiis iure tempora.`,
		Output: "string",
		Aliases: []string{
			"lorem paragraph",
			"ipsum paragraph",
			"placeholder paragraph",
			"latin paragraph",
		},
		Keywords: []string{
			"lorem", "ipsum", "paragraph", "placeholder",
			"latin", "dummy", "filler", "text",
			"typography", "mockup",
		},
		Params: []Param{
			{Field: "paragraphcount", Display: "Paragraph Count", Type: "int", Default: "2", Description: "Number of paragraphs"},
			{Field: "sentencecount", Display: "Sentence Count", Type: "int", Default: "2", Description: "Number of sentences in a paragraph"},
			{Field: "wordcount", Display: "Word Count", Type: "int", Default: "5", Description: "Number of words in a sentence"},
			{Field: "paragraphseparator", Display: "Paragraph Separator", Type: "string", Default: "<br />", Description: "String value to add between paragraphs"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			paragraphCount, err := info.GetInt(m, "paragraphcount")
			if err != nil {
				return nil, err
			}
			if paragraphCount <= 0 || paragraphCount > 20 {
				return nil, errors.New("invalid paragraph count, must be greater than 0, less than 20")
			}

			sentenceCount, err := info.GetInt(m, "sentencecount")
			if err != nil {
				return nil, err
			}
			if sentenceCount <= 0 || sentenceCount > 20 {
				return nil, errors.New("invalid sentence count, must be greater than 0, less than 20")
			}

			wordCount, err := info.GetInt(m, "wordcount")
			if err != nil {
				return nil, err
			}
			if wordCount <= 0 || wordCount > 50 {
				return nil, errors.New("invalid word count, must be greater than 0, less than 50")
			}

			paragraphSeparator, err := info.GetString(m, "paragraphseparator")
			if err != nil {
				return nil, err
			}

			return loremIpsumParagraph(f, paragraphCount, sentenceCount, wordCount, paragraphSeparator), nil
		},
	})
}
