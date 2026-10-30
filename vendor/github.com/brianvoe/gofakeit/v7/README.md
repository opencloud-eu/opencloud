![Gofakeit](https://raw.githubusercontent.com/brianvoe/gofakeit/master/logo.png)

# Gofakeit [![Go Report Card](https://goreportcard.com/badge/github.com/brianvoe/gofakeit)](https://goreportcard.com/report/github.com/brianvoe/gofakeit) ![Test](https://github.com/brianvoe/gofakeit/workflows/Test/badge.svg?branch=master) [![GoDoc](https://godoc.org/github.com/brianvoe/gofakeit/v7?status.svg)](https://godoc.org/github.com/brianvoe/gofakeit/v7) [![license](http://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/brianvoe/gofakeit/master/LICENSE.txt)

Random data generator written in go

## Support

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/G2G0R5EJT) <a href="https://www.buymeacoffee.com/brianvoe" target="_blank"><img src="https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png" alt="Buy Me A Coffee" style="height: auto !important;width: auto !important;" ></a>

## Merch

[![Merch](https://raw.githubusercontent.com/brianvoe/gofakeit/master/merch.png)](https://gofakeit-buy-shop.fourthwall.com)

## Features

- [310+ Functions!!!](#functions)
- [Random Sources](#random-sources)
- [Global Rand](#global-rand-set)
- [Struct Generator](#struct)
- [Custom Functions](#custom-functions)
- [Templates](#templates)
- [Http Server](https://github.com/brianvoe/gofakeit/tree/master/cmd/gofakeitserver)
- [Command Line Tool](https://github.com/brianvoe/gofakeit/tree/master/cmd/gofakeit)
- Zero dependencies
- [Benchmarks](https://github.com/brianvoe/gofakeit/blob/master/BENCHMARKS.md)
- [Issue](https://github.com/brianvoe/gofakeit/issues)

## Contributors

Thank you to all our Gofakeit contributors!

<a href="https://github.com/brianvoe/gofakeit/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=brianvoe/gofakeit" />
</a>

## Installation

```go
go get github.com/brianvoe/gofakeit/v7
```

## Simple Usage

```go
import "github.com/brianvoe/gofakeit/v7"

gofakeit.Name()                // Markus Moen
gofakeit.Email()               // alaynawuckert@kozey.biz
gofakeit.Phone()               // (570)245-7485
gofakeit.BS()                  // front-end
gofakeit.BeerName()            // Duvel
gofakeit.Color()               // MediumOrchid
gofakeit.Company()             // Moen, Pagac and Wuckert
gofakeit.CreditCardNumber(nil) // 4287271570245748
gofakeit.HackerPhrase()        // Connecting the array won't do anything, we need to generate the haptic COM driver!
gofakeit.JobTitle()            // Director
gofakeit.CurrencyShort()       // USD
```

[See full list of functions](#functions)

## Seed

If you are using the default global usage and dont care about seeding no need to set anything.
Gofakeit will seed it with a cryptographically secure number.

If you need a reproducible outcome you can set it via the Seed function call. Every example in
this repo sets it for testing purposes.

```go
import "github.com/brianvoe/gofakeit/v7"

gofakeit.Seed(0) // If 0 will use crypto/rand to generate a number

// or

gofakeit.Seed(8675309) // Set it to whatever number you want
```

## Random Sources

Gofakeit has a few rand sources, by default it uses math/rand/v2 PCG which is a pseudo random number generator and is thread locked.

If you want to see other potential sources you can see the sub package [Source](https://github.com/brianvoe/gofakeit/tree/master/source) for more information.

```go
import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/brianvoe/gofakeit/v7/source"
	"math/rand/v2"
)

// Uses math/rand/v2(PCG Pseudo) with mutex locking
faker := gofakeit.New(0)

// NewFaker takes in a source and whether or not it should be thread safe
faker := gofakeit.NewFaker(src rand.Source, lock bool)

// PCG Pseudo
faker := gofakeit.NewFaker(rand.NewPCG(11, 11), true)

// ChaCha8
faker := gofakeit.NewFaker(rand.NewChaCha8([32]byte{0, 1, 2, 3, 4, 5}), true)


// Additional from Gofakeit sub package source

// JSF(Jenkins Small Fast)
faker := gofakeit.NewFaker(source.NewJSF(11), true)

// SFC(Simple Fast Counter)
faker := gofakeit.NewFaker(source.NewSFC(11), true)

// Crypto - Uses crypto/rand
faker := gofakeit.NewFaker(source.NewCrypto(), true)

// Dumb - simple incrementing number
faker := gofakeit.NewFaker(source.NewDumb(11), true)
```

## Global Rand Set

If you would like to use the simple function calls but need to use something like
crypto/rand you can override the default global with the random source that you want.

```go
import "github.com/brianvoe/gofakeit/v7"

gofakeit.GlobalFaker = gofakeit.New(0)
```

## Struct

Gofakeit can generate random data for struct fields. For the most part it covers all the basic type
as well as some non-basic like time.Time.

Struct fields can also use tags to more specifically generate data for that field type.

```go
import "github.com/brianvoe/gofakeit/v7"

// Create structs with random injected data
type Foo struct {
	Str           string
	Int           int
	Pointer       *int
	Name          string         `fake:"{firstname}"`         // Any available function all lowercase
	Sentence      string         `fake:"{sentence:3}"`        // Can call with parameters
	RandStr       string         `fake:"{randomstring:[hello,world]}"`
	Number        string         `fake:"{number:1,10}"`       // Comma separated for multiple values
	Regex         string         `fake:"{regex:[abcdef]{5}}"` // Generate string from regex
	Map           map[string]int `fakesize:"2"`
	Array         []string       `fakesize:"2"`
	ArrayRange    []string       `fakesize:"2,6"`
	Bar           Bar
	Skip          *string        `fake:"skip"`                // Set to "skip" to not generate data for
	SkipAlt       *string        `fake:"-"`                   // Set to "-" to not generate data for
	Created       time.Time                                   // Can take in a fake tag as well as a format tag
	CreatedFormat time.Time      `fake:"{year}-{month}-{day}" format:"2006-01-02"`
}

type Bar struct {
	Name    string
	Number  int
	Float   float32
}

// Pass your struct as a pointer
var f Foo
err := gofakeit.Struct(&f)

fmt.Println(f.Str)      		// hrukpttuezptneuvunh
fmt.Println(f.Int)      		// -7825289004089916589
fmt.Println(*f.Pointer) 		// -343806609094473732
fmt.Println(f.Name)     		// fred
fmt.Println(f.Sentence) 		// Record river mind.
fmt.Println(f.RandStr)          // world
fmt.Println(f.Number)   		// 4
fmt.Println(f.Regex)    		// cbdfc
fmt.Println(f.Map)    			// map[PxLIo:52 lxwnqhqc:846]
fmt.Println(f.Array)    		// cbdfc
fmt.Printf("%+v", f.Bar)    	// {Name:QFpZ Number:-2882647639396178786 Float:1.7636692e+37}
fmt.Println(f.Skip)     		// <nil>
fmt.Println(f.Created.String()) // 1908-12-07 04:14:25.685339029 +0000 UTC

// Supported formats
// int, int8, int16, int32, int64,
// uint, uint8, uint16, uint32, uint64,
// float32, float64,
// bool, string,
// array, pointers, map
// time.Time // If setting time you can also set a format tag
// Nested Struct Fields and Embedded Fields
```

## Fakeable types

It is possible to extend a struct by implementing the `Fakeable` interface
in order to control the generation.

For example, this is useful when it is not possible to modify the struct that you want to fake by adding struct tags to a field but you still need to be able to control the generation process.

```go
// Custom string that you want to generate your own data for
type Friend string

func (c *Friend) Fake(f *gofakeit.Faker) (any, error) {
	// Can call any other faker methods
	return f.RandomString([]string{"billy", "fred", "susan"}), nil
}

// Custom time that you want to generate your own data for
type Age time.Time

func (c *Age) Fake(f *gofakeit.Faker) (any, error) {
	return Age(f.DateRange(time.Now().AddDate(-100, 0, 0), time.Now().AddDate(-18, 0, 0))), nil
}

// This is the struct that we cannot modify to add struct tags
type User struct {
	Name Friend
	Age *Age
}

var u User
gofakeit.Struct(&u)
fmt.Println(u.Name)            // billy
fmt.Println(time.Time(*u.Age)) // 1990-12-07 04:14:25.685339029 +0000 UTC
```

## Custom Functions

In a lot of situations you may need to use your own random function usage for your specific needs.

If you would like to extend the usage of struct tags, generate function, available usages in the gofakeit server
or gofakeit command sub packages. You can do so via the AddFuncLookup. Each function has their own lookup, if
you need more reference examples you can look at each files lookups.

```go
// Simple
gofakeit.AddFuncLookup("friendname", gofakeit.Info{
	Category:    "custom",
	Description: "Random friend name",
	Example:     "bill",
	Output:      "string",
	Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
		return f.RandomString([]string{"bill", "bob", "sally"}), nil
	},
})

// With Params
gofakeit.AddFuncLookup("jumbleword", gofakeit.Info{
	Category:    "jumbleword",
	Description: "Take a word and jumble it up",
	Example:     "loredlowlh",
	Output:      "string",
	Params: []gofakeit.Param{
		{Field: "word", Type: "string", Description: "Word you want to jumble"},
	},
	Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
		word, err := info.GetString(m, "word")
		if err != nil {
			return nil, err
		}

		split := strings.Split(word, "")
		f.ShuffleStrings(split)
		return strings.Join(split, ""), nil
	},
})

type Foo struct {
	FriendName string `fake:"{friendname}"`
	JumbleWord string `fake:"{jumbleword:helloworld}"`
}

var f Foo
gofakeit.Struct(&f)
fmt.Println(f.FriendName) // bill
fmt.Println(f.JumbleWord) // loredlowlh
```

## Templates

Generate custom outputs using golang's template engine [https://pkg.go.dev/text/template](https://pkg.go.dev/text/template).

We have added all the available functions to the template engine as well as some additional ones that are useful for template building.

Additional Available Functions
```go
- ToUpper(s string) string   			// Make string upper case
- ToLower(s string) string   			// Make string lower case
- ToString(s any)            			// Convert to string
- ToDate(s string) time.Time 			// Convert string to date
- SpliceAny(args ...any) []any 			// Build a slice of anys, used with Weighted
- SpliceString(args ...string) []string // Build a slice of strings, used with Teams and RandomString
- SpliceUInt(args ...uint) []uint 		// Build a slice of uint, used with Dice and RandomUint
- SpliceInt(args ...int) []int 			// Build a slice of int, used with RandomInt
```

<details>
  <summary>Unavailable Gofakeit functions</summary>

```go
// Any functions that dont have a return value
- AnythingThatReturnsVoid(): void

// Not available to use in templates
- Template(co *TemplateOptions) ([]byte, error)
- RandomMapKey(mapI any) any
```
</details>


### Example Usages

```go
import "github.com/brianvoe/gofakeit/v7"

func main() {
	// Accessing the Lines variable from within the template.
	template := `
	Subject: {{RandomString (SliceString "Greetings" "Hello" "Hi")}}

	Dear {{LastName}},

	{{RandomString (SliceString "Greetings!" "Hello there!" "Hi, how are you?")}}

	{{Paragraph 1 5 10 "\n\n"}}

	{{RandomString (SliceString "Warm regards" "Best wishes" "Sincerely")}}
	{{$person:=Person}}
	{{$person.FirstName}} {{$person.LastName}}
	{{$person.Contact.Email}}
	{{$person.Contact.Phone}}
	`

	value, err := gofakeit.Template(template, &gofakeit.TemplateOptions{Data: 5})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(value)
}
```

Output:
```text
Subject: Hello

Dear Krajcik,

Greetings!

Quia voluptatem voluptatem voluptatem. Quia voluptatem voluptatem voluptatem. Quia voluptatem voluptatem voluptatem.

Warm regards

Kaitlyn Krajcik
kaitlynkrajcik@krajcik
570-245-7485
```

## Functions

All functions also exist as methods on the Faker struct

### File

Passing `nil` to `CSV`, `JSON` or `XML` will auto generate data using default values.

```go
CSV(co *CSVOptions) ([]byte, error)
JSON(jo *JSONOptions) ([]byte, error)
XML(xo *XMLOptions) ([]byte, error)
FileExtension() string
FileMimeType() string
```

### Template

Passing `nil` will auto generate data using default values.

```go
Template(co *TemplateOptions) (string, error)
Markdown(co *MarkdownOptions) (string, error)
EmailText(co *EmailOptions) (string, error)
FixedWidth(co *FixedWidthOptions) (string, error)
```

### Product

```go
Product() *ProductInfo
ProductName() string
ProductDescription() string
ProductCategory() string
ProductFeature() string
ProductMaterial() string
ProductUPC() string
ProductAudience() string
ProductDimension() string
ProductUseCase() string
ProductBenefit() string
ProductSuffix() string
ProductISBN(opts *ISBNOptions) string

```

### Person

```go
Person() *PersonInfo
Name() string
NamePrefix() string
NameSuffix() string
FirstName() string
MiddleName() string
LastName() string
Gender() string
SSN() string
EIN() string
Hobby() string
Contact() *ContactInfo
Email() string
Phone() string
PhoneFormatted() string
Teams(peopleArray []string, teamsArray []string) map[string][]string
```

### Generate

```go
Struct(v any)
Slice(v any)
Map() map[string]any
Generate(value string) string
Regex(value string) string
```

### Auth

```go
Username() string
Password(lower bool, upper bool, numeric bool, special bool, space bool, num int) string
```

### Address

```go
Address() *AddressInfo
City() string
Country() string
CountryAbr() string
State() string
StateAbr() string
Street() string
StreetName() string
StreetNumber() string
StreetPrefix() string
StreetSuffix() string
Unit() string
Zip() string
Latitude() float64
LatitudeInRange(min, max float64) (float64, error)
Longitude() float64
LongitudeInRange(min, max float64) (float64, error)
```

### Game

```go
Gamertag() string
Dice(numDice uint, sides []uint) []uint
```

### Beer

```go
BeerAlcohol() string
BeerBlg() string
BeerHop() string
BeerIbu() string
BeerMalt() string
BeerName() string
BeerStyle() string
BeerYeast() string
```

### Car

```go
Car() *CarInfo
CarMaker() string
CarModel() string
CarType() string
CarFuelType() string
CarTransmissionType() string
```

### Words

```go
// Nouns
Noun() string
NounCommon() string
NounConcrete() string
NounAbstract() string
NounCollectivePeople() string
NounCollectiveAnimal() string
NounCollectiveThing() string
NounCountable() string
NounUncountable() string

// Verbs
Verb() string
VerbAction() string
VerbLinking() string
VerbHelping() string

// Adverbs
Adverb() string
AdverbManner() string
AdverbDegree() string
AdverbPlace() string
AdverbTimeDefinite() string
AdverbTimeIndefinite() string
AdverbFrequencyDefinite() string
AdverbFrequencyIndefinite() string

// Propositions
Preposition() string
PrepositionSimple() string
PrepositionDouble() string
PrepositionCompound() string

// Adjectives
Adjective() string
AdjectiveDescriptive() string
AdjectiveQuantitative() string
AdjectiveProper() string
AdjectiveDemonstrative() string
AdjectivePossessive() string
AdjectiveInterrogative() string
AdjectiveIndefinite() string

// Pronouns
Pronoun() string
PronounPersonal() string
PronounObject() string
PronounPossessive() string
PronounReflective() string
PronounDemonstrative() string
PronounInterrogative() string
PronounRelative() string

// Connectives
Connective() string
ConnectiveTime() string
ConnectiveComparative() string
ConnectiveComplaint() string
ConnectiveListing() string
ConnectiveCasual() string
ConnectiveExamplify() string

// Words
Word() string

// Sentences
Sentence(wordCount int) string
Paragraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string
LoremIpsumWord() string
LoremIpsumSentence(wordCount int) string
LoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string
Question() string
Quote() string
Phrase() string
```

### Foods

```go
Fruit() string
Vegetable() string
Breakfast() string
Lunch() string
Dinner() string
Snack() string
Dessert() string
```

### Misc

```go
Bool() bool
UUID() string
Weighted(options []any, weights []float32) (any, error)
FlipACoin() string
RandomMapKey(mapI any) any
ShuffleAnySlice(v any)
```

### Colors

```go
Color() string
HexColor() string
RGBColor() []int
SafeColor() string
NiceColors() string
```

### Images

```go
Image(width int, height int) *img.RGBA
ImageJpeg(width int, height int) []byte
ImagePng(width int, height int) []byte
```

### Internet

```go
URL() string
DomainName() string
DomainSuffix() string
IPv4Address() string
IPv6Address() string
MacAddress() string
HTTPStatusCode() string
HTTPStatusCodeSimple() int
LogLevel(logType string) string
HTTPMethod() string
HTTPVersion() string
UserAgent() string
ChromeUserAgent() string
FirefoxUserAgent() string
OperaUserAgent() string
SafariUserAgent() string
```

### HTML

```go
InputName() string
Svg(options *SVGOptions) string
```

### Date/Time

```go
Date() time.Time
PastDate() time.Time
FutureDate() time.Time
DateRange(start, end time.Time) time.Time
NanoSecond() int
Second() int
Minute() int
Hour() int
Month() int
MonthString() string
Day() int
WeekDay() string
Year() int
TimeZone() string
TimeZoneAbv() string
TimeZoneFull() string
TimeZoneOffset() float32
TimeZoneRegion() string
```

### Payment

```go
Price(min, max float64) float64
CreditCard() *CreditCardInfo
CreditCardCvv() string
CreditCardExp() string
CreditCardNumber(*CreditCardOptions) string
CreditCardType() string
Currency() *CurrencyInfo
CurrencyLong() string
CurrencyShort() string
AchRouting() string
AchAccount() string
BitcoinAddress() string
BitcoinPrivateKey() string
BankName() string
BankType() string
```

### Finance

```go
Cusip() string
Isin() string
```

### Company

```go
BS() string
Blurb() string
BuzzWord() string
Company() string
CompanySuffix() string
Job() *JobInfo
JobDescriptor() string
JobLevel() string
JobTitle() string
Slogan() string
```

### Hacker

```go
HackerAbbreviation() string
HackerAdjective() string
Hackeringverb() string
HackerNoun() string
HackerPhrase() string
HackerVerb() string
```

### Hipster

```go
HipsterWord() string
HipsterSentence(wordCount int) string
HipsterParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string
```

### App

```go
AppName() string
AppVersion() string
AppAuthor() string
```

### Animal

```go
PetName() string
Animal() string
AnimalType() string
FarmAnimal() string
Cat() string
Dog() string
Bird() string
```

### Emoji

```go
Emoji() string
EmojiDescription() string
EmojiCategory() string
EmojiAlias() string
EmojiTag() string
```

### Language

```go
Language() string
LanguageAbbreviation() string
ProgrammingLanguage() string
ProgrammingLanguageBest() string
```

### Number

```go
Number(min int, max int) int
Int() int
IntN(n int) int
Int8() int8
Int16() int16
Int32() int32
Int64() int64
Uint() uint
UintN(n uint) uint
Uint8() uint8
Uint16() uint16
Uint32() uint32
Uint64() uint64
Float32() float32
Float32Range(min, max float32) float32
Float64() float64
Float64Range(min, max float64) float64
ShuffleInts(a []int)
RandomInt(i []int) int
HexUint(bitsize int) string
```

### String

```go
Digit() string
DigitN(n uint) string
Letter() string
LetterN(n uint) string
Lexify(str string) string
Numerify(str string) string
ShuffleStrings(a []string)
RandomString(a []string) string
```

### Celebrity

```go
CelebrityActor() string
CelebrityBusiness() string
CelebritySport() string
```

### Minecraft

```go
MinecraftOre() string
MinecraftWood() string
MinecraftArmorTier() string
MinecraftArmorPart() string
MinecraftWeapon() string
MinecraftTool() string
MinecraftDye() string
MinecraftFood() string
MinecraftAnimal() string
MinecraftVillagerJob() string
MinecraftVillagerStation() string
MinecraftVillagerLevel() string
MinecraftMobPassive() string
MinecraftMobNeutral() string
MinecraftMobHostile() string
MinecraftMobBoss() string
MinecraftBiome() string
MinecraftWeather() string
```

### Book

```go
Book() *BookInfo
BookTitle() string
BookAuthor() string
BookGenre() string
```

### Movie

```go
Movie() *MovieInfo
MovieName() string
MovieGenre() string
```

### Error

```go
Error() error
ErrorDatabase() error
ErrorGRPC() error
ErrorHTTP() error
ErrorHTTPClient() error
ErrorHTTPServer() error
ErrorInput() error
ErrorRuntime() error
```

### School

```go
School() string
```

### Song

```go
Song() *SongInfo
SongName() string
SongArtist() string
SongGenre() string
```