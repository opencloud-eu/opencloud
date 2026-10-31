package gofakeit

// Username will generate a random username based upon picking a random lastname and random numbers at the end
func Username() string {
	return username(GlobalFaker)
}

// Username will generate a random username based upon picking a random lastname and random numbers at the end
func (f *Faker) Username() string {
	return username(f)
}

func username(f *Faker) string {
	return getRandValue(f, []string{"person", "last"}) + replaceWithNumbers(f, "####")
}

// Password will generate a random password.
// Minimum number length of 5 if less than.
func Password(lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	return password(GlobalFaker, lower, upper, numeric, special, space, num)
}

// Password will generate a random password.
// Minimum number length of 5 if less than.
func (f *Faker) Password(lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	return password(f, lower, upper, numeric, special, space, num)
}

func password(f *Faker, lower bool, upper bool, numeric bool, special bool, space bool, num int) string {
	// Make sure the num minimum is at least 5
	if num < 5 {
		num = 5
	}

	// Setup weights
	items := make([]any, 0)
	weights := make([]float32, 0)
	if lower {
		items = append(items, "l")
		weights = append(weights, 4)
	}
	if upper {
		items = append(items, "u")
		weights = append(weights, 4)
	}
	if numeric {
		items = append(items, "n")
		weights = append(weights, 3)
	}
	if special {
		items = append(items, "e")
		weights = append(weights, 2)
	}
	if space {
		items = append(items, "a")
		weights = append(weights, 1)
	}

	// If no items are selected then default to lower, upper, numeric
	if len(items) == 0 {
		items = append(items, "l", "u", "n")
		weights = append(weights, 4, 4, 3)
	}

	// Create byte slice
	b := make([]byte, num)

	for i := 0; i <= num-1; i++ {
		// Run weighted
		weight, _ := weighted(f, items, weights)

		switch weight.(string) {
		case "l":
			b[i] = lowerStr[f.Int64()%int64(len(lowerStr))]
		case "u":
			b[i] = upperStr[f.Int64()%int64(len(upperStr))]
		case "n":
			b[i] = numericStr[f.Int64()%int64(len(numericStr))]
		case "e":
			b[i] = specialSafeStr[f.Int64()%int64(len(specialSafeStr))]
		case "a":
			b[i] = spaceStr[f.Int64()%int64(len(spaceStr))]
		}
	}

	// Shuffle bytes
	for i := range b {
		j := f.IntN(i + 1)
		b[i], b[j] = b[j], b[i]
	}

	// Replace first or last character if it's a space, and other options are available
	if b[0] == ' ' {
		b[0] = password(f, lower, upper, numeric, special, false, 1)[0]
	}
	if b[len(b)-1] == ' ' {
		b[len(b)-1] = password(f, lower, upper, numeric, special, false, 1)[0]
	}

	return string(b)
}

func addAuthLookup() {
	AddFuncLookup("username", Info{
		Display:     "Username",
		Category:    "auth",
		Description: "Unique identifier assigned to a user for accessing an account or system",
		Example:     "Daniel1364",
		Output:      "string",
		Aliases: []string{
			"user name",
			"login name",
			"account username",
			"account login",
			"screen name",
			"user handle",
		},
		Keywords: []string{
			"username", "login", "handle", "userid", "screenname",
			"user", "account", "credential", "signin", "alias", "profile", "uid",
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			return username(f), nil
		},
	})

	AddFuncLookup("password", Info{
		Display:     "Password",
		Category:    "auth",
		Description: "Secret word or phrase used to authenticate access to a system or account",
		Example:     "EEP+wwpk 4lU-eHNXlJZ4n K9%v&TZ9e",
		Output:      "string",
		Aliases: []string{
			"user password",
			"account password",
			"login password",
			"secret phrase",
			"auth secret",
		},
		Keywords: []string{
			"password", "passphrase", "pwd", "secret",
			"credential", "authentication", "auth",
			"security", "signin", "login",
			"access", "key", "token", "hash", "encryption",
		},
		Params: []Param{
			{Field: "lower", Display: "Lower", Type: "bool", Default: "true", Description: "Whether or not to add lower case characters"},
			{Field: "upper", Display: "Upper", Type: "bool", Default: "true", Description: "Whether or not to add upper case characters"},
			{Field: "numeric", Display: "Numeric", Type: "bool", Default: "true", Description: "Whether or not to add numeric characters"},
			{Field: "special", Display: "Special", Type: "bool", Default: "true", Description: "Whether or not to add special characters"},
			{Field: "space", Display: "Space", Type: "bool", Default: "false", Description: "Whether or not to add spaces"},
			{Field: "length", Display: "Length", Type: "int", Default: "12", Description: "Number of characters in password"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			lower, err := info.GetBool(m, "lower")
			if err != nil {
				return nil, err
			}

			upper, err := info.GetBool(m, "upper")
			if err != nil {
				return nil, err
			}

			numeric, err := info.GetBool(m, "numeric")
			if err != nil {
				return nil, err
			}

			special, err := info.GetBool(m, "special")
			if err != nil {
				return nil, err
			}

			space, err := info.GetBool(m, "space")
			if err != nil {
				return nil, err
			}

			length, err := info.GetInt(m, "length")
			if err != nil {
				return nil, err
			}

			return password(f, lower, upper, numeric, special, space, length), nil
		},
	})
}
