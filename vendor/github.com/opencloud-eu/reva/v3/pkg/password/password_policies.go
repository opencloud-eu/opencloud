package password

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// https://owasp.org/www-community/password-special-characters
var _defaultSpecialCharacters = " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

// Validator describes the interface providing a password Validate method
type Validator interface {
	Validate(str string) error
}

// Policies represents a password validation rules
type Policies struct {
	minCharacters           int
	minLowerCaseCharacters  int
	minUpperCaseCharacters  int
	minDigits               int
	minSpecialCharacters    int
	bannedPasswordsList     map[string]struct{}
	digitsRegexp            *regexp.Regexp
	specialCharactersRegexp *regexp.Regexp
}

// NewPasswordPolicy returns a new NewPasswordPolicy instance
func NewPasswordPolicy(minCharacters, minLowerCaseCharacters, minUpperCaseCharacters, minDigits, minSpecialCharacters int, bannedPasswordsList map[string]struct{}) Validator {
	p := &Policies{
		minCharacters:          minCharacters,
		minLowerCaseCharacters: minLowerCaseCharacters,
		minUpperCaseCharacters: minUpperCaseCharacters,
		minDigits:              minDigits,
		minSpecialCharacters:   minSpecialCharacters,
		bannedPasswordsList:    bannedPasswordsList,
	}

	p.digitsRegexp = regexp.MustCompile("[0-9]")
	p.specialCharactersRegexp = regexp.MustCompile(specialCharactersExp(_defaultSpecialCharacters))
	return p
}

// Validate implements a password validation regarding the policy
func (s Policies) Validate(str string) error {
	var allErr error
	if !utf8.ValidString(str) {
		return fmt.Errorf("the password contains invalid characters")
	}
	err := s.validateBannedList(str)
	if err != nil {
		return err
	}
	err = s.validateCharacters(str)
	if err != nil {
		allErr = errors.Join(allErr, err)
	}
	err = s.validateLowerCase(str)
	if err != nil {
		allErr = errors.Join(allErr, err)
	}
	err = s.validateUpperCase(str)
	if err != nil {
		allErr = errors.Join(allErr, err)
	}
	err = s.validateDigits(str)
	if err != nil {
		allErr = errors.Join(allErr, err)
	}
	err = s.validateSpecialCharacters(str)
	if err != nil {
		allErr = errors.Join(allErr, err)
	}
	if allErr != nil {
		return allErr
	}
	return nil
}

func (s Policies) validateBannedList(str string) error {
	if len(s.bannedPasswordsList) == 0 {
		return nil
	}
	if _, ok := s.bannedPasswordsList[str]; ok {
		return fmt.Errorf("unfortunately, your password is commonly used. please pick a harder-to-guess password for your safety")
	}
	return nil
}

func (s Policies) validateCharacters(str string) error {
	if s.count(str) < s.minCharacters {
		return fmt.Errorf("at least %d characters are required", s.minCharacters)
	}
	return nil
}

func (s Policies) validateLowerCase(str string) error {
	if s.countLowerCaseCharacters(str) < s.minLowerCaseCharacters {
		return fmt.Errorf("at least %d lowercase letters are required", s.minLowerCaseCharacters)
	}
	return nil
}

func (s Policies) validateUpperCase(str string) error {
	if s.countUpperCaseCharacters(str) < s.minUpperCaseCharacters {
		return fmt.Errorf("at least %d uppercase letters are required", s.minUpperCaseCharacters)
	}
	return nil
}

func (s Policies) validateDigits(str string) error {
	if s.countDigits(str) < s.minDigits {
		return fmt.Errorf("at least %d numbers are required", s.minDigits)
	}
	return nil
}

func (s Policies) validateSpecialCharacters(str string) error {
	if s.countSpecialCharacters(str) < s.minSpecialCharacters {
		return fmt.Errorf("at least %d special characters are required %s", s.minSpecialCharacters, _defaultSpecialCharacters)
	}
	return nil
}

func (s Policies) count(str string) int {
	return utf8.RuneCount([]byte(str))
}

func (s Policies) countLowerCaseCharacters(str string) int {
	var count int
	for _, c := range str {
		if strings.ToLower(string(c)) == string(c) && strings.ToUpper(string(c)) != string(c) {
			count++
		}
	}
	return count
}

func (s Policies) countUpperCaseCharacters(str string) int {
	var count int
	for _, c := range str {
		if strings.ToUpper(string(c)) == string(c) && strings.ToLower(string(c)) != string(c) {
			count++
		}
	}
	return count
}

func (s Policies) countDigits(str string) int {
	return len(s.digitsRegexp.FindAllStringIndex(str, -1))
}

func (s Policies) countSpecialCharacters(str string) int {
	if s.specialCharactersRegexp == nil {
		return 0
	}
	res := s.specialCharactersRegexp.FindAllStringIndex(str, -1)
	return len(res)
}

func specialCharactersExp(str string) string {
	// escape the '-' character because it is a not meta-characters but, they are special inside of []
	return "[" + strings.ReplaceAll(regexp.QuoteMeta(str), "-", `\-`) + "]"
}
