package gofakeit

// Letter will generate a single random lower case ASCII letter
func Letter() string { return letter(GlobalFaker) }

// Letter will generate a single random lower case ASCII letter
func (f *Faker) Letter() string { return letter(f) }

func letter(f *Faker) string { return string(randLetter(f)) }

// LetterN will generate a random ASCII string with length N. Note that this function returns a string with a length of 1 when 0 is passed.
func LetterN(n uint) string { return letterN(GlobalFaker, n) }

// LetterN will generate a random ASCII string with length N. Note that this function returns a string with a length of 1 when 0 is passed.
func (f *Faker) LetterN(n uint) string { return letterN(f, n) }

func letterN(f *Faker, n uint) string {
	// Make sure we dont use 0
	if n == 0 {
		n = 1
	}
	out := make([]rune, n)
	for i := 0; i < int(n); i++ {
		out[i] = randLetter(f)
	}
	return string(out)
}

// Vowel will generate a single random lower case vowel
func Vowel() string { return vowel(GlobalFaker) }

// Vowel will generate a single random lower case vowel
func (f *Faker) Vowel() string { return vowel(f) }

func vowel(f *Faker) string { return string(randCharacter(f, vowels)) }

// Digit will generate a single ASCII digit
func Digit() string { return digit(GlobalFaker) }

// Digit will generate a single ASCII digit
func (f *Faker) Digit() string { return digit(f) }

func digit(f *Faker) string { return string(randDigit(f)) }

// DigitN will generate a random string of length N consists of ASCII digits. Note that the string generated can start with 0 and this function returns a string with a length of 1 when 0 is passed.
func DigitN(n uint) string { return digitN(GlobalFaker, n) }

// DigitN will generate a random string of length N consists of ASCII digits. Note that the string generated can start with 0 and this function returns a string with a length of 1 when 0 is passed.
func (f *Faker) DigitN(n uint) string { return digitN(f, n) }

func digitN(f *Faker, n uint) string {
	// Make sure we dont use 0
	if n == 0 {
		n = 1
	}
	out := make([]rune, n)
	for i := 0; i < int(n); i++ {
		out[i] = randDigit(f)
	}
	return string(out)
}

// Numerify will replace # with random numerical values
func Numerify(str string) string { return numerify(GlobalFaker, str) }

// Numerify will replace # with random numerical values
func (f *Faker) Numerify(str string) string { return numerify(f, str) }

func numerify(f *Faker, str string) string { return replaceWithNumbers(f, str) }

// Lexify will replace ? with random generated letters
func Lexify(str string) string { return lexify(GlobalFaker, str) }

// Lexify will replace ? with random generated letters
func (f *Faker) Lexify(str string) string { return lexify(f, str) }

func lexify(f *Faker, str string) string { return replaceWithLetters(f, str) }

// ShuffleStrings will randomize a slice of strings
func ShuffleStrings(a []string) { shuffleStrings(GlobalFaker, a) }

// ShuffleStrings will randomize a slice of strings
func (f *Faker) ShuffleStrings(a []string) { shuffleStrings(f, a) }

func shuffleStrings(f *Faker, a []string) {
	swap := func(i, j int) {
		a[i], a[j] = a[j], a[i]
	}
	//to avoid upgrading to 1.10 I copied the algorithm
	n := len(a)
	if n <= 1 {
		return
	}

	//if size is > int32 probably it will never finish, or ran out of entropy
	i := n - 1
	for ; i > 0; i-- {
		j := int(int32NFunc(f, int32(i+1)))
		swap(i, j)
	}
}

// RandomString will take in a slice of string and return a randomly selected value
func RandomString(a []string) string { return randomString(GlobalFaker, a) }

// RandomString will take in a slice of string and return a randomly selected value
func (f *Faker) RandomString(a []string) string { return randomString(f, a) }

func randomString(f *Faker, a []string) string {
	size := len(a)
	if size == 0 {
		return ""
	}
	if size == 1 {
		return a[0]
	}
	return a[f.IntN(size)]
}

func addStringLookup() {
	AddFuncLookup("letter", Info{
		Display:     "Letter",
		Category:    "string",
		Description: "Character or symbol from the American Standard Code for Information Interchange (ASCII) character set",
		Example:     "g",
		Output:      "string",
		Aliases: []string{
			"alphabet", "character", "text symbol", "ascii char", "alphabetical sign",
		},
		Keywords: []string{
			"standard", "code", "information", "interchange", "set", "printable", "typography", "symbolic", "encoding",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return letter(f), nil
		},
	})

	AddFuncLookup("lettern", Info{
		Display:     "LetterN",
		Category:    "string",
		Description: "ASCII string with length N",
		Example:     "gbRMaRxHki",
		Output:      "string",
		Aliases: []string{
			"random letters", "ascii string", "text sequence", "generated letters", "alphabetical string",
		},
		Keywords: []string{
			"sequence", "multiple", "concatenated", "combined", "series", "generated", "batch", "collection",
		},
		Params: []Param{
			{Field: "count", Display: "Count", Type: "uint", Description: "Number of digits to generate"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			ui, err := info.GetUint(m, "count")
			if err != nil {
				return nil, err
			}

			return letterN(f, ui), nil
		},
	})

	AddFuncLookup("vowel", Info{
		Display:     "Vowel",
		Category:    "string",
		Description: "Speech sound produced with an open vocal tract",
		Example:     "a",
		Output:      "string",
		Aliases: []string{
			"vocal sound", "speech letter", "phonetic vowel", "linguistic vowel", "spoken sound",
		},
		Keywords: []string{
			"open", "e", "i", "o", "u", "phonetic", "linguistic", "articulation", "pronunciation", "syllable",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return vowel(f), nil
		},
	})

	AddFuncLookup("digit", Info{
		Display:     "Digit",
		Category:    "string",
		Description: "Numerical symbol used to represent numbers",
		Example:     "0",
		Output:      "string",
		Aliases: []string{
			"number symbol", "numeric character", "decimal digit", "ascii number", "numerical sign",
		},
		Keywords: []string{
			"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "decimal", "base10", "notation", "numeric",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return digit(f), nil
		},
	})

	AddFuncLookup("digitn", Info{
		Display:     "DigitN",
		Category:    "string",
		Description: "String of length N consisting of ASCII digits",
		Example:     "0136459948",
		Output:      "string",
		Aliases: []string{
			"numeric string", "digit sequence", "number series", "generated digits", "ascii digits",
		},
		Keywords: []string{
			"consisting", "multiple", "concatenated", "combined", "series", "numeric", "sequence", "continuous", "string", "digits",
		},
		Params: []Param{
			{Field: "count", Display: "Count", Type: "uint", Description: "Number of digits to generate"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			ui, err := info.GetUint(m, "count")
			if err != nil {
				return nil, err
			}

			return digitN(f, ui), nil
		},
	})

	AddFuncLookup("numerify", Info{
		Display:     "Numerify",
		Category:    "string",
		Description: "Replace # with random numerical values",
		Example:     "(###)###-#### => (555)867-5309",
		Output:      "string",
		Aliases: []string{
			"hash replace", "number substitute", "pattern filler", "digit replacer", "placeholder numbers",
		},
		Keywords: []string{
			"replace", "hash", "pound", "template", "placeholder", "format", "substitute", "pattern", "randomize", "masking",
		},
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "String value to replace #'s"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			return numerify(f, str), nil
		},
	})

	AddFuncLookup("lexify", Info{
		Display:     "Lexify",
		Category:    "string",
		Description: "Replace ? with random generated letters",
		Example:     "?????@??????.com => billy@mister.com",
		Output:      "string",
		Aliases: []string{
			"letter substitute", "pattern letters", "placeholder letters", "random letter filler", "character replacer",
		},
		Keywords: []string{
			"replace", "question", "mark", "template", "placeholder", "format", "substitute", "pattern", "randomize", "masking",
		},
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "String value to replace ?'s"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			return lexify(f, str), nil
		},
	})

	AddFuncLookup("shufflestrings", Info{
		Display:     "Shuffle Strings",
		Category:    "string",
		Description: "Shuffle an array of strings",
		Example:     "hello,world,whats,up => whats,world,hello,up",
		Output:      "[]string",
		ContentType: "application/json",
		Aliases: []string{
			"array shuffle", "list randomize", "string reorder", "string mixer", "sequence shuffle",
		},
		Keywords: []string{
			"collection", "list", "slice", "permutation", "randomized", "scrambled", "jumbled", "unordered",
		},
		Params: []Param{
			{Field: "strs", Display: "Strings", Type: "[]string", Description: "Delimited separated strings"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			strs, err := info.GetStringArray(m, "strs")
			if err != nil {
				return nil, err
			}

			shuffleStrings(f, strs)

			return strs, nil
		},
	})

	AddFuncLookup("randomstring", Info{
		Display:     "Random String",
		Category:    "string",
		Description: "Return a random string from a string array",
		Example:     "hello,world,whats,up => world",
		Output:      "string",
		Aliases: []string{
			"string picker", "array choice", "string select", "random pick", "string chooser",
		},
		Keywords: []string{
			"selection", "chosen", "picked", "random", "list", "slice", "array", "choice", "element", "option",
		},
		Params: []Param{
			{Field: "strs", Display: "Strings", Type: "[]string", Description: "Delimited separated strings"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			strs, err := info.GetStringArray(m, "strs")
			if err != nil {
				return nil, err
			}

			return randomString(f, strs), nil
		},
	})

}
