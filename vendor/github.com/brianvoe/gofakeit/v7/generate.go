package gofakeit

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp/syntax"
	"strings"
)

// Generate fake information from given string.
// Replaceable values should be within {}
//
// Functions
// Ex: {firstname} - billy
// Ex: {sentence:3} - Record river mind.
// Ex: {number:1,10} - 4
// Ex: {uuid} - 590c1440-9888-45b0-bd51-a817ee07c3f2
//
// Letters/Numbers
// Ex: ### - 481 - random numbers
// Ex: ??? - fda - random letters
//
// For a complete list of runnable functions use FuncsLookup
func Generate(dataVal string) (string, error) { return generate(GlobalFaker, dataVal) }

// Generate fake information from given string.
// Replaceable values should be within {}
//
// Functions
// Ex: {firstname} - billy
// Ex: {sentence:3} - Record river mind.
// Ex: {number:1,10} - 4
// Ex: {uuid} - 590c1440-9888-45b0-bd51-a817ee07c3f2
//
// Letters/Numbers
// Ex: ### - 481 - random numbers
// Ex: ??? - fda - random letters
//
// For a complete list of runnable functions use FuncsLookup
func (f *Faker) Generate(dataVal string) (string, error) { return generate(f, dataVal) }

func generate(f *Faker, dataVal string) (string, error) {
	// Replace # with numbers and ? with letters
	dataVal = replaceWithNumbers(f, dataVal)
	dataVal = replaceWithLetters(f, dataVal)

	// Check if string has any replaceable values
	// Even if it doesnt its ok we will just return the string
	if !strings.Contains(dataVal, "{") && !strings.Contains(dataVal, "}") {
		return dataVal, nil
	}

	// Variables to identify the index in which it exists
	startCurly := -1
	startCurlyIgnore := []int{}
	endCurly := -1
	endCurlyIgnore := []int{}

	// Loop through string characters
	for i := 0; i < len(dataVal); i++ {
		// Check for ignores if equal skip
		shouldSkip := false
		for _, igs := range startCurlyIgnore {
			if i == igs {
				shouldSkip = true
			}
		}
		for _, ige := range endCurlyIgnore {
			if i == ige {
				shouldSkip = true
			}
		}
		if shouldSkip {
			continue
		}

		// Identify items between brackets. Ex: {firstname}
		if string(dataVal[i]) == "{" {
			startCurly = i
			continue
		}
		if startCurly != -1 && string(dataVal[i]) == "}" {
			endCurly = i
		}
		if startCurly == -1 || endCurly == -1 {
			continue
		}

		// Get the value between brackets
		fParts := dataVal[startCurly+1 : endCurly]

		// Check if has params separated by :
		fNameSplit := strings.SplitN(fParts, ":", 2)
		fName := ""
		fParams := ""
		if len(fNameSplit) >= 1 {
			fName = fNameSplit[0]
		}
		if len(fNameSplit) >= 2 {
			fParams = fNameSplit[1]
		}

		// Check to see if its a replaceable lookup function
		if info := GetFuncLookup(fName); info != nil {
			// Get parameters, make sure params and the split both have values
			mapParams := NewMapParams()
			paramsLen := len(info.Params)

			// If just one param and its a string simply just pass it
			if paramsLen == 1 && info.Params[0].Type == "string" {
				mapParams.Add(info.Params[0].Field, fParams)
			} else if paramsLen > 0 && fParams != "" {
				var err error
				splitVals, err := funcLookupSplit(fParams)
				if err != nil {
					return "", err
				}
				mapParams, err = addSplitValsToMapParams(splitVals, info, mapParams)
				if err != nil {
					return "", err
				}
			}
			if mapParams.Size() == 0 {
				mapParams = nil
			}

			// Call function
			fValue, err := info.Generate(f, mapParams, info)
			if err != nil {
				return "", err
			}

			// Successfully found, run replace with new value
			dataVal = strings.Replace(dataVal, "{"+fParts+"}", fmt.Sprintf("%v", fValue), 1)

			// Reset the curly index back to -1 and reset ignores
			startCurly = -1
			startCurlyIgnore = []int{}
			endCurly = -1
			endCurlyIgnore = []int{}
			i = -1 // Reset back to the start of the string
			continue
		}

		// Couldnt find anything - mark curly brackets to skip and rerun
		startCurlyIgnore = append(startCurlyIgnore, startCurly)
		endCurlyIgnore = append(endCurlyIgnore, endCurly)

		// Reset the curly index back to -1
		startCurly = -1
		endCurly = -1
		i = -1 // Reset back to the start of the string
		continue
	}

	return dataVal, nil
}

// FixedWidthOptions defines values needed for csv generation
type FixedWidthOptions struct {
	RowCount int     `json:"row_count" xml:"row_count" fake:"{number:1,10}"`
	Fields   []Field `json:"fields" xml:"fields" fake:"{fields}"`
}

// FixedWidth generates an table of random data in fixed width format
// A nil FixedWidthOptions returns a randomly structured FixedWidth.
func FixedWidth(co *FixedWidthOptions) (string, error) { return fixeWidthFunc(GlobalFaker, co) }

// FixedWidth generates an table of random data in fixed width format
// A nil FixedWidthOptions returns a randomly structured FixedWidth.
func (f *Faker) FixedWidth(co *FixedWidthOptions) (string, error) { return fixeWidthFunc(f, co) }

// Function to generate a fixed width document
func fixeWidthFunc(f *Faker, co *FixedWidthOptions) (string, error) {
	// If we didn't get FixedWidthOptions, create a new random one
	if co == nil {
		co = &FixedWidthOptions{}
	}

	// Make sure you set a row count
	if co.RowCount <= 0 {
		co.RowCount = f.IntN(10) + 1
	}

	// Check fields
	if len(co.Fields) <= 0 {
		// Create random fields
		co.Fields = []Field{
			{Name: "Name", Function: "{firstname} {lastname}"},
			{Name: "Email", Function: "email"},
			{Name: "Password", Function: "password", Params: MapParams{"special": {"false"}, "space": {"false"}}},
		}
	}

	data := [][]string{}
	hasHeader := false

	// Loop through fields, generate data and add to data array
	for _, field := range co.Fields {
		// Start new row
		row := []string{}

		// Add name to first value
		if field.Name != "" {
			hasHeader = true
		}
		row = append(row, field.Name)

		// Get function
		funcInfo := GetFuncLookup(field.Function)
		var value any
		if funcInfo == nil {
			// Try to run the function through generate
			for i := 0; i < co.RowCount; i++ {
				genStr, err := generate(f, field.Function)
				if err != nil {
					return "", err
				}

				row = append(row, genStr)
			}
		} else {
			// Generate function value
			var err error
			for i := 0; i < co.RowCount; i++ {
				value, err = funcInfo.Generate(f, &field.Params, funcInfo)
				if err != nil {
					return "", err
				}

				// Add value to row
				row = append(row, anyToString(value))
			}
		}

		// Add row to data
		data = append(data, row)
	}

	var result strings.Builder

	// Calculate column widths
	colWidths := make([]int, len(data))
	for i, row := range data {
		for _, value := range row {
			width := len(value) + 5
			if width > colWidths[i] {
				colWidths[i] = width
			}
		}
	}

	// Append table rows to the string, excluding the entire row if the first value is empty
	for i := 0; i < len(data[0]); i++ {
		if !hasHeader && i == 0 {
			continue // Skip the entire column if the first value is empty
		}

		var resultRow strings.Builder
		for j, row := range data {
			resultRow.WriteString(fmt.Sprintf("%-*s", colWidths[j], row[i]))
		}

		// Trim trailing spaces
		result.WriteString(strings.TrimRight(resultRow.String(), " "))

		// Only add new line if not the last row
		if i != len(data[0])-1 {
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// Regex will generate a string based upon a RE2 syntax
func Regex(regexStr string) string { return regex(GlobalFaker, regexStr) }

// Regex will generate a string based upon a RE2 syntax
func (f *Faker) Regex(regexStr string) string { return regex(f, regexStr) }

func regex(f *Faker, regexStr string) (gen string) {
	re, err := syntax.Parse(regexStr, syntax.Perl)
	if err != nil {
		return "Could not parse regex string"
	}

	// Panic catch
	defer func() {
		if r := recover(); r != nil {
			gen = fmt.Sprint(f)
			return

		}
	}()

	return regexGenerate(f, re, len(regexStr)*100)
}

func regexGenerate(f *Faker, re *syntax.Regexp, limit int) string {
	if limit <= 0 {
		panic("Length limit reached when generating output")
	}

	op := re.Op
	switch op {
	case syntax.OpNoMatch: // matches no strings
		// Do Nothing
	case syntax.OpEmptyMatch: // matches empty string
		return ""
	case syntax.OpLiteral: // matches Runes sequence
		var b strings.Builder
		for _, ru := range re.Rune {
			b.WriteRune(ru)
		}
		return b.String()
	case syntax.OpCharClass: // matches Runes interpreted as range pair list
		// number of possible chars
		sum := 0
		for i := 0; i < len(re.Rune); i += 2 {
			sum += int(re.Rune[i+1]-re.Rune[i]) + 1
			if re.Rune[i+1] == 0x10ffff { // rune range end
				sum = -1
				break
			}
		}

		// pick random char in range (inverse match group)
		if sum == -1 {
			chars := []uint8{}
			for j := 0; j < len(allStr); j++ {
				c := allStr[j]

				// Check c in range
				for i := 0; i < len(re.Rune); i += 2 {
					if rune(c) >= re.Rune[i] && rune(c) <= re.Rune[i+1] {
						chars = append(chars, c)
						break
					}
				}
			}
			if len(chars) > 0 {
				return string([]byte{chars[f.IntN(len(chars))]})
			}
		}

		r := f.IntN(int(sum))
		var ru rune
		sum = 0
		for i := 0; i < len(re.Rune); i += 2 {
			gap := int(re.Rune[i+1]-re.Rune[i]) + 1
			if sum+gap > r {
				ru = re.Rune[i] + rune(r-sum)
				break
			}
			sum += gap
		}

		return string(ru)
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar: // matches any character(and except newline)
		return randCharacter(f, allStr)
	case syntax.OpBeginLine: // matches empty string at beginning of line
	case syntax.OpEndLine: // matches empty string at end of line
	case syntax.OpBeginText: // matches empty string at beginning of text
	case syntax.OpEndText: // matches empty string at end of text
	case syntax.OpWordBoundary: // matches word boundary `\b`
	case syntax.OpNoWordBoundary: // matches word non-boundary `\B`
	case syntax.OpCapture: // capturing subexpression with index Cap, optional name Name
		return regexGenerate(f, re.Sub0[0], limit)
	case syntax.OpStar: // matches Sub[0] zero or more times
		var b strings.Builder
		for i := 0; i < number(f, 0, 10); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(f, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpPlus: // matches Sub[0] one or more times
		var b strings.Builder
		for i := 0; i < number(f, 1, 10); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(f, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpQuest: // matches Sub[0] zero or one times
		var b strings.Builder
		for i := 0; i < number(f, 0, 1); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(f, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpRepeat: // matches Sub[0] at least Min times, at most Max (Max == -1 is no limit)
		var b strings.Builder
		count := 0
		re.Max = int(math.Min(float64(re.Max), float64(10)))
		if re.Max > re.Min {
			count = f.IntN(re.Max - re.Min + 1)
		}
		for i := 0; i < re.Min || i < (re.Min+count); i++ {
			for _, rs := range re.Sub {
				b.WriteString(regexGenerate(f, rs, limit-b.Len()))
			}
		}
		return b.String()
	case syntax.OpConcat: // matches concatenation of Subs
		var b strings.Builder
		for _, rs := range re.Sub {
			b.WriteString(regexGenerate(f, rs, limit-b.Len()))
		}
		return b.String()
	case syntax.OpAlternate: // matches alternation of Subs
		return regexGenerate(f, re.Sub[number(f, 0, len(re.Sub)-1)], limit)
	}

	return ""
}

// Map will generate a random set of map data
func Map() map[string]any { return mapFunc(GlobalFaker) }

// Map will generate a random set of map data
func (f *Faker) Map() map[string]any { return mapFunc(f) }

func mapFunc(f *Faker) map[string]any {
	m := map[string]any{}

	randWordType := func() string {
		s := randomString(f, []string{"lorem", "bs", "job", "name", "address"})
		switch s {
		case "bs":
			return bs(f)
		case "job":
			return jobTitle(f)
		case "name":
			return name(f)
		case "address":
			return street(f) + ", " + city(f) + ", " + state(f) + " " + zip(f)
		}
		return word(f)
	}

	randSlice := func() []string {
		var sl []string
		for ii := 0; ii < number(f, 3, 10); ii++ {
			sl = append(sl, word(f))
		}
		return sl
	}

	for i := 0; i < number(f, 3, 10); i++ {
		t := randomString(f, []string{"string", "int", "float", "slice", "map"})
		switch t {
		case "string":
			m[word(f)] = randWordType()
		case "int":
			m[word(f)] = number(f, 1, 10000000)
		case "float":
			m[word(f)] = float32Range(f, 1, 1000000)
		case "slice":
			m[word(f)] = randSlice()
		case "map":
			mm := map[string]any{}
			tt := randomString(f, []string{"string", "int", "float", "slice"})
			switch tt {
			case "string":
				mm[word(f)] = randWordType()
			case "int":
				mm[word(f)] = number(f, 1, 10000000)
			case "float":
				mm[word(f)] = float32Range(f, 1, 1000000)
			case "slice":
				mm[word(f)] = randSlice()
			}
			m[word(f)] = mm
		}
	}

	return m
}

func addGenerateLookup() {
	AddFuncLookup("generate", Info{
		Display:     "Generate",
		Category:    "generate",
		Description: "Random string generated from string value based upon available data sets",
		Example:     "{firstname} {lastname} {email} - Markus Moen markusmoen@pagac.net",
		Output:      "string",
		Aliases: []string{
			"template expander",
			"placeholder interpolator",
			"variable substitution",
			"token formatter",
			"pattern builder",
			"macro resolver",
		},
		Keywords: []string{
			"generate", "upon", "datasets", "random",
			"string", "value", "available", "data",
			"sets", "based",
		},
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "String value to generate from"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			// Limit the length of the string passed
			if len(str) > 1000 {
				return nil, errors.New("string length is too large. limit to 1000 characters")
			}

			return generate(f, str)
		},
	})

	AddFuncLookup("fixed_width", Info{
		Display:     "Fixed Width",
		Category:    "generate",
		Description: "Fixed width rows of output data based on input fields",
		Example: `Name               Email                          Password         Age
Markus Moen        sylvanmraz@murphy.net          6VlvH6qqXc7g     13
Alayna Wuckert     santinostanton@carroll.biz     g7sLrS0gEwLO     46
Lura Lockman       zacherykuhic@feil.name         S8gV7Z64KlHG     12`,
		Output:      "[]byte",
		ContentType: "text/plain",
		Aliases: []string{
			"fixed rows", "columnar data", "padded text", "aligned output", "structured fields",
		},
		Keywords: []string{
			"tabular", "data", "format", "alignment", "columns", "rows", "layout", "monospace", "table", "presentation",
		},
		Params: []Param{
			{Field: "rowcount", Display: "Row Count", Type: "int", Default: "10", Description: "Number of rows"},
			{Field: "fields", Display: "Fields", Type: "[]Field", Description: "Fields name, function and params"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			co := FixedWidthOptions{}

			rowCount, err := info.GetInt(m, "rowcount")
			if err != nil {
				return nil, err
			}

			co.RowCount = rowCount

			fields, _ := info.GetStringArray(m, "fields")

			// Check to make sure fields has length
			if len(fields) > 0 {
				co.Fields = make([]Field, len(fields))
				for i, f := range fields {
					// Unmarshal fields string into fields array
					err = json.Unmarshal([]byte(f), &co.Fields[i])
					if err != nil {
						return nil, err
					}
				}
			} else {
				return nil, errors.New("missing fields")
			}

			out, err := fixeWidthFunc(f, &co)
			if err != nil {
				return nil, err
			}

			return out, nil
		},
	})

	AddFuncLookup("regex", Info{
		Display:     "Regex",
		Category:    "generate",
		Description: "Pattern-matching tool used in text processing to search and manipulate strings",
		Example:     "[abcdef]{5} - affec",
		Output:      "string",
		Aliases: []string{
			"regular expression",
			"string matcher",
			"text parser",
			"pattern engine",
			"token analyzer",
			"rule evaluator",
		},
		Keywords: []string{
			"regex", "strings", "re2", "syntax",
			"pattern-matching", "tool", "search",
			"validation", "compile", "replace",
		},
		Params: []Param{
			{Field: "str", Display: "String", Type: "string", Description: "Regex RE2 syntax string"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			str, err := info.GetString(m, "str")
			if err != nil {
				return nil, err
			}

			// Limit the length of the string passed
			if len(str) > 500 {
				return nil, errors.New("string length is too large. limit to 500 characters")
			}

			return regex(f, str), nil
		},
	})

	AddFuncLookup("map", Info{
		Display:     "Map",
		Category:    "generate",
		Description: "Data structure that stores key-value pairs",
		Example: `{
	"software": 7518355,
	"that": ["despite", "pack", "whereas", "recently", "there", "anyone", "time", "read"],
	"use": 683598,
	"whom": "innovate",
	"yourselves": 1987784
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Aliases: []string{
			"associative array",
			"lookup table",
			"symbol table",
			"keyed collection",
			"map structure",
			"object store",
		},
		Keywords: []string{
			"map", "stores", "key", "value",
			"dictionary", "hash", "collection",
			"pairs", "keys", "values", "structure",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return mapFunc(f), nil
		},
	})
}
