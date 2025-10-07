package gofakeit

// Company will generate a random company name string
func Company() string { return company(GlobalFaker) }

// Company will generate a random company name string
func (f *Faker) Company() string { return company(f) }

func company(f *Faker) string { return getRandValue(f, []string{"company", "name"}) }

// CompanySuffix will generate a random company suffix string
func CompanySuffix() string { return companySuffix(GlobalFaker) }

// CompanySuffix will generate a random company suffix string
func (f *Faker) CompanySuffix() string { return companySuffix(f) }

func companySuffix(f *Faker) string { return getRandValue(f, []string{"company", "suffix"}) }

// Blurb will generate a random company blurb string
func Blurb() string { return blurb(GlobalFaker) }

func (f *Faker) Blurb() string { return blurb(f) }

func blurb(f *Faker) string { return getRandValue(f, []string{"company", "blurb"}) }

// BuzzWord will generate a random company buzz word string
func BuzzWord() string { return buzzWord(GlobalFaker) }

// BuzzWord will generate a random company buzz word string
func (f *Faker) BuzzWord() string { return buzzWord(f) }

func buzzWord(f *Faker) string { return getRandValue(f, []string{"company", "buzzwords"}) }

// BS will generate a random company bs string
func BS() string { return bs(GlobalFaker) }

// BS will generate a random company bs string
func (f *Faker) BS() string { return bs(f) }

func bs(f *Faker) string { return getRandValue(f, []string{"company", "bs"}) }

// JobInfo is a struct of job information
type JobInfo struct {
	Company    string `json:"company" xml:"company"`
	Title      string `json:"title" xml:"title"`
	Descriptor string `json:"descriptor" xml:"descriptor"`
	Level      string `json:"level" xml:"level"`
}

// Job will generate a struct with random job information
func Job() *JobInfo { return job(GlobalFaker) }

// Job will generate a struct with random job information
func (f *Faker) Job() *JobInfo { return job(f) }

func job(f *Faker) *JobInfo {
	return &JobInfo{
		Company:    company(f),
		Title:      jobTitle(f),
		Descriptor: jobDescriptor(f),
		Level:      jobLevel(f),
	}
}

// JobTitle will generate a random job title string
func JobTitle() string { return jobTitle(GlobalFaker) }

// JobTitle will generate a random job title string
func (f *Faker) JobTitle() string { return jobTitle(f) }

func jobTitle(f *Faker) string { return getRandValue(f, []string{"job", "title"}) }

// JobDescriptor will generate a random job descriptor string
func JobDescriptor() string { return jobDescriptor(GlobalFaker) }

// JobDescriptor will generate a random job descriptor string
func (f *Faker) JobDescriptor() string { return jobDescriptor(f) }

func jobDescriptor(f *Faker) string { return getRandValue(f, []string{"job", "descriptor"}) }

// JobLevel will generate a random job level string
func JobLevel() string { return jobLevel(GlobalFaker) }

// JobLevel will generate a random job level string
func (f *Faker) JobLevel() string { return jobLevel(f) }

func jobLevel(f *Faker) string { return getRandValue(f, []string{"job", "level"}) }

// Slogan will generate a random company slogan
func Slogan() string { return slogan(GlobalFaker) }

// Slogan will generate a random company slogan
func (f *Faker) Slogan() string { return slogan(f) }

// Slogan will generate a random company slogan
func slogan(f *Faker) string {
	slogan := ""
	var sloganStyle = number(f, 0, 2)
	switch sloganStyle {
	// Noun. Buzzword!
	case 0:
		slogan = getRandValue(f, []string{"company", "blurb"}) + ". " + getRandValue(f, []string{"company", "buzzwords"}) + "!"
	// Buzzword Noun, Buzzword Noun.
	case 1:
		slogan = getRandValue(f, []string{"company", "buzzwords"}) + " " + getRandValue(f, []string{"company", "blurb"}) + ", " + getRandValue(f, []string{"company", "buzzwords"}) + " " + getRandValue(f, []string{"company", "blurb"}) + "."
	// Buzzword bs Noun, Buzzword.
	case 2:
		slogan = getRandValue(f, []string{"company", "buzzwords"}) + " " + getRandValue(f, []string{"company", "bs"}) + " " + getRandValue(f, []string{"company", "blurb"}) + ", " + getRandValue(f, []string{"company", "buzzwords"}) + "."
	}
	return slogan
}

func addCompanyLookup() {
	AddFuncLookup("company", Info{
		Display:     "Company",
		Category:    "company",
		Description: "Designated official name of a business or organization",
		Example:     "Moen, Pagac and Wuckert",
		Output:      "string",
		Aliases: []string{
			"business name",
			"company name",
			"organization name",
			"corporate name",
			"legal entity",
		},
		Keywords: []string{
			"company", "business", "organization", "corporation",
			"enterprise", "firm", "entity", "brand",
			"employer", "vendor", "partner",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return company(f), nil
		},
	})

	AddFuncLookup("companysuffix", Info{
		Display:     "Company Suffix",
		Category:    "company",
		Description: "Suffix at the end of a company name, indicating business structure, like 'Inc.' or 'LLC'",
		Example:     "Inc",
		Output:      "string",
		Aliases: []string{
			"business suffix",
			"legal suffix",
			"company ending",
			"corporate suffix",
			"entity suffix",
		},
		Keywords: []string{
			"suffix", "ending", "company", "business", "entity",
			"inc", "incorporated", "llc", "ltd", "limited",
			"corp", "corporation", "plc", "gmbh", "sarl",
			"legal", "structure", "designation",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return companySuffix(f), nil
		},
	})

	AddFuncLookup("bs", Info{
		Display:     "BS",
		Category:    "company",
		Description: "Random bs company word",
		Example:     "front-end",
		Output:      "string",
		Aliases: []string{
			"business jargon",
			"corporate jargon",
			"marketing buzzword",
			"tech buzzword",
			"consulting speak",
		},
		Keywords: []string{
			"bs", "jargon", "buzzwords", "synergy", "leverage",
			"disrupt", "innovate", "scalable", "agile", "optimize",
			"pipeline", "roadmap", "vision", "strategy", "corporate",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return bs(f), nil
		},
	})

	AddFuncLookup("blurb", Info{
		Display:     "Blurb",
		Category:    "company",
		Description: "Brief description or summary of a company's purpose, products, or services",
		Example:     "word",
		Output:      "string",
		Aliases: []string{
			"company blurb",
			"company summary",
			"company description",
			"short overview",
			"about text",
		},
		Keywords: []string{
			"blurb", "summary", "overview", "description",
			"company", "profile", "about", "intro",
			"purpose", "mission", "vision", "statement",
			"services", "products", "offerings",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return blurb(f), nil
		},
	})

	AddFuncLookup("buzzword", Info{
		Display:     "Buzzword",
		Category:    "company",
		Description: "Trendy or overused term often used in business to sound impressive",
		Example:     "disintermediate",
		Output:      "string",
		Aliases: []string{
			"business buzzword",
			"corporate buzzword",
			"trendy term",
			"catchphrase",
			"marketing phrase",
		},
		Keywords: []string{
			"buzzword", "jargon", "hype", "trend", "phrase",
			"term", "corporate", "management", "marketing",
			"innovation", "paradigm", "disruptive", "visionary",
			"fashionable", "impressive",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return buzzWord(f), nil
		},
	})

	AddFuncLookup("job", Info{
		Display:     "Job",
		Category:    "company",
		Description: "Position or role in employment, involving specific tasks and responsibilities",
		Example: `{
	"company": "ClearHealthCosts",
	"title": "Agent",
	"descriptor": "Future",
	"level": "Tactics"
}`,
		Output:      "map[string]string",
		ContentType: "application/json",
		Aliases: []string{
			"job role",
			"job position",
			"employment role",
			"work role",
			"career role",
			"occupation role",
		},
		Keywords: []string{
			"job", "role", "position", "employment", "work",
			"career", "occupation", "profession", "title",
			"responsibilities", "tasks", "duties",
			"staff", "employee", "hiring", "positioning",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return job(f), nil
		},
	})

	AddFuncLookup("jobtitle", Info{
		Display:     "Job Title",
		Category:    "company",
		Description: "Specific title for a position or role within a company or organization",
		Example:     "Director",
		Output:      "string",
		Aliases: []string{
			"job designation",
			"position title",
			"role title",
			"employment title",
			"official title",
		},
		Keywords: []string{
			"job", "title", "designation", "position", "role",
			"occupation", "profession", "career",
			"company", "organization", "staff", "employee",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return jobTitle(f), nil
		},
	})

	AddFuncLookup("jobdescriptor", Info{
		Display:     "Job Descriptor",
		Category:    "company",
		Description: "Word used to describe the duties, requirements, and nature of a job",
		Example:     "Central",
		Output:      "string",
		Aliases: []string{
			"job modifier",
			"job adjective",
			"role descriptor",
			"title descriptor",
			"position descriptor",
		},
		Keywords: []string{
			"descriptor", "modifier", "adjective", "qualifier",
			"job", "role", "title", "position",
			"label", "term", "descriptive", "characterization",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return jobDescriptor(f), nil
		},
	})

	AddFuncLookup("joblevel", Info{
		Display:     "Job Level",
		Category:    "company",
		Description: "Random job level",
		Example:     "Assurance",
		Output:      "string",
		Aliases: []string{
			"seniority level",
			"career level",
			"position level",
			"role level",
			"job grade",
			"job band",
		},
		Keywords: []string{
			"level", "seniority", "rank", "tier", "grade", "band",
			"entry", "junior", "associate", "mid", "senior",
			"lead", "staff", "principal", "manager", "director",
			"executive",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return jobLevel(f), nil
		},
	})

	AddFuncLookup("slogan", Info{
		Display:     "Slogan",
		Category:    "company",
		Description: "Catchphrase or motto used by a company to represent its brand or values",
		Example:     "Universal seamless Focus, interactive.",
		Output:      "string",
		Aliases: []string{
			"company slogan",
			"brand slogan",
			"brand tagline",
			"company motto",
			"advertising slogan",
		},
		Keywords: []string{
			"slogan", "tagline", "motto", "catchphrase",
			"brand", "company", "marketing", "advertising",
			"identity", "values", "mission", "vision",
			"strapline", "promo", "campaign",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return slogan(f), nil
		},
	})
}
