package gofakeit

// Connective will generate a random connective
func Connective() string { return connective(GlobalFaker) }

// Connective will generate a random connective
func (f *Faker) Connective() string { return connective(f) }

func connective(f *Faker) string {
	var connectiveType = map[int]string{
		0: "connective_time",
		1: "connective_comparative",
		2: "connective_complaint",
		3: "connective_listing",
		4: "connective_casual",
		5: "connective_examplify",
	}
	return getRandValue(f, []string{"word", connectiveType[number(f, 0, 5)]})
}

// ConnectiveTime will generate a random connective time
func ConnectiveTime() string { return connectiveTime(GlobalFaker) }

// ConnectiveTime will generate a random connective time

func (f *Faker) ConnectiveTime() string { return connectiveTime(f) }

func connectiveTime(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_time"})
}

// ConnectiveComparative will generate a random comparative connective
func ConnectiveComparative() string { return connectiveComparative(GlobalFaker) }

// ConnectiveComparative will generate a random comparative connective
func (f *Faker) ConnectiveComparative() string { return connectiveComparative(f) }

func connectiveComparative(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_comparative"})
}

// ConnectiveComplaint will generate a random complaint connective
func ConnectiveComplaint() string { return connectiveComplaint(GlobalFaker) }

// ConnectiveComplaint will generate a random complaint connective
func (f *Faker) ConnectiveComplaint() string { return connectiveComplaint(f) }

func connectiveComplaint(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_complaint"})
}

// ConnectiveListing will generate a random listing connective
func ConnectiveListing() string { return connectiveListing(GlobalFaker) }

// ConnectiveListing will generate a random listing connective
func (f *Faker) ConnectiveListing() string { return connectiveListing(f) }

func connectiveListing(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_listing"})
}

// ConnectiveCasual will generate a random casual connective
func ConnectiveCasual() string { return connectiveCasual(GlobalFaker) }

// ConnectiveCasual will generate a random casual connective
func (f *Faker) ConnectiveCasual() string { return connectiveCasual(f) }

func connectiveCasual(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_casual"})
}

// ConnectiveExamplify will generate a random examplify connective
func ConnectiveExamplify() string { return connectiveExamplify(GlobalFaker) }

// ConnectiveExamplify will generate a random examplify connective
func (f *Faker) ConnectiveExamplify() string { return connectiveExamplify(f) }

func connectiveExamplify(f *Faker) string {
	return getRandValue(f, []string{"word", "connective_examplify"})
}

func addWordConnectiveLookup() {
	AddFuncLookup("connective", Info{
		Display:     "Connective",
		Category:    "word",
		Description: "Word used to connect words or sentences",
		Example:     "such as",
		Output:      "string",
		Aliases:     []string{"joining element", "grammar connector", "sentence bridge", "word linker"},
		Keywords:    []string{"connective", "word", "connect", "sentence", "grammar", "used", "conjunction", "link", "joining"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connective(f), nil
		},
	})

	AddFuncLookup("connectivetime", Info{
		Display:     "Connective Time",
		Category:    "word",
		Description: "Connective word used to indicate a temporal relationship between events or actions",
		Example:     "finally",
		Output:      "string",
		Aliases:     []string{"temporal connector", "time relationship", "chronological link", "sequence element"},
		Keywords:    []string{"connective", "time", "temporal", "relationship", "events", "grammar", "actions", "chronological", "sequence", "timing"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveTime(f), nil
		},
	})

	AddFuncLookup("connectivecomparative", Info{
		Display:     "Connective Comparitive",
		Category:    "word",
		Description: "Connective word used to indicate a comparison between two or more things",
		Example:     "in addition",
		Output:      "string",
		Aliases:     []string{"comparison connector", "contrast element", "similarity link", "grammar bridge"},
		Keywords:    []string{"connective", "comparative", "comparison", "things", "grammar", "indicate", "contrast", "similarity", "relative"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveComparative(f), nil
		},
	})

	AddFuncLookup("connectivecomplaint", Info{
		Display:     "Connective Complaint",
		Category:    "word",
		Description: "Connective word used to express dissatisfaction or complaints about a situation",
		Example:     "besides",
		Output:      "string",
		Aliases:     []string{"objection connector", "criticism element", "dissatisfaction link", "grammar bridge"},
		Keywords:    []string{"connective", "complaint", "dissatisfaction", "situation", "grammar", "express", "objection", "criticism", "negative"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveComplaint(f), nil
		},
	})

	AddFuncLookup("connectivelisting", Info{
		Display:     "Connective Listing",
		Category:    "word",
		Description: "Connective word used to list or enumerate items or examples",
		Example:     "firstly",
		Output:      "string",
		Aliases:     []string{"enumeration connector", "sequence element", "order link", "grammar bridge"},
		Keywords:    []string{"connective", "listing", "enumerate", "items", "examples", "grammar", "list", "sequence", "order", "numbered"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveListing(f), nil
		},
	})

	AddFuncLookup("connectivecasual", Info{
		Display:     "Connective Casual",
		Category:    "word",
		Description: "Connective word used to indicate a cause-and-effect relationship between events or actions",
		Example:     "an outcome of",
		Output:      "string",
		Aliases:     []string{"causal connector", "effect relationship", "consequence link", "grammar bridge"},
		Keywords:    []string{"connective", "casual", "cause", "effect", "relationship", "grammar", "events", "actions", "causal", "consequence", "result"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveCasual(f), nil
		},
	})

	AddFuncLookup("connectiveexamplify", Info{
		Display:     "Connective Examplify",
		Category:    "word",
		Description: "Connective word used to provide examples or illustrations of a concept or idea",
		Example:     "then",
		Output:      "string",
		Aliases:     []string{"example connector", "illustration element", "instance link", "grammar bridge"},
		Keywords:    []string{"connective", "examplify", "examples", "illustrations", "concept", "grammar", "provide", "instance", "demonstration", "sample"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return connectiveExamplify(f), nil
		},
	})
}
