package parser

import "testing"

func TestValidateQueryTemplate(t *testing.T) {
	for template, valid := range map[string]bool{
		"":             true,
		`({{.QUERY}})`: true,
		`({{.QUERY}}) AND (NOT path:"./.space" AND NOT id:{{.SPACE_ROOT_PERSONAL}})`: true,
		`({{.QUERY}}) OR mediatype:document`:                                         true,
		`({{.QUERY}}) AND NOT name:"a(b"`:                                            true,
		`{{.QUERY}}`:                                                                 false,
		`{{.QUERY}} AND NOT mediatype:folder`:                                        false,
		`({{.QUERY}}) OR ({{.QUERY}})`:                                               false,
		`({{.QUERY}}) AND {{.QUERY}}`:                                                false,
		`NOT mediatype:folder`:                                                       false,
		`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_GRANT}}`:                              false,
		`({{.QUERY}}) AND NOT id:{{.SPACE_ROOT_PERSONAL}`:                            false,
		`(({{.QUERY}})`:                                                              false,
		`({{.QUERY}}) AND AND`:                                                       false,
		`({{.QUERY}}) AND`:                                                           false,
	} {
		if err := validateQueryTemplate(template); (err == nil) != valid {
			t.Errorf("validateQueryTemplate(%q) = %v, want valid %v", template, err, valid)
		}
	}
}
