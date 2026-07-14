package gofakeit

import (
	"errors"
	"strings"
)

// AddressInfo is a struct full of address information
type AddressInfo struct {
	Address   string  `json:"address" xml:"address"`
	Street    string  `json:"street" xml:"street"`
	Unit      string  `json:"unit" xml:"unit"`
	City      string  `json:"city" xml:"city"`
	State     string  `json:"state" xml:"state"`
	Zip       string  `json:"zip" xml:"zip"`
	Country   string  `json:"country" xml:"country"`
	Latitude  float64 `json:"latitude" xml:"latitude"`
	Longitude float64 `json:"longitude" xml:"longitude"`
}

// Address will generate a struct of address information
func Address() *AddressInfo { return address(GlobalFaker) }

// Address will generate a struct of address information
func (f *Faker) Address() *AddressInfo { return address(f) }

func address(f *Faker) *AddressInfo {
	street := street(f)
	city := city(f)
	state := state(f)
	zip := zip(f)

	// 30% chance to include a unit in the address
	var unitStr string
	var unitField string
	if randIntRange(f, 1, 10) <= 3 {
		unitStr = ", " + unit(f)
		unitField = unit(f)
	}

	addressStr := street + unitStr + ", " + city + ", " + state + " " + zip

	return &AddressInfo{
		Address:   addressStr,
		Street:    street,
		Unit:      unitField,
		City:      city,
		State:     state,
		Zip:       zip,
		Country:   country(f),
		Latitude:  latitude(f),
		Longitude: longitude(f),
	}
}

// Street will generate a random address street string
func Street() string { return street(GlobalFaker) }

// Street will generate a random address street string
func (f *Faker) Street() string { return street(f) }

func street(f *Faker) string {
	var street = ""
	switch randInt := randIntRange(f, 1, 2); randInt {
	case 1:
		street = streetNumber(f) + " " + streetPrefix(f) + " " + streetName(f) + streetSuffix(f)
	case 2:
		street = streetNumber(f) + " " + streetName(f) + streetSuffix(f)
	}

	return street
}

// StreetNumber will generate a random address street number string
func StreetNumber() string { return streetNumber(GlobalFaker) }

// StreetNumber will generate a random address street number string
func (f *Faker) StreetNumber() string { return streetNumber(f) }

func streetNumber(f *Faker) string {
	return strings.TrimLeft(replaceWithNumbers(f, getRandValue(f, []string{"address", "number"})), "0")
}

// StreetPrefix will generate a random address street prefix string
func StreetPrefix() string { return streetPrefix(GlobalFaker) }

// StreetPrefix will generate a random address street prefix string
func (f *Faker) StreetPrefix() string { return streetPrefix(f) }

func streetPrefix(f *Faker) string { return getRandValue(f, []string{"address", "street_prefix"}) }

// StreetName will generate a random address street name string
func StreetName() string { return streetName(GlobalFaker) }

// StreetName will generate a random address street name string
func (f *Faker) StreetName() string { return streetName(f) }

func streetName(f *Faker) string { return getRandValue(f, []string{"address", "street_name"}) }

// StreetSuffix will generate a random address street suffix string
func StreetSuffix() string { return streetSuffix(GlobalFaker) }

// StreetSuffix will generate a random address street suffix string
func (f *Faker) StreetSuffix() string { return streetSuffix(f) }

func streetSuffix(f *Faker) string { return getRandValue(f, []string{"address", "street_suffix"}) }

// Unit will generate a random unit string
func Unit() string { return unit(GlobalFaker) }

// Unit will generate a random unit string
func (f *Faker) Unit() string { return unit(f) }

func unit(f *Faker) string {
	unitType := getRandValue(f, []string{"address", "unit"})
	unitNumber := replaceWithNumbers(f, "###")
	return unitType + " " + unitNumber
}

// City will generate a random city string
func City() string { return city(GlobalFaker) }

// City will generate a random city string
func (f *Faker) City() string { return city(f) }

func city(f *Faker) string { return getRandValue(f, []string{"address", "city"}) }

// State will generate a random state string
func State() string { return state(GlobalFaker) }

// State will generate a random state string
func (f *Faker) State() string { return state(f) }

func state(f *Faker) string { return getRandValue(f, []string{"address", "state"}) }

// StateAbr will generate a random abbreviated state string
func StateAbr() string { return stateAbr(GlobalFaker) }

// StateAbr will generate a random abbreviated state string
func (f *Faker) StateAbr() string { return stateAbr(f) }

func stateAbr(f *Faker) string { return getRandValue(f, []string{"address", "state_abr"}) }

// Zip will generate a random Zip code string
func Zip() string { return zip(GlobalFaker) }

// Zip will generate a random Zip code string
func (f *Faker) Zip() string { return zip(f) }

func zip(f *Faker) string {
	return replaceWithNumbers(f, getRandValue(f, []string{"address", "zip"}))
}

// Country will generate a random country string
func Country() string { return country(GlobalFaker) }

// Country will generate a random country string
func (f *Faker) Country() string { return country(f) }

func country(f *Faker) string { return getRandValue(f, []string{"address", "country"}) }

// CountryAbr will generate a random abbreviated country string
func CountryAbr() string { return countryAbr(GlobalFaker) }

// CountryAbr will generate a random abbreviated country string
func (f *Faker) CountryAbr() string { return countryAbr(f) }

func countryAbr(f *Faker) string { return getRandValue(f, []string{"address", "country_abr"}) }

// Latitude will generate a random latitude float64
func Latitude() float64 { return latitude(GlobalFaker) }

// Latitude will generate a random latitude float64
func (f *Faker) Latitude() float64 { return latitude(f) }

func latitude(f *Faker) float64 { return toFixed((f.Float64()*180)-90, 6) }

// LatitudeInRange will generate a random latitude within the input range
func LatitudeInRange(min, max float64) (float64, error) {
	return latitudeInRange(GlobalFaker, min, max)
}

// LatitudeInRange will generate a random latitude within the input range
func (f *Faker) LatitudeInRange(min, max float64) (float64, error) {
	return latitudeInRange(f, min, max)
}

func latitudeInRange(f *Faker, min, max float64) (float64, error) {
	if min > max || min < -90 || min > 90 || max < -90 || max > 90 {
		return 0, errors.New("invalid min or max range, must be valid floats and between -90 and 90")
	}
	return toFixed(float64Range(f, min, max), 6), nil
}

// Longitude will generate a random longitude float64
func Longitude() float64 { return longitude(GlobalFaker) }

// Longitude will generate a random longitude float64
func (f *Faker) Longitude() float64 { return longitude(f) }

func longitude(f *Faker) float64 { return toFixed((f.Float64()*360)-180, 6) }

// LongitudeInRange will generate a random longitude within the input range
func LongitudeInRange(min, max float64) (float64, error) {
	return longitudeInRange(GlobalFaker, min, max)
}

// LongitudeInRange will generate a random longitude within the input range
func (f *Faker) LongitudeInRange(min, max float64) (float64, error) {
	return longitudeInRange(f, min, max)
}

func longitudeInRange(f *Faker, min, max float64) (float64, error) {
	if min > max || min < -180 || min > 180 || max < -180 || max > 180 {
		return 0, errors.New("invalid min or max range, must be valid floats and between -180 and 180")
	}
	return toFixed(float64Range(f, min, max), 6), nil
}

func addAddressLookup() {
	AddFuncLookup("address", Info{
		Display:     "Address",
		Category:    "address",
		Description: "Residential location including street, city, state, country and postal code",
		Example: `{
	"address": "364 Unionsville, Apt 123, Norfolk, Ohio 99536",
	"street": "364 Unionsville",
	"apartment": "Apt 123",
	"city": "Norfolk",
	"state": "Ohio",
	"zip": "99536",
	"country": "Lesotho",
	"latitude": 88.792592,
	"longitude": 174.504681
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Aliases:     []string{"full address", "residential address", "mailing address", "street address", "home address"},
		Keywords:    []string{"address", "residential", "location", "street", "city", "state", "country", "postal", "code", "mailing", "home", "house", "apartment", "zipcode", "coordinates"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return address(f), nil
		},
	})

	AddFuncLookup("city", Info{
		Display:     "City",
		Category:    "address",
		Description: "Part of a country with significant population, often a central hub for culture and commerce",
		Example:     "Marcelside",
		Output:      "string",
		Aliases:     []string{"city name", "urban area", "municipality name", "town name", "metropolitan area"},
		Keywords:    []string{"city", "town", "municipality", "urban", "area", "population", "hub", "culture", "commerce", "metropolitan", "settlement", "community", "district"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return city(f), nil
		},
	})

	AddFuncLookup("country", Info{
		Display:     "Country",
		Category:    "address",
		Description: "Nation with its own government and defined territory",
		Example:     "United States of America",
		Output:      "string",
		Aliases:     []string{"country name", "nation name", "sovereign state", "national territory", "independent country"},
		Keywords:    []string{"country", "nation", "government", "territory", "sovereign", "independent", "state", "republic", "kingdom", "empire", "federation", "commonwealth"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return country(f), nil
		},
	})

	AddFuncLookup("countryabr", Info{
		Display:     "Country Abbreviation",
		Category:    "address",
		Description: "Shortened 2-letter form of a country's name",
		Example:     "US",
		Output:      "string",
		Aliases:     []string{"country code", "iso alpha-2", "iso3166-1 alpha-2", "two-letter country", "country short code"},
		Keywords:    []string{"country", "abbreviation", "shortened", "2-letter", "nation", "iso", "code", "alpha-2", "iso3166-1", "standard", "international", "identifier"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return countryAbr(f), nil
		},
	})

	AddFuncLookup("state", Info{
		Display:     "State",
		Category:    "address",
		Description: "Governmental division within a country, often having its own laws and government",
		Example:     "Illinois",
		Output:      "string",
		Aliases:     []string{"state name", "province name", "region name", "administrative division", "territory name"},
		Keywords:    []string{"state", "province", "region", "division", "governmental", "territory", "area", "laws", "government", "administrative", "subdivision", "district", "county"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return state(f), nil
		},
	})

	AddFuncLookup("stateabr", Info{
		Display:     "State Abbreviation",
		Category:    "address",
		Description: "Shortened 2-letter form of a state or province",
		Example:     "IL",
		Output:      "string",
		Aliases:     []string{"state code", "province code", "region code", "usps code", "iso3166-2 code"},
		Keywords:    []string{"state", "abbreviation", "shortened", "2-letter", "region", "province", "country", "code", "usps", "iso3166-2", "identifier"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return stateAbr(f), nil
		},
	})

	AddFuncLookup("street", Info{
		Display:     "Street",
		Category:    "address",
		Description: "Public road in a city or town, typically with houses and buildings on each side",
		Example:     "364 East Rapidsborough",
		Output:      "string",
		Aliases:     []string{"street address", "shipping address", "billing address", "mailing address", "address line 1", "line 1", "road address", "avenue address", "drive address", "thoroughfare address"},
		Keywords:    []string{"address", "street", "road", "avenue", "drive", "lane", "way", "public", "thoroughfare", "boulevard", "court", "place", "circle", "terrace", "highway"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return street(f), nil
		},
	})

	AddFuncLookup("streetname", Info{
		Display:     "Street Name",
		Category:    "address",
		Description: "Name given to a specific road or street",
		Example:     "View",
		Output:      "string",
		Aliases:     []string{"street title", "road name", "avenue name", "drive name", "thoroughfare name"},
		Keywords:    []string{"street", "name", "road", "avenue", "drive", "lane", "way", "thoroughfare", "specific", "title", "designation", "label", "identifier"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return streetName(f), nil
		},
	})

	AddFuncLookup("streetnumber", Info{
		Display:     "Street Number",
		Category:    "address",
		Description: "Numerical identifier assigned to a street",
		Example:     "13645",
		Output:      "string",
		Aliases:     []string{"house number", "building number", "address number", "street identifier", "numerical address"},
		Keywords:    []string{"street", "number", "identifier", "numerical", "address", "location", "building", "assigned", "house", "digit", "numeric", "sequence", "position"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return streetNumber(f), nil
		},
	})

	AddFuncLookup("streetprefix", Info{
		Display:     "Street Prefix",
		Category:    "address",
		Description: "Directional or descriptive term preceding a street name (e.g., 'East', 'N')",
		Example:     "East",
		Output:      "string",
		Aliases:     []string{"directional prefix", "street prefix", "name prefix", "road prefix", "thoroughfare prefix"},
		Keywords:    []string{"street", "prefix", "directional", "north", "south", "east", "west", "n", "s", "e", "w"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return streetPrefix(f), nil
		},
	})

	AddFuncLookup("streetsuffix", Info{
		Display:     "Street Suffix",
		Category:    "address",
		Description: "Designation at the end of a street name indicating type (e.g., 'Ave', 'St')",
		Example:     "Ave",
		Output:      "string",
		Aliases:     []string{"street type", "road type", "avenue suffix", "thoroughfare suffix", "street ending"},
		Keywords:    []string{"street", "suffix", "designation", "type", "ave", "st", "rd", "dr", "ln", "blvd", "ct", "pl", "cir", "ter", "hwy"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return streetSuffix(f), nil
		},
	})

	AddFuncLookup("unit", Info{
		Display:     "Unit",
		Category:    "address",
		Description: "Unit identifier within a building, such as apartment number, suite, or office",
		Example:     "Apt 123",
		Output:      "string",
		Aliases:     []string{"apartment unit", "suite number", "office number", "building unit", "room number", "address line 2", "line 2"},
		Keywords:    []string{"unit", "apartment", "suite", "office", "identifier", "building", "number", "within", "room", "floor", "level", "section", "compartment"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return unit(f), nil
		},
	})

	AddFuncLookup("zip", Info{
		Display:     "Zip",
		Category:    "address",
		Description: "Numerical code for postal address sorting, specific to a geographic area",
		Example:     "13645",
		Output:      "string",
		Aliases:     []string{"zip code", "postal code", "mail code", "delivery code"},
		Keywords:    []string{"zip", "postal", "postcode", "code", "address", "sorting", "geographic", "area", "numerical", "mailing", "delivery", "zone", "district", "region", "identifier"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return zip(f), nil
		},
	})

	AddFuncLookup("latitude", Info{
		Display:     "Latitude",
		Category:    "address",
		Description: "Geographic coordinate specifying north-south position on Earth's surface",
		Example:     "-73.534056",
		Output:      "float",
		Aliases:     []string{"lat coordinate", "north-south coordinate", "geographic latitude", "earth latitude", "position latitude"},
		Keywords:    []string{"latitude", "lat", "coordinate", "north-south", "degrees", "gps", "wgs84", "geodesy", "parallel", "equator", "pole"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return latitude(f), nil
		},
	})

	AddFuncLookup("latituderange", Info{
		Display:     "Latitude Range",
		Category:    "address",
		Description: "Latitude number between the given range (default min=0, max=90)",
		Example:     "22.921026",
		Output:      "float",
		Aliases:     []string{"latitude bounds", "lat range", "north-south range", "geographic bounds", "coordinate range"},
		Keywords:    []string{"latitude", "lat", "range", "min", "max", "degrees", "gps", "wgs84", "bounds", "interval"},
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Default: "0", Description: "Minimum range"},
			{Field: "max", Display: "Max", Type: "float", Default: "90", Description: "Maximum range"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}
			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}
			return latitudeInRange(f, min, max)
		},
	})

	AddFuncLookup("longitude", Info{
		Display:     "Longitude",
		Category:    "address",
		Description: "Geographic coordinate indicating east-west position on Earth's surface",
		Example:     "-147.068112",
		Output:      "float",
		Aliases:     []string{"long coordinate", "east-west coordinate", "geographic longitude", "earth longitude", "position longitude"},
		Keywords:    []string{"longitude", "lon", "coordinate", "east-west", "degrees", "gps", "wgs84", "geodesy", "meridian", "idl"},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return longitude(f), nil
		},
	})

	AddFuncLookup("longituderange", Info{
		Display:     "Longitude Range",
		Category:    "address",
		Description: "Longitude number between the given range (default min=0, max=180)",
		Example:     "-8.170450",
		Output:      "float",
		Aliases:     []string{"longitude bounds", "long range", "east-west range", "geographic bounds", "coordinate range"},
		Keywords:    []string{"longitude", "lon", "range", "min", "max", "degrees", "gps", "wgs84", "bounds", "interval"},
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Default: "0", Description: "Minimum range"},
			{Field: "max", Display: "Max", Type: "float", Default: "180", Description: "Maximum range"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}
			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}
			return longitudeInRange(f, min, max)
		},
	})

}
