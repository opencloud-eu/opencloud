package gofakeit

// PetName will return a random fun pet name
func PetName() string {
	return petName(GlobalFaker)
}

// PetName will return a random fun pet name
func (f *Faker) PetName() string {
	return petName(f)
}

func petName(f *Faker) string {
	return getRandValue(f, []string{"animal", "petname"})
}

// Animal will return a random animal
func Animal() string {
	return animal(GlobalFaker)
}

// Animal will return a random animal
func (f *Faker) Animal() string {
	return animal(f)
}

func animal(f *Faker) string {
	return getRandValue(f, []string{"animal", "animal"})
}

// AnimalType will return a random animal type
func AnimalType() string {
	return animalType(GlobalFaker)
}

// AnimalType will return a random animal type
func (f *Faker) AnimalType() string {
	return animalType(f)
}

func animalType(f *Faker) string {
	return getRandValue(f, []string{"animal", "type"})
}

// FarmAnimal will return a random animal that usually lives on a farm
func FarmAnimal() string {
	return farmAnimal(GlobalFaker)
}

// FarmAnimal will return a random animal that usually lives on a farm
func (f *Faker) FarmAnimal() string {
	return farmAnimal(f)
}

func farmAnimal(f *Faker) string {
	return getRandValue(f, []string{"animal", "farm"})
}

// Cat will return a random cat breed
func Cat() string {
	return cat(GlobalFaker)
}

// Cat will return a random cat breed
func (f *Faker) Cat() string {
	return cat(f)
}

func cat(f *Faker) string {
	return getRandValue(f, []string{"animal", "cat"})
}

// Dog will return a random dog breed
func Dog() string {
	return dog(GlobalFaker)
}

// Dog will return a random dog breed
func (f *Faker) Dog() string {
	return dog(f)
}

func dog(f *Faker) string {
	return getRandValue(f, []string{"animal", "dog"})
}

// Bird will return a random bird species
func Bird() string {
	return bird(GlobalFaker)
}

// Bird will return a random bird species
func (f *Faker) Bird() string {
	return bird(f)
}

func bird(f *Faker) string {
	return getRandValue(f, []string{"animal", "bird"})
}

func addAnimalLookup() {
	AddFuncLookup("petname", Info{
		Display:     "Pet Name",
		Category:    "animal",
		Description: "Affectionate nickname given to a pet",
		Example:     "Ozzy Pawsborne",
		Output:      "string",
		Aliases:     []string{"pet nickname", "animal name", "companion name", "friendly name", "affectionate name"},
		Keywords:    []string{"pet", "name", "nickname", "affectionate", "animal", "companion", "friendly", "cute", "funny", "playful", "loving", "adorable"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return petName(f), nil
		},
	})

	AddFuncLookup("animal", Info{
		Display:     "Animal",
		Category:    "animal",
		Description: "Living creature with the ability to move, eat, and interact with its environment",
		Example:     "elk",
		Output:      "string",
		Aliases:     []string{"wild animal", "living creature", "wildlife species", "animal species", "creature name"},
		Keywords:    []string{"animal", "creature", "living", "move", "eat", "environment", "wildlife", "species", "fauna", "beast", "organism", "vertebrate", "invertebrate"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return animal(f), nil
		},
	})

	AddFuncLookup("animaltype", Info{
		Display:     "Animal Type",
		Category:    "animal",
		Description: "Type of animal, such as mammals, birds, reptiles, etc.",
		Example:     "amphibians",
		Output:      "string",
		Aliases:     []string{"animal classification", "species type", "taxonomic group", "animal category", "biological class"},
		Keywords:    []string{"animal", "type", "mammals", "birds", "reptiles", "amphibians", "classification", "taxonomy", "phylum", "class", "order", "family", "genus", "species"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return animalType(f), nil
		},
	})

	AddFuncLookup("farmanimal", Info{
		Display:     "Farm Animal",
		Category:    "animal",
		Description: "Animal name commonly found on a farm",
		Example:     "Chicken",
		Output:      "string",
		Aliases:     []string{"livestock animal", "barnyard animal", "agricultural animal", "domestic animal", "farm livestock"},
		Keywords:    []string{"farm", "animal", "livestock", "domestic", "agriculture", "commonly", "cattle", "barnyard", "herd", "poultry", "swine", "sheep", "goat", "horse", "pig"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return farmAnimal(f), nil
		},
	})

	AddFuncLookup("cat", Info{
		Display:     "Cat",
		Category:    "animal",
		Description: "Various breeds that define different cats",
		Example:     "Chausie",
		Output:      "string",
		Aliases:     []string{"cat breed", "feline breed", "domestic cat", "pet cat", "kitty breed"},
		Keywords:    []string{"cat", "breed", "feline", "pet", "domestic", "various", "persian", "siamese", "maine", "coon", "tabby", "calico", "tuxedo", "kitten"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return cat(f), nil
		},
	})

	AddFuncLookup("dog", Info{
		Display:     "Dog",
		Category:    "animal",
		Description: "Various breeds that define different dogs",
		Example:     "Norwich Terrier",
		Output:      "string",
		Aliases:     []string{"dog breed", "canine breed", "domestic dog", "pet dog", "fido breed"},
		Keywords:    []string{"dog", "breed", "canine", "pet", "domestic", "various", "labrador", "retriever", "terrier", "shepherd", "bulldog", "poodle", "puppy", "hound"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return dog(f), nil
		},
	})

	AddFuncLookup("bird", Info{
		Display:     "Bird",
		Category:    "animal",
		Description: "Distinct species of birds",
		Example:     "goose",
		Output:      "string",
		Aliases:     []string{"bird species", "avian species", "feathered animal", "winged creature", "bird type"},
		Keywords:    []string{"bird", "species", "avian", "feather", "wing", "distinct", "sparrow", "eagle", "hawk", "owl", "duck", "goose", "parrot", "finch", "robin"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return bird(f), nil
		},
	})
}
