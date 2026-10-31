package data

// Prefixes for ISBN standards
const (
	ISBN13Prefix = "978"
	ISBN10Prefix = "979"
)

// ISBNRule defines a registrant rule range and its length
type ISBNRule struct {
	Min    string
	Max    string
	Length int
}

// ISBNRules maps prefix -> registration group -> registrant rules
var ISBNRules = map[string]map[string][]ISBNRule{
	ISBN13Prefix: {
		"0": {
			{Min: "0000000", Max: "1999999", Length: 2},
			{Min: "2000000", Max: "2279999", Length: 3},
			{Min: "2280000", Max: "2289999", Length: 4},
			{Min: "2290000", Max: "6479999", Length: 3},
			{Min: "6480000", Max: "6489999", Length: 7},
			{Min: "6490000", Max: "6999999", Length: 3},
			{Min: "7000000", Max: "8499999", Length: 4},
			{Min: "8500000", Max: "8999999", Length: 5},
			{Min: "9000000", Max: "9499999", Length: 6},
			{Min: "9500000", Max: "9999999", Length: 7},
		},
		"1": {
			{Min: "0000000", Max: "0999999", Length: 2},
			{Min: "1000000", Max: "3999999", Length: 3},
			{Min: "4000000", Max: "5499999", Length: 4},
			{Min: "5500000", Max: "7319999", Length: 5},
			{Min: "7320000", Max: "7399999", Length: 7},
			{Min: "7400000", Max: "8697999", Length: 5},
			{Min: "8698000", Max: "9729999", Length: 6},
			{Min: "9730000", Max: "9877999", Length: 4},
			{Min: "9878000", Max: "9989999", Length: 6},
			{Min: "9990000", Max: "9999999", Length: 7},
		},
	},
	ISBN10Prefix: {
		"8": {
			{Min: "0000000", Max: "1999999", Length: 2},
			{Min: "2000000", Max: "2279999", Length: 3},
			{Min: "2280000", Max: "2289999", Length: 4},
			{Min: "2290000", Max: "6479999", Length: 3},
			{Min: "6480000", Max: "6489999", Length: 7},
			{Min: "6490000", Max: "6999999", Length: 3},
			{Min: "7000000", Max: "8499999", Length: 4},
			{Min: "8500000", Max: "8999999", Length: 5},
			{Min: "9000000", Max: "9499999", Length: 6},
			{Min: "9500000", Max: "9999999", Length: 7},
		},
	},
}
