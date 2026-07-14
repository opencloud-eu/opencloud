package gofakeit

// Emoji will return a random fun emoji
func Emoji() string { return emoji(GlobalFaker) }

// Emoji will return a random fun emoji
func (f *Faker) Emoji() string { return emoji(f) }

func emoji(f *Faker) string { return getRandValue(f, []string{"emoji", "emoji"}) }

// EmojiDescription will return a random fun emoji description
func EmojiDescription() string { return emojiDescription(GlobalFaker) }

// EmojiDescription will return a random fun emoji description
func (f *Faker) EmojiDescription() string { return emojiDescription(f) }

func emojiDescription(f *Faker) string { return getRandValue(f, []string{"emoji", "description"}) }

// EmojiCategory will return a random fun emoji category
func EmojiCategory() string { return emojiCategory(GlobalFaker) }

// EmojiCategory will return a random fun emoji category
func (f *Faker) EmojiCategory() string { return emojiCategory(f) }

func emojiCategory(f *Faker) string { return getRandValue(f, []string{"emoji", "category"}) }

// EmojiAlias will return a random fun emoji alias
func EmojiAlias() string { return emojiAlias(GlobalFaker) }

// EmojiAlias will return a random fun emoji alias
func (f *Faker) EmojiAlias() string { return emojiAlias(f) }

func emojiAlias(f *Faker) string { return getRandValue(f, []string{"emoji", "alias"}) }

// EmojiTag will return a random fun emoji tag
func EmojiTag() string { return emojiTag(GlobalFaker) }

// EmojiTag will return a random fun emoji tag
func (f *Faker) EmojiTag() string { return emojiTag(f) }

func emojiTag(f *Faker) string { return getRandValue(f, []string{"emoji", "tag"}) }

func addEmojiLookup() {
	AddFuncLookup("emoji", Info{
		Display:     "Emoji",
		Category:    "emoji",
		Description: "Digital symbol expressing feelings or ideas in text messages and online chats",
		Example:     "🤣",
		Output:      "string",
		Aliases: []string{
			"emoticon symbol",
			"chat icon",
			"unicode pictograph",
			"emotional glyph",
			"digital expression",
		},
		Keywords: []string{
			"emoji", "symbol", "text", "message", "online", "chats", "ideas", "feelings", "digital", "reaction",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return emoji(f), nil
		},
	})

	AddFuncLookup("emojidescription", Info{
		Display:     "Emoji Description",
		Category:    "emoji",
		Description: "Brief explanation of the meaning or emotion conveyed by an emoji",
		Example:     "face vomiting",
		Output:      "string",
		Aliases: []string{
			"emoji meaning",
			"emoji definition",
			"emoji explanation",
			"emoji summary",
			"emoji interpretation",
		},
		Keywords: []string{
			"emoji", "brief", "explanation", "meaning", "emotion", "conveyed", "context", "description", "usage",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return emojiDescription(f), nil
		},
	})

	AddFuncLookup("emojicategory", Info{
		Display:     "Emoji Category",
		Category:    "emoji",
		Description: "Group or classification of emojis based on their common theme or use, like 'smileys' or 'animals'",
		Example:     "Smileys & Emotion",
		Output:      "string",
		Aliases: []string{
			"emoji group",
			"emoji theme",
			"emoji section",
			"emoji classification",
			"emoji grouping",
		},
		Keywords: []string{
			"emoji", "smileys", "emotion", "animals", "theme", "classification", "set", "category", "collection",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return emojiCategory(f), nil
		},
	})

	AddFuncLookup("emojialias", Info{
		Display:     "Emoji Alias",
		Category:    "emoji",
		Description: "Alternative name or keyword used to represent a specific emoji in text or code",
		Example:     "smile",
		Output:      "string",
		Aliases: []string{
			"emoji nickname",
			"emoji shorthand",
			"emoji label",
			"emoji alt text",
			"emoji identifier",
		},
		Keywords: []string{
			"emoji", "alias", "smile", "code", "specific", "represent", "alternative", "keyword", "mapping",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return emojiAlias(f), nil
		},
	})

	AddFuncLookup("emojitag", Info{
		Display:     "Emoji Tag",
		Category:    "emoji",
		Description: "Label or keyword associated with an emoji to categorize or search for it easily",
		Example:     "happy",
		Output:      "string",
		Aliases: []string{
			"emoji keyword",
			"emoji marker",
			"emoji label",
			"emoji hashtag",
			"emoji reference",
		},
		Keywords: []string{
			"emoji", "tag", "happy", "associated", "categorize", "search", "label", "index", "metadata",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return emojiTag(f), nil
		},
	})

}
