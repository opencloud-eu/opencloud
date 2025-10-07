package gofakeit

import (
	"github.com/brianvoe/gofakeit/v7/data"
)

// Color will generate a random color string
func Color() string { return color(GlobalFaker) }

// Color will generate a random color string
func (f *Faker) Color() string { return color(f) }

func color(f *Faker) string { return getRandValue(f, []string{"color", "full"}) }

// NiceColor will generate a random safe color string
func NiceColors() []string { return niceColors(GlobalFaker) }

// NiceColor will generate a random safe color string
func (f *Faker) NiceColors() []string { return niceColors(f) }

func niceColors(f *Faker) []string {
	return data.ColorsNice[randIntRange(f, 0, len(data.ColorsNice)-1)]
}

// SafeColor will generate a random safe color string
func SafeColor() string { return safeColor(GlobalFaker) }

// SafeColor will generate a random safe color string
func (f *Faker) SafeColor() string { return safeColor(f) }

func safeColor(f *Faker) string { return getRandValue(f, []string{"color", "safe"}) }

// HexColor will generate a random hexadecimal color string
func HexColor() string { return hexColor(GlobalFaker) }

// HexColor will generate a random hexadecimal color string
func (f *Faker) HexColor() string { return hexColor(f) }

func hexColor(f *Faker) string {
	color := make([]byte, 6)
	hashQuestion := []byte("?#")
	for i := 0; i < 6; i++ {
		color[i] = hashQuestion[f.IntN(2)]
	}

	return "#" + replaceWithHexLetters(f, replaceWithNumbers(f, string(color)))
}

// RGBColor will generate a random int slice color
func RGBColor() []int { return rgbColor(GlobalFaker) }

// RGBColor will generate a random int slice color
func (f *Faker) RGBColor() []int { return rgbColor(f) }

func rgbColor(f *Faker) []int {
	return []int{randIntRange(f, 0, 255), randIntRange(f, 0, 255), randIntRange(f, 0, 255)}
}

func addColorLookup() {
	AddFuncLookup("color", Info{
		Display:     "Color",
		Category:    "color",
		Description: "Hue seen by the eye, returns the name of the color like red or blue",
		Example:     "MediumOrchid",
		Output:      "string",
		Aliases:     []string{"color name", "hue name", "visual color", "shade name", "color label"},
		Keywords:    []string{"color", "red", "blue", "green", "yellow", "purple", "orange", "pink", "hue", "chroma", "shade", "tone", "css", "name"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return color(f), nil
		},
	})

	AddFuncLookup("nicecolors", Info{
		Display:     "Nice Colors",
		Category:    "color",
		Description: "Attractive and appealing combinations of colors, returns a list of color hex codes",
		Example:     `["#cfffdd","#b4dec1","#5c5863","#a85163","#ff1f4c"]`,
		Output:      "[]string",
		ContentType: "application/json",
		Aliases:     []string{"color palette", "nice palette", "harmonious colors", "aesthetic palette", "design colors"},
		Keywords:    []string{"nice", "colors", "palette", "array", "hex", "design", "aesthetic", "beautiful", "harmonious", "scheme", "ui", "ux"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return niceColors(f), nil
		},
	})

	AddFuncLookup("safecolor", Info{
		Display:     "Safe Color",
		Category:    "color",
		Description: "Colors displayed consistently on different web browsers and devices",
		Example:     "black",
		Output:      "string",
		Aliases:     []string{"web safe color", "browser safe", "cross platform color", "universal color", "standard color"},
		Keywords:    []string{"safe", "color", "cross-platform", "css", "html", "compatible", "browser", "device", "universal", "stable", "standard"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return safeColor(f), nil
		},
	})

	AddFuncLookup("hexcolor", Info{
		Display:     "Hex Color",
		Category:    "color",
		Description: "Six-digit hexadecimal code representing a color in the RGB color model",
		Example:     "#a99fb4",
		Output:      "string",
		Aliases:     []string{"hex color code", "css hex", "html hex", "web hex", "hexadecimal color"},
		Keywords:    []string{"hex", "hexcolor", "color", "rgb", "six-digit", "web", "css", "html", "design", "hexadecimal", "hash", "code"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return hexColor(f), nil
		},
	})

	AddFuncLookup("rgbcolor", Info{
		Display:     "RGB Color",
		Category:    "color",
		Description: "Color defined by red, green, and blue light values",
		Example:     "[85, 224, 195]",
		Output:      "[]int",
		ContentType: "application/json",
		Aliases:     []string{"rgb triplet", "rgb array", "rgb value", "red green blue", "rgb color code"},
		Keywords:    []string{"rgb", "color", "red", "green", "blue", "triplet", "digital", "screen", "display", "primary", "additive", "value", "css"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return rgbColor(f), nil
		},
	})

}
