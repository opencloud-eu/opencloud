package gofakeit

// CelebrityActor will generate a random celebrity actor
func CelebrityActor() string { return celebrityActor(GlobalFaker) }

// CelebrityActor will generate a random celebrity actor
func (f *Faker) CelebrityActor() string { return celebrityActor(f) }

func celebrityActor(f *Faker) string { return getRandValue(f, []string{"celebrity", "actor"}) }

// CelebrityBusiness will generate a random celebrity business person
func CelebrityBusiness() string { return celebrityBusiness(GlobalFaker) }

// CelebrityBusiness will generate a random celebrity business person
func (f *Faker) CelebrityBusiness() string { return celebrityBusiness(f) }

func celebrityBusiness(f *Faker) string {
	return getRandValue(f, []string{"celebrity", "business"})
}

// CelebritySport will generate a random celebrity sport person
func CelebritySport() string { return celebritySport(GlobalFaker) }

// CelebritySport will generate a random celebrity sport person
func (f *Faker) CelebritySport() string { return celebritySport(f) }

func celebritySport(f *Faker) string { return getRandValue(f, []string{"celebrity", "sport"}) }

func addCelebrityLookup() {
	AddFuncLookup("celebrityactor", Info{
		Display:     "Celebrity Actor",
		Category:    "celebrity",
		Description: "Famous person known for acting in films, television, or theater",
		Example:     "Brad Pitt",
		Output:      "string",
		Aliases:     []string{"actor", "famous", "films", "television", "theater", "entertainment"},
		Keywords:    []string{"celebrity", "known", "brad", "pitt", "hollywood", "movie", "star", "performer", "artist"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return celebrityActor(f), nil
		},
	})

	AddFuncLookup("celebritybusiness", Info{
		Display:     "Celebrity Business",
		Category:    "celebrity",
		Description: "High-profile individual known for significant achievements in business or entrepreneurship",
		Example:     "Elon Musk",
		Output:      "string",
		Aliases:     []string{"business", "entrepreneur", "high-profile", "achievements", "executive"},
		Keywords:    []string{"celebrity", "significant", "elon", "musk", "ceo", "founder", "investor", "tycoon", "magnate"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return celebrityBusiness(f), nil
		},
	})

	AddFuncLookup("celebritysport", Info{
		Display:     "Celebrity Sport",
		Category:    "celebrity",
		Description: "Famous athlete known for achievements in a particular sport",
		Example:     "Michael Phelps",
		Output:      "string",
		Aliases:     []string{"athlete", "famous", "achievements", "competition", "athletic", "player"},
		Keywords:    []string{"celebrity", "particular", "michael", "phelps", "olympics", "champion", "medalist", "record", "holder"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return celebritySport(f), nil
		},
	})
}
