package gofakeit

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"reflect"
	"strings"
	"text/template"
)

// TemplateOptions defines values needed for template document generation
type TemplateOptions struct {
	Funcs template.FuncMap `fake:"-"`
	Data  any              `json:"data" xml:"data" fake:"-"`
}

// Template generates an document based on the the supplied template
func Template(template string, co *TemplateOptions) (string, error) {
	if co == nil {
		co = &TemplateOptions{}
		GlobalFaker.Struct(co)
	}
	return templateFunc(template, templateFuncMap(GlobalFaker, &co.Funcs), co)
}

// Template generates an document based on the the supplied template
func (f *Faker) Template(template string, co *TemplateOptions) (string, error) {
	if co == nil {
		co = &TemplateOptions{}
		f.Struct(co)
	}
	return templateFunc(template, templateFuncMap(f, &co.Funcs), co)
}

// MarkdownOptions defines values needed for markdown document generation
type MarkdownOptions struct {
}

// Template for Markdown
const templateMarkdown = `
{{$repo := Gamertag}}
{{$language := RandomString (SliceString "go" "python" "javascript")}}
{{$username := Gamertag}}
{{$weightedSlice := SliceAny "github.com" "gitlab.com" "bitbucket.org"}}
{{$weightedWeights := SliceF32 5 1 1}}
{{$domain := Weighted $weightedSlice $weightedWeights}}
{{$action := RandomString (SliceString "process" "run" "execute" "perform" "handle")}}
{{$usage := RandomString (SliceString "whimsical story" "quirky message" "playful alert" "funny request" "lighthearted command")}}
{{$result := RandomString (SliceString "success" "error" "unknown" "completed" "failed" "finished" "in progress" "terminated")}}

# {{$repo}}

*Author: {{FirstName}} {{LastName}}*

{{Paragraph 2 5 7 "\n\n"}}

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)

## Installation
{{if eq $language "go"}}'''go
go get {{$domain}}/{{$username}}/{{$repo}}
'''{{else if eq $language "python"}}'''bash
pip install {{$repo}}
'''{{else if eq $language "javascript"}}'''js
npm install {{$repo}}
'''{{end}}

## Usage
{{if eq $language "go"}}'''go
result := {{$repo}}.{{$action}}("{{ToLower $usage}}")
fmt.Println("{{ToLower $repo}} result:", "{{ToLower $result}}")
'''{{else if eq $language "python"}}'''python
result = {{ToLower $repo}}.{{$action}}("{{ToLower $usage}}")
print("{{ToLower $repo}} result:", "{{ToLower $result}}")
'''{{else if eq $language "javascript"}}'''javascript
const result = {{ToLower $repo}}.{{$action}}("{{ToLower $usage}}");
console.log("{{ToLower $repo}} result:", "{{ToLower $result}}");
'''{{end}}

## License
{{RandomString (SliceString "MIT" "Apache 2.0" "GPL-3.0" "BSD-3-Clause" "ISC")}}
`

// Markdown will return a single random Markdown template document
func Markdown(co *MarkdownOptions) (string, error) {
	if co == nil {
		co = &MarkdownOptions{}
		GlobalFaker.Struct(co)
	}
	return templateFunc(templateMarkdown, templateFuncMap(GlobalFaker, nil), co)
}

// Markdown will return a single random Markdown template document
func (f *Faker) Markdown(co *MarkdownOptions) (string, error) {
	if co == nil {
		co = &MarkdownOptions{}
		f.Struct(co)
	}
	return templateFunc(templateMarkdown, templateFuncMap(f, nil), co)
}

// EmailOptions defines values needed for email document generation
type EmailOptions struct {
}

// Template for email text
const templateEmail = `
Subject: {{RandomString (SliceString "Greetings" "Hello" "Hi")}} from {{FirstName}}!

Dear {{LastName}},

{{RandomString (SliceString "Greetings!" "Hello there!" "Hi, how are you?")}} {{RandomString (SliceString "How's everything going?" "I hope your day is going well." "Sending positive vibes your way.")}}

{{RandomString (SliceString "I trust this email finds you well." "I hope you're doing great." "Hoping this message reaches you in good spirits.")}} {{RandomString (SliceString  "Wishing you a fantastic day!" "May your week be filled with joy." "Sending good vibes your way.")}}

{{Paragraph 3 5 10 "\n\n"}}

{{RandomString (SliceString "I would appreciate your thoughts on" "I'm eager to hear your feedback on" "I'm curious to know what you think about")}} it. If you have a moment, please feel free to check out the project on {{RandomString (SliceString "GitHub" "GitLab" "Bitbucket")}}

{{RandomString (SliceString "Your insights would be invaluable." "I'm eager to hear what you think." "Feel free to share your opinions with me.")}} {{RandomString (SliceString "Looking forward to your feedback!" "Your perspective is highly valued." "Your thoughts matter to me.")}}

{{RandomString (SliceString "Thank you for your consideration!" "I appreciate your attention to this matter." "Your support means a lot to me.")}} {{RandomString (SliceString "Wishing you a wonderful day!" "Thanks in advance for your time." "Your feedback is greatly appreciated.")}}

{{RandomString (SliceString "Warm regards" "Best wishes" "Kind regards" "Sincerely" "With gratitude")}}
{{FirstName}} {{LastName}}
{{Email}}
{{PhoneFormatted}}
`

// EmailText will return a single random text email template document
func EmailText(co *EmailOptions) (string, error) {
	if co == nil {
		co = &EmailOptions{}
		GlobalFaker.Struct(co)
	}
	return templateFunc(templateEmail, templateFuncMap(GlobalFaker, nil), co)
}

// EmailText will return a single random text email template document
func (f *Faker) EmailText(co *EmailOptions) (string, error) {
	if co == nil {
		co = &EmailOptions{}
		f.Struct(co)
	}
	return templateFunc(templateEmail, templateFuncMap(f, nil), co)
}

// functions that wont work with template engine
var templateExclusion = []string{
	"RandomMapKey",
	"SQL",
	"Template",
}

// Build the template.FuncMap for the template engine
func templateFuncMap(f *Faker, fm *template.FuncMap) *template.FuncMap {

	// create a new function map
	funcMap := template.FuncMap{}

	v := reflect.ValueOf(f)

	// loop through the methods
	for i := 0; i < v.NumMethod(); i++ {
		// check if the method is in the exclusion list
		if stringInSlice(v.Type().Method(i).Name, templateExclusion) {
			continue
		}

		// Check if method has return values
		// If not don't add to function map
		if v.Type().Method(i).Type.NumOut() == 0 {
			continue
		}

		// add the method to the function map
		funcMap[v.Type().Method(i).Name] = v.Method(i).Interface()
	}

	// make string upper case
	funcMap["ToUpper"] = strings.ToUpper

	// make string lower case
	funcMap["ToLower"] = strings.ToLower

	// make string title case
	funcMap["IntRange"] = func(start, end int) []int {
		n := end - start + 1
		result := make([]int, n)
		for i := 0; i < n; i++ {
			result[i] = start + i
		}
		return result
	}

	// enable passing any type to return a string
	funcMap["ToInt"] = func(args any) int {
		switch v := args.(type) {
		case string:
			i, err := strconv.Atoi(v)
			if err != nil {
				return 0
			}

			return i
		case float64:
			return int(v)
		case float32:
			return int(v)
		case int:
			return v

		// Anything else return 0
		default:
			return 0
		}
	}

	// enable passing any type to return a float64
	funcMap["ToFloat"] = func(args any) float64 {
		switch v := args.(type) {
		case string:
			i, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return 0
			}

			return i
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)

		// Anything else return 0
		default:
			return 0
		}
	}

	// ensable passing any type to return a string
	funcMap["ToString"] = func(args any) string {
		switch v := args.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case float32:
			return strconv.FormatFloat(float64(v), 'f', -1, 32)
		case int:
			return strconv.Itoa(v)

		// Anything else return empty string
		default:
			return ""
		}
	}

	// function to convert string to date time
	funcMap["ToDate"] = func(dateString string) time.Time {
		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			return time.Now()
		}
		return date
	}

	// enable passing slice of interface to functions
	funcMap["SliceAny"] = func(args ...any) []any {
		return args
	}

	// enable passing slice of string to functions
	funcMap["SliceString"] = func(args ...string) []string {
		return args
	}

	// enable passing slice of uint to functions
	funcMap["SliceUInt"] = func(args ...uint) []uint {
		return args
	}

	// enable passing slice of int to functions
	funcMap["SliceInt"] = func(args ...int) []int {
		return args
	}

	// enable passing slice of int to functions
	funcMap["SliceF32"] = func(args ...float32) []float32 {
		return args
	}

	// Add passed in function map to the function map
	if fm != nil {
		for k, v := range *fm {
			funcMap[k] = v
		}
	}

	return &funcMap
}

// function to build the function map for the template engine from the global faker
func templateFunc(temp string, funcs *template.FuncMap, data any) (string, error) {
	if temp == "" {
		return "", fmt.Errorf("template parameter is empty")
	}

	// Create a new template and parse
	template_gen, err := template.New("CodeRun").Funcs(*funcs).Parse(temp)
	if err != nil {
		return "", err
	}

	b := &bytes.Buffer{}
	err = template_gen.Execute(b, data)
	if err != nil {
		return "", err
	}

	// Return the result
	return strings.ReplaceAll(b.String(), "\\n", "\n"), nil

}

// addTemplateLookup will add the template functions to the global lookup
func addTemplateLookup() {
	AddFuncLookup("template", Info{
		Display:     "Template",
		Category:    "template",
		Description: "Generates document from template",
		Example: `{{Firstname}} {{Lastname}}

// output
Markus Moen`,
		Output:      "string",
		ContentType: "text/plain",
		Aliases: []string{
			"document template", "layout", "blueprint", "design pattern", "text template", "generator", "format schema",
		},
		Keywords: []string{
			"template", "generates", "format", "structure", "engine", "document", "pattern", "design", "syntax", "render", "compile",
		},
		Params: []Param{
			{Field: "template", Display: "Template", Type: "string", Description: "Golang template to generate the document from"},
			{Field: "data", Display: "Custom Data", Type: "string", Default: "", Optional: true, Description: "Custom data to pass to the template"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			tpl, err := info.GetString(m, "template")
			if err != nil {
				return nil, err
			}

			data, err := info.GetAny(m, "data")
			if err != nil {
				return nil, err
			}

			templateOut, err := templateFunc(tpl, templateFuncMap(f, nil), &TemplateOptions{Data: data})
			if err != nil {
				return nil, err
			}

			return templateOut, nil
		},
	})

	AddFuncLookup("markdown", Info{
		Display:     "Random markdown document",
		Category:    "template",
		Description: "Lightweight markup language used for formatting plain text",
		Example: `# PurpleSheep5

*Author: Amie Feil*

Quarterly without week it hungry thing someone. Him regularly today whomever this revolt hence. From his timing as quantity us these. Yours live these frantic not may another. How this ours his them those whose.

Them batch its Iraqi most that few. Abroad cheese this whereas next how there. Gorgeous genetics time choir fiction therefore yourselves. Am those infrequently heap software quarterly rather. Punctuation yellow where several his orchard to.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [License](#license)

## Installation
'''bash
pip install PurpleSheep5
'''

## Usage
'''python
result = purplesheep5.process("funny request")
print("purplesheep5 result:", "in progress")
'''

## License
MIT`,
		Output: "string",
		Aliases: []string{
			"markup language", "readme format", "lightweight markup", "documentation style", "plain text format", "md file", "doc format",
		},
		Keywords: []string{
			"markdown", "markup", "language", "formatting", "plain", "text", "documentation", "lightweight", "syntax", "rendering", "structure", "readme",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			template_result, err := templateFunc(templateMarkdown, templateFuncMap(f, nil), &MarkdownOptions{})
			return string(template_result), err
		},
	})

	AddFuncLookup("email_text", Info{
		Display:     "Random text email Document",
		Category:    "template",
		Description: "Written content of an email message, including the sender's message to the recipient",
		Example: `Subject: Greetings from Marcel!

Dear Pagac,

Hello there! Sending positive vibes your way.

I hope you're doing great. May your week be filled with joy.

Virtually woman where team late quarterly without week it hungry. Thing someone him regularly today whomever this revolt hence from. His timing as quantity us these yours live these frantic. Not may another how this ours his them those whose. Them batch its Iraqi most that few abroad cheese this.

Whereas next how there gorgeous genetics time choir fiction therefore. Yourselves am those infrequently heap software quarterly rather punctuation yellow. Where several his orchard to frequently hence victorious boxers each. Does auspicious yourselves first soup tomorrow this that must conclude. Anyway some yearly who cough laugh himself both yet rarely.

Me dolphin intensely block would leap plane us first then. Down them eager would hundred super throughout animal yet themselves. Been group flock shake part purchase up usually it her. None it hers boat what their there Turkmen moreover one. Lebanese to brace these shower in it everybody should whatever.

I'm curious to know what you think about it. If you have a moment, please feel free to check out the project on Bitbucket

I'm eager to hear what you think. Looking forward to your feedback!

Thank you for your consideration! Thanks in advance for your time.

Kind regards
Milford Johnston
jamelhaag@king.org
(507)096-3058`,
		Output: "string",
		Aliases: []string{
			"email body",
			"email text",
			"email message",
			"message body",
			"email content",
		},
		Keywords: []string{
			"email", "body", "message", "content",
			"subject", "salutation", "greeting", "closing",
			"signature", "footer", "paragraph", "plaintext",
			"correspondence", "communication",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			template_result, err := templateFunc(templateEmail, templateFuncMap(f, nil), &EmailOptions{})
			return string(template_result), err
		},
	})
}
