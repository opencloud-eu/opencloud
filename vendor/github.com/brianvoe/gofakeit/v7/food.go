package gofakeit

import (
	"strings"
)

// Fruit will return a random fruit name
func Fruit() string { return fruit(GlobalFaker) }

// Fruit will return a random fruit name
func (f *Faker) Fruit() string { return fruit(f) }

func fruit(f *Faker) string { return getRandValue(f, []string{"food", "fruit"}) }

// Vegetable will return a random vegetable name
func Vegetable() string { return vegetable(GlobalFaker) }

// Vegetable will return a random vegetable name
func (f *Faker) Vegetable() string { return vegetable(f) }

func vegetable(f *Faker) string { return getRandValue(f, []string{"food", "vegetable"}) }

// Breakfast will return a random breakfast name
func Breakfast() string { return breakfast(GlobalFaker) }

// Breakfast will return a random breakfast name
func (f *Faker) Breakfast() string { return breakfast(f) }

func breakfast(f *Faker) string {
	v := getRandValue(f, []string{"food", "breakfast"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Lunch will return a random lunch name
func Lunch() string { return lunch(GlobalFaker) }

// Lunch will return a random lunch name
func (f *Faker) Lunch() string { return lunch(f) }

func lunch(f *Faker) string {
	v := getRandValue(f, []string{"food", "lunch"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Dinner will return a random dinner name
func Dinner() string { return dinner(GlobalFaker) }

// Dinner will return a random dinner name
func (f *Faker) Dinner() string { return dinner(f) }

func dinner(f *Faker) string {
	v := getRandValue(f, []string{"food", "dinner"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Drink will return a random drink name
func Drink() string { return drink(GlobalFaker) }

// Drink will return a random drink name
func (f *Faker) Drink() string { return drink(f) }

func drink(f *Faker) string {
	v := getRandValue(f, []string{"food", "drink"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Snack will return a random snack name
func Snack() string { return snack(GlobalFaker) }

// Snack will return a random snack name
func (f *Faker) Snack() string { return snack(f) }

func snack(f *Faker) string {
	v := getRandValue(f, []string{"food", "snack"})
	return strings.ToUpper(v[:1]) + v[1:]
}

// Dessert will return a random dessert name
func Dessert() string { return dessert(GlobalFaker) }

// Dessert will return a random dessert name
func (f *Faker) Dessert() string { return dessert(f) }

func dessert(f *Faker) string {
	v := getRandValue(f, []string{"food", "dessert"})
	return strings.ToUpper(v[:1]) + v[1:]
}

func addFoodLookup() {
	AddFuncLookup("fruit", Info{
		Display:     "Fruit",
		Category:    "food",
		Description: "Edible plant part, typically sweet, enjoyed as a natural snack or dessert",
		Example:     "Peach",
		Output:      "string",
		Aliases: []string{
			"fruit item",
			"natural snack",
			"sweet produce",
			"edible plant food",
			"dessert fruit",
		},
		Keywords: []string{
			"fruit", "edible", "plant", "peach",
			"snack", "dessert", "sweet", "natural",
			"produce", "fresh",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return fruit(f), nil
		},
	})

	AddFuncLookup("vegetable", Info{
		Display:     "Vegetable",
		Category:    "food",
		Description: "Edible plant or part of a plant, often used in savory cooking or salads",
		Example:     "Amaranth Leaves",
		Output:      "string",
		Aliases: []string{
			"veggie",
			"plant food",
			"green produce",
			"savory food",
			"leafy edible",
		},
		Keywords: []string{
			"vegetable", "greens", "produce", "amaranth",
			"leaves", "cooking", "salads", "plant",
			"edible", "savory",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return vegetable(f), nil
		},
	})

	AddFuncLookup("breakfast", Info{
		Display:     "Breakfast",
		Category:    "food",
		Description: "First meal of the day, typically eaten in the morning",
		Example:     "Blueberry banana happy face pancakes",
		Output:      "string",
		Aliases: []string{
			"morning meal",
			"first meal",
			"day starter",
			"early food",
			"sunrise meal",
		},
		Keywords: []string{
			"breakfast", "morning", "meal", "start",
			"pancakes", "blueberry", "banana", "food",
			"first", "early",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return breakfast(f), nil
		},
	})

	AddFuncLookup("lunch", Info{
		Display:     "Lunch",
		Category:    "food",
		Description: "Midday meal, often lighter than dinner, eaten around noon",
		Example:     "No bake hersheys bar pie",
		Output:      "string",
		Aliases: []string{
			"midday meal",
			"noon food",
			"afternoon meal",
			"light meal",
			"daytime meal",
		},
		Keywords: []string{
			"lunch", "meal", "midday", "noon",
			"lighter", "food", "pie", "bar",
			"afternoon",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return lunch(f), nil
		},
	})

	AddFuncLookup("dinner", Info{
		Display:     "Dinner",
		Category:    "food",
		Description: "Evening meal, typically the day's main and most substantial meal",
		Example:     "Wild addicting dip",
		Output:      "string",
		Aliases: []string{
			"evening meal",
			"main meal",
			"days supper",
			"night food",
			"hearty meal",
		},
		Keywords: []string{
			"dinner", "supper", "evening", "meal",
			"main", "substantial", "night", "food",
			"heavy", "course",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return dinner(f), nil
		},
	})

	AddFuncLookup("drink", Info{
		Display:     "Drink",
		Category:    "food",
		Description: "Liquid consumed for hydration, pleasure, or nutritional benefits",
		Example:     "Soda",
		Output:      "string",
		Aliases: []string{
			"beverage",
			"refreshment",
			"hydration",
			"liquid food",
			"consumable fluid",
		},
		Keywords: []string{
			"drink", "soda", "liquid",
			"pleasure", "nutrition", "fluid", "quencher",
			"consumed",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return drink(f), nil
		},
	})

	AddFuncLookup("snack", Info{
		Display:     "Snack",
		Category:    "food",
		Description: "Small, quick food item eaten between meals",
		Example:     "Trail mix",
		Output:      "string",
		Aliases: []string{
			"light bite",
			"quick food",
			"mini meal",
			"finger food",
			"nibble",
		},
		Keywords: []string{
			"snack", "between", "meals", "quick",
			"small", "food", "item", "random",
			"bite", "treat",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return snack(f), nil
		},
	})

	AddFuncLookup("dessert", Info{
		Display:     "Dessert",
		Category:    "food",
		Description: "Sweet treat often enjoyed after a meal",
		Example:     "French napoleons",
		Output:      "string",
		Aliases: []string{
			"after meal sweet",
			"pastry treat",
			"confection",
			"final course",
			"delicacy",
		},
		Keywords: []string{
			"dessert", "sweet", "treat", "meal",
			"after", "pastry", "cake", "enjoyed",
			"final", "sugar",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return dessert(f), nil
		},
	})

}
