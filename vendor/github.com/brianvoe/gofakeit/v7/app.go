package gofakeit

import (
	"fmt"
)

// AppName will generate a random app name
func AppName() string {
	return appName(GlobalFaker)
}

// AppName will generate a random app name
func (f *Faker) AppName() string {
	return appName(f)
}

func appName(f *Faker) string {
	name := ""
	switch number(f, 1, 3) {
	case 1:
		name = noun(f) + verb(f)
	case 2:
		name = color(f) + noun(f)
	case 3:
		name = animal(f) + verb(f)
	}

	return title(name)
}

// AppVersion will generate a random app version
func AppVersion() string {
	return appVersion(GlobalFaker)
}

// AppVersion will generate a random app version
func (f *Faker) AppVersion() string {
	return appVersion(f)
}

func appVersion(f *Faker) string {
	return fmt.Sprintf("%d.%d.%d", number(f, 1, 5), number(f, 1, 20), number(f, 1, 20))
}

// AppAuthor will generate a random company or person name
func AppAuthor() string {
	return appAuthor(GlobalFaker)
}

// AppAuthor will generate a random company or person name
func (f *Faker) AppAuthor() string {
	return appAuthor(f)
}

func appAuthor(f *Faker) string {
	if boolFunc(f) {
		return name(f)
	}

	return company(f)
}

func addAppLookup() {
	AddFuncLookup("appname", Info{
		Display:     "App Name",
		Category:    "app",
		Description: "Software program designed for a specific purpose or task on a computer or mobile device",
		Example:     "Parkrespond",
		Output:      "string",
		Aliases:     []string{"software name", "application name", "mobile app name", "program title", "app title"},
		Keywords:    []string{"app", "name", "software", "program", "application", "mobile", "device", "computer", "ios", "android", "desktop", "web", "platform", "title"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return appName(f), nil
		},
	})

	AddFuncLookup("appversion", Info{
		Display:     "App Version",
		Category:    "app",
		Description: "Particular release of an application in Semantic Versioning format",
		Example:     "1.12.14",
		Output:      "string",
		Aliases:     []string{"semantic version", "app release", "software version", "version number", "release version"},
		Keywords:    []string{"app", "version", "release", "semantic", "versioning", "application", "major", "minor", "patch", "build", "number", "format", "tag"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return appVersion(f), nil
		},
	})

	AddFuncLookup("appauthor", Info{
		Display:     "App Author",
		Category:    "app",
		Description: "Person or group creating and developing an application",
		Example:     "Qado Energy, Inc.",
		Output:      "string",
		Aliases:     []string{"app developer", "software author", "application creator", "program developer", "app creator"},
		Keywords:    []string{"app", "author", "developer", "creator", "person", "company", "group", "creating", "programmer", "coder", "engineer", "team", "organization", "studio", "publisher"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return appAuthor(f), nil
		},
	})
}
