package gofakeit

import (
	"strconv"
)

// BeerName will return a random beer name
func BeerName() string {
	return beerName(GlobalFaker)
}

// BeerName will return a random beer name
func (f *Faker) BeerName() string {
	return beerName(f)
}

func beerName(f *Faker) string {
	return getRandValue(f, []string{"beer", "name"})
}

// BeerStyle will return a random beer style
func BeerStyle() string {
	return beerStyle(GlobalFaker)
}

// BeerStyle will return a random beer style
func (f *Faker) BeerStyle() string {
	return beerStyle(f)
}

func beerStyle(f *Faker) string {
	return getRandValue(f, []string{"beer", "style"})
}

// BeerHop will return a random beer hop
func BeerHop() string {
	return beerHop(GlobalFaker)
}

// BeerHop will return a random beer hop
func (f *Faker) BeerHop() string {
	return beerHop(f)
}

func beerHop(f *Faker) string {
	return getRandValue(f, []string{"beer", "hop"})
}

// BeerYeast will return a random beer yeast
func BeerYeast() string {
	return beerYeast(GlobalFaker)
}

// BeerYeast will return a random beer yeast
func (f *Faker) BeerYeast() string {
	return beerYeast(f)
}

func beerYeast(f *Faker) string {
	return getRandValue(f, []string{"beer", "yeast"})
}

// BeerMalt will return a random beer malt
func BeerMalt() string {
	return beerMalt(GlobalFaker)
}

// BeerMalt will return a random beer malt
func (f *Faker) BeerMalt() string {
	return beerMalt(f)
}

func beerMalt(f *Faker) string {
	return getRandValue(f, []string{"beer", "malt"})
}

// BeerAlcohol will return a random beer alcohol level between 2.0 and 10.0
func BeerAlcohol() string {
	return beerAlcohol(GlobalFaker)
}

// BeerAlcohol will return a random beer alcohol level between 2.0 and 10.0
func (f *Faker) BeerAlcohol() string {
	return beerAlcohol(f)
}

func beerAlcohol(f *Faker) string {
	return strconv.FormatFloat(float64Range(f, 2.0, 10.0), 'f', 1, 64) + "%"
}

// BeerIbu will return a random beer ibu value between 10 and 100
func BeerIbu() string {
	return beerIbu(GlobalFaker)
}

// BeerIbu will return a random beer ibu value between 10 and 100
func (f *Faker) BeerIbu() string {
	return beerIbu(f)
}

func beerIbu(f *Faker) string {
	return strconv.Itoa(randIntRange(f, 10, 100)) + " IBU"
}

// BeerBlg will return a random beer blg between 5.0 and 20.0
func BeerBlg() string {
	return beerBlg(GlobalFaker)
}

// BeerBlg will return a random beer blg between 5.0 and 20.0
func (f *Faker) BeerBlg() string {
	return beerBlg(f)
}

func beerBlg(f *Faker) string {
	return strconv.FormatFloat(float64Range(f, 5.0, 20.0), 'f', 1, 64) + "°Blg"
}

func addBeerLookup() {
	AddFuncLookup("beername", Info{
		Display:     "Beer Name",
		Category:    "beer",
		Description: "Specific brand or variety of beer",
		Example:     "Duvel",
		Output:      "string",
		Aliases:     []string{"brand", "brewery", "label", "craft", "microbrew"},
		Keywords:    []string{"beer", "name", "variety", "specific", "alcoholic", "beverage", "lager", "ale", "stout", "pilsner", "ipa"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerName(f), nil
		},
	})

	AddFuncLookup("beerstyle", Info{
		Display:     "Beer Style",
		Category:    "beer",
		Description: "Distinct characteristics and flavors of beer",
		Example:     "European Amber Lager",
		Output:      "string",
		Aliases:     []string{"style", "type", "category", "classification", "variety"},
		Keywords:    []string{"beer", "characteristics", "flavors", "distinct", "lager", "ale", "stout", "pilsner", "porter", "wheat", "amber"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerStyle(f), nil
		},
	})

	AddFuncLookup("beerhop", Info{
		Display:     "Beer Hop",
		Category:    "beer",
		Description: "The flower used in brewing to add flavor, aroma, and bitterness to beer",
		Example:     "Glacier",
		Output:      "string",
		Aliases:     []string{"hop", "flower", "plant", "cone", "vine"},
		Keywords:    []string{"beer", "brewing", "flavor", "aroma", "bitterness", "ingredient", "humulus", "lupulus", "cascade", "citra", "mosaic"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerHop(f), nil
		},
	})

	AddFuncLookup("beeryeast", Info{
		Display:     "Beer Yeast",
		Category:    "beer",
		Description: "Microorganism used in brewing to ferment sugars, producing alcohol and carbonation in beer",
		Example:     "1388 - Belgian Strong Ale",
		Output:      "string",
		Aliases:     []string{"yeast", "fungus", "microorganism", "culture", "strain"},
		Keywords:    []string{"beer", "brewing", "ferment", "sugars", "alcohol", "carbonation", "ingredient", "saccharomyces", "cerevisiae", "belgian", "ale"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerYeast(f), nil
		},
	})

	AddFuncLookup("beermalt", Info{
		Display:     "Beer Malt",
		Category:    "beer",
		Description: "Processed barley or other grains, provides sugars for fermentation and flavor to beer",
		Example:     "Munich",
		Output:      "string",
		Aliases:     []string{"malt", "barley", "grain", "cereal", "kernel"},
		Keywords:    []string{"beer", "sugars", "fermentation", "flavor", "processed", "ingredient", "munich", "pale", "crystal", "roasted", "wheat", "rye"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerMalt(f), nil
		},
	})

	AddFuncLookup("beeralcohol", Info{
		Display:     "Beer Alcohol",
		Category:    "beer",
		Description: "Measures the alcohol content in beer",
		Example:     "2.7%",
		Output:      "string",
		Aliases:     []string{"alcohol", "abv", "strength", "proof", "percentage"},
		Keywords:    []string{"beer", "content", "measure", "volume", "concentration", "level", "degree", "potency"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerAlcohol(f), nil
		},
	})

	AddFuncLookup("beeribu", Info{
		Display:     "Beer IBU",
		Category:    "beer",
		Description: "Scale measuring bitterness of beer from hops",
		Example:     "29 IBU",
		Output:      "string",
		Aliases:     []string{"ibu", "bitterness", "scale", "units", "measurement"},
		Keywords:    []string{"beer", "hops", "measuring", "international", "bittering", "alpha", "acid", "level", "intensity"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerIbu(f), nil
		},
	})

	AddFuncLookup("beerblg", Info{
		Display:     "Beer BLG",
		Category:    "beer",
		Description: "Scale indicating the concentration of extract in worts",
		Example:     "6.4°Blg",
		Output:      "string",
		Aliases:     []string{"blg", "density", "gravity", "extract", "concentration"},
		Keywords:    []string{"beer", "worts", "scale", "indicating", "balling", "plato", "sugar", "soluble", "solids", "degree"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return beerBlg(f), nil
		},
	})
}
