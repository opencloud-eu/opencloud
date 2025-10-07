package data

// CreditCardInfo contains credit card info
type CreditCardInfo struct {
	Display  string
	Patterns []uint
	Gaps     []uint
	Lengths  []uint
	Code     CreditCardCode
}

// CreditCardCode contains code type and size
type CreditCardCode struct {
	Name string
	Size uint
}

// CreditCardTypes is an array of credit card types
var CreditCardTypes = []string{"visa", "mastercard", "american-express", "diners-club", "discover", "jcb", "unionpay", "maestro", "elo", "hiper", "hipercard"}

// CreditCards contains payment information
var CreditCards = map[string]CreditCardInfo{
	"visa": {
		Display:  "Visa",
		Patterns: []uint{4},
		Gaps:     []uint{4, 8, 12},
		Lengths:  []uint{16},
		Code: CreditCardCode{
			Name: "CVV",
			Size: 3,
		},
	},
	"mastercard": {
		Display: "Mastercard",
		Patterns: []uint{
			51, 55,
			2221, 2229,
			223, 229,
			23, 26,
			270, 271,
			2720,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{16},
		Code: CreditCardCode{
			Name: "CVC",
			Size: 3,
		},
	},
	"american-express": {
		Display:  "American Express",
		Patterns: []uint{34, 37},
		Gaps:     []uint{4, 10},
		Lengths:  []uint{15},
		Code: CreditCardCode{
			Name: "CID",
			Size: 4,
		},
	},
	"diners-club": {
		Display: "Diners Club",
		Patterns: []uint{
			300, 305,
			36, 38, 39,
		},
		Gaps:    []uint{4, 10},
		Lengths: []uint{14, 16, 19},
		Code: CreditCardCode{
			Name: "CVV",
			Size: 3,
		},
	},
	"discover": {
		Display: "Discover",
		Patterns: []uint{
			6011, 644, 649, 65,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{16, 19},
		Code: CreditCardCode{
			Name: "CID",
			Size: 3,
		},
	},
	"jcb": {
		Display: "JCB",
		Patterns: []uint{
			2131, 1800, 3528, 3589,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{16, 17, 18, 19},
		Code: CreditCardCode{
			Name: "CVV",
			Size: 3,
		},
	},
	"unionpay": {
		Display: "UnionPay",
		Patterns: []uint{
			620, 624, 626,
			62100, 62182,
			62184, 62187,
			62185, 62197,
			62200, 62205,
			622010, 622999,
			622018,
			622019, 622999,
			62207, 62209,
			622126, 622925,
			623, 626,
			6270, 6272, 6276,
			627700, 627779,
			627781, 627799,
			6282, 6289,
			6291, 6292,
			810,
			8110, 8131,
			8132, 8151,
			8152, 8163,
			8164, 817,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{14, 15, 16, 17, 18, 19},
		Code: CreditCardCode{
			Name: "CVN",
			Size: 3,
		},
	},
	"maestro": {
		Display: "Maestro",
		Patterns: []uint{
			493698,
			500000, 506698,
			506779, 508999,
			56, 59,
			6, 63, 67,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{12, 13, 14, 15, 16, 17, 18, 19},
		Code: CreditCardCode{
			Name: "CVC",
			Size: 3,
		},
	},
	"elo": {
		Display: "Elo",
		Patterns: []uint{
			401178, 401179,
			438935, 457631,
			457632, 431274,
			451416, 457393,
			504175, 506699,
			506778, 509000,
			509999, 627780,
			636297, 636368,
			650031, 650033,
			650035, 650051,
			650405, 650439,
			650485, 650538,
			650541, 650598,
			650700, 650718,
			650720, 650727,
			650901, 650978,
			651652, 651679,
			655000, 655019,
			655021, 65505,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{16},
		Code: CreditCardCode{
			Name: "CVE",
			Size: 3,
		},
	},
	"mir": {
		Display:  "Mir",
		Patterns: []uint{2200, 2204},
		Gaps:     []uint{4, 8, 12},
		Lengths:  []uint{16, 17, 18, 19},
		Code: CreditCardCode{
			Name: "CVP2",
			Size: 3,
		},
	},
	"hiper": {
		Display: "Hiper",
		Patterns: []uint{
			637095,
			637568,
			637599,
			637609,
			637612,
		},
		Gaps:    []uint{4, 8, 12},
		Lengths: []uint{16},
		Code: CreditCardCode{
			Name: "CVC",
			Size: 3,
		},
	},
	"hipercard": {
		Display:  "Hipercard",
		Patterns: []uint{606282},
		Gaps:     []uint{4, 8, 12},
		Lengths:  []uint{16},
		Code: CreditCardCode{
			Name: "CVC",
			Size: 3,
		},
	},
}
