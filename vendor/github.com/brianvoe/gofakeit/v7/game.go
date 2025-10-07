package gofakeit

import (
	"fmt"
	"strings"
)

// Gamertag will generate a random video game username
func Gamertag() string { return gamertag(GlobalFaker) }

// Gamertag will generate a random video game username
func (f *Faker) Gamertag() string { return gamertag(f) }

func gamertag(f *Faker) string {
	str := ""
	num := number(f, 1, 4)
	switch num {
	case 1:
		str = fmt.Sprintf("%s%ser", title(nounConcrete(f)), title(verbAction(f)))
	case 2:
		str = fmt.Sprintf("%s%s", title(adjectiveDescriptive(f)), title(animal(f)))
	case 3:
		str = fmt.Sprintf("%s%s", title(adjectiveDescriptive(f)), title(nounConcrete(f)))
	case 4:
		str = fmt.Sprintf("%s%s", title(fruit(f)), title(adjectiveDescriptive(f)))
	}

	// Randomly determine if we should add a number
	if f.IntN(3) == 1 {
		str += digitN(f, uint(number(f, 1, 3)))
	}

	// Remove any spaces
	str = strings.Replace(str, " ", "", -1)

	return str
}

// Dice will generate a random set of dice
func Dice(numDice uint, sides []uint) []uint { return dice(GlobalFaker, numDice, sides) }

// Dice will generate a random set of dice
func (f *Faker) Dice(numDice uint, sides []uint) []uint { return dice(f, numDice, sides) }

func dice(f *Faker, numDice uint, sides []uint) []uint {
	dice := make([]uint, numDice)

	// If we dont have any sides well set the sides to 6
	if len(sides) == 0 {
		sides = []uint{6}
	}

	for i := range dice {
		// If sides[i] doesnt exist use the first side
		if len(sides)-1 < i {
			dice[i] = uint(number(f, 1, int(sides[0])))
		} else {
			dice[i] = uint(number(f, 1, int(sides[i])))
		}
	}

	return dice
}

func addGameLookup() {
	AddFuncLookup("gamertag", Info{
		Display:     "Gamertag",
		Category:    "game",
		Description: "User-selected online username or alias used for identification in games",
		Example:     "footinterpret63",
		Output:      "string",
		Aliases: []string{
			"player handle",
			"gaming nickname",
			"online tag",
			"user alias",
			"profile name",
		},
		Keywords: []string{
			"gamertag", "user-selected", "username",
			"alias", "identification", "online", "gaming",
			"video", "games", "player",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return gamertag(f), nil
		},
	})

	AddFuncLookup("dice", Info{
		Display:     "Dice",
		Category:    "game",
		Description: "Small, cube-shaped objects used in games of chance for random outcomes",
		Example:     "[5, 2, 3]",
		Output:      "[]uint",
		Aliases: []string{
			"rolling cubes",
			"chance cubes",
			"game dice",
			"random rollers",
			"luck blocks",
		},
		Keywords: []string{
			"dice", "games", "cube-shaped", "chance",
			"random", "outcomes", "roll", "sides",
			"objects", "probability",
		},
		Params: []Param{
			{Field: "numdice", Display: "Number of Dice", Type: "uint", Default: "1", Description: "Number of dice to roll"},
			{Field: "sides", Display: "Number of Sides", Type: "[]uint", Default: "[6]", Description: "Number of sides on each dice"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			numDice, err := info.GetUint(m, "numdice")
			if err != nil {
				return nil, err
			}

			sides, err := info.GetUintArray(m, "sides")
			if err != nil {
				return nil, err
			}

			return dice(f, numDice, sides), nil
		},
	})

}
