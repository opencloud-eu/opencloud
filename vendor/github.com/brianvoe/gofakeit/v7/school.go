package gofakeit

// School will generate a random School type
func School() string { return school(GlobalFaker) }

func (f *Faker) School() string { return school(f) }

func school(f *Faker) string {
	return getRandValue(f, []string{"school", "name"}) + " " +
		getRandValue(f, []string{"school", "isPrivate"}) + " " +
		getRandValue(f, []string{"school", "type"})
}

func addSchoolLookup() {
	AddFuncLookup("school", Info{
		Display:     "School",
		Category:    "school",
		Description: "An institution for formal education and learning",
		Example:     `Harborview State Academy`,
		Output:      "string",
		Aliases: []string{
			"academy", "educational institute", "learning center", "training school", "academic institution",
		},
		Keywords: []string{
			"institution", "education", "learning", "teaching", "university", "college", "campus", "classroom", "study", "pupil", "curriculum", "instruction",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return school(f), nil
		},
	})
}
