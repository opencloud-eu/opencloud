// Copyright 2018-2023 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package cfg

import (
	"errors"
	"reflect"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/go-viper/mapstructure/v2"
)

// Setter is the interface a configuration struct may implement
// to set the default options.
type Setter interface {
	// ApplyDefaults applies the default options.
	ApplyDefaults()
}

var validate = validator.New()
var english = en.New()
var uni = ut.New(english, english)
var trans, _ = uni.GetTranslator("en")
var _ = en_translations.RegisterDefaultTranslations(validate, trans)

func init() {
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		if k := field.Tag.Get("mapstructure"); k != "" {
			return k
		}
		// if not specified, fall back to field name
		return field.Name
	})
}

// Decode decodes the given raw input interface to the target pointer c.
// It applies the default configuration if the target struct
// implements the Setter interface.
// It also perform a validation to all the fields of the configuration.
func Decode(input map[string]any, c any) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   c,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	if err := decoder.Decode(input); err != nil {
		return err
	}
	if s, ok := c.(Setter); ok {
		s.ApplyDefaults()
	}

	return translateError(validate.Struct(c), trans)
}

func translateError(err error, trans ut.Translator) error {
	if err == nil {
		return nil
	}
	errs := err.(validator.ValidationErrors)
	translated := make([]error, 0, len(errs))
	for _, err := range errs {
		translated = append(translated, errors.New(err.Translate(trans)))
	}
	return errors.Join(translated...)
}
