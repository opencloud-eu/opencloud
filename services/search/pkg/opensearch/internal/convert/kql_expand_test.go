package convert_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/convert"

	"github.com/opencloud-eu/opencloud/pkg/ast"
	"github.com/opencloud-eu/opencloud/services/search/pkg/opensearch/internal/test"
)

func TestExpandKQLAST(t *testing.T) {
	t.Run("always converts a value node to a pointer node", func(t *testing.T) {
		tests := []opensearchtest.TableTest[[]ast.Node, []ast.Node]{
			{
				Name: "ast.node.V -> ast.node.PTR",
				Got: []ast.Node{
					&ast.StringNode{Key: "a"},
					&ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "b"},
					ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{Key: "c"},
					&ast.OperatorNode{Value: "OR"},
					ast.DateTimeNode{Key: "d"},
					ast.OperatorNode{Value: "OR"},
					&ast.BooleanNode{Key: "f"},
					&ast.OperatorNode{Value: "NOT"},
					ast.BooleanNode{Key: "g"},
					ast.OperatorNode{Value: "NOT"},
					&ast.GroupNode{Key: "h", Nodes: []ast.Node{
						&ast.StringNode{Key: "a"},
						&ast.OperatorNode{Value: "AND"},
						ast.StringNode{Key: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.GroupNode{Key: "h", Nodes: []ast.Node{
							&ast.StringNode{Key: "a"},
							&ast.OperatorNode{Value: "AND"},
							ast.StringNode{Key: "b"},
							&ast.OperatorNode{Value: "OR"},
							&ast.GroupNode{Key: "h", Nodes: []ast.Node{
								&ast.StringNode{Key: "a"},
								&ast.OperatorNode{Value: "AND"},
								ast.StringNode{Key: "b"},
							}},
						}},
					}},
					ast.GroupNode{Key: "i", Nodes: []ast.Node{
						ast.StringNode{Key: "a"},
						ast.OperatorNode{Value: "AND"},
						ast.StringNode{Key: "b"},
						ast.OperatorNode{Value: "OR"},
						ast.GroupNode{Key: "h", Nodes: []ast.Node{
							ast.StringNode{Key: "a"},
							ast.OperatorNode{Value: "AND"},
							ast.StringNode{Key: "b"},
							ast.OperatorNode{Value: "OR"},
							ast.GroupNode{Key: "h", Nodes: []ast.Node{
								ast.StringNode{Key: "a"},
								ast.OperatorNode{Value: "AND"},
								ast.StringNode{Key: "b"},
							}},
						}},
					}},
				},
				Want: []ast.Node{
					&ast.StringNode{Key: "a"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "b"},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{Key: "c"},
					&ast.OperatorNode{Value: "OR"},
					&ast.DateTimeNode{Key: "d"},
					&ast.OperatorNode{Value: "OR"},
					&ast.BooleanNode{Key: "f"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.BooleanNode{Key: "g"},
					&ast.OperatorNode{Value: "NOT"},
					&ast.GroupNode{Key: "h", Nodes: []ast.Node{
						&ast.StringNode{Key: "a"},
						&ast.OperatorNode{Value: "AND"},
						&ast.StringNode{Key: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.GroupNode{Key: "h", Nodes: []ast.Node{
							&ast.StringNode{Key: "a"},
							&ast.OperatorNode{Value: "AND"},
							&ast.StringNode{Key: "b"},
							&ast.OperatorNode{Value: "OR"},
							&ast.GroupNode{Key: "h", Nodes: []ast.Node{
								&ast.StringNode{Key: "a"},
								&ast.OperatorNode{Value: "AND"},
								&ast.StringNode{Key: "b"},
							}},
						}},
					}},
					&ast.GroupNode{Key: "i", Nodes: []ast.Node{
						&ast.StringNode{Key: "a"},
						&ast.OperatorNode{Value: "AND"},
						&ast.StringNode{Key: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.GroupNode{Key: "h", Nodes: []ast.Node{
							&ast.StringNode{Key: "a"},
							&ast.OperatorNode{Value: "AND"},
							&ast.StringNode{Key: "b"},
							&ast.OperatorNode{Value: "OR"},
							&ast.GroupNode{Key: "h", Nodes: []ast.Node{
								&ast.StringNode{Key: "a"},
								&ast.OperatorNode{Value: "AND"},
								&ast.StringNode{Key: "b"},
							}},
						}},
					}},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				result, err := convert.ExpandKQL(test.Got)
				require.NoError(t, err)
				require.Equal(t, test.Want, result)
			})
		}
	})

	t.Run("remaps some keys", func(t *testing.T) {
		var tests []opensearchtest.TableTest[[]ast.Node, []ast.Node]

		for k, v := range map[string]string{
			"":          "Name", // Default to "Name" if no key is provided
			"rootid":    "RootID",
			"path":      "Path",
			"id":        "ID",
			"name":      "Name",
			"size":      "Size",
			"mtime":     "Mtime",
			"mediatype": "MimeType",
			"type":      "Type",
			"tag":       "Tags",
			"tags":      "Tags",
			"content":   "Content",
			"hidden":    "Hidden",
			"any":       "any", // Example of an unknown key that should remain unchanged
		} {
			tests = append(tests, opensearchtest.TableTest[[]ast.Node, []ast.Node]{
				Name: fmt.Sprintf("%s -> %s", k, v),
				Got: []ast.Node{
					&ast.StringNode{Key: k},
					&ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: k},
					ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{Key: k},
					&ast.OperatorNode{Value: "OR"},
					ast.DateTimeNode{Key: k},
					ast.OperatorNode{Value: "OR"},
					&ast.BooleanNode{Key: k},
					&ast.OperatorNode{Value: "NOT"},
					ast.BooleanNode{Key: k},
					ast.OperatorNode{Value: "NOT"},
					&ast.GroupNode{Key: k, Nodes: []ast.Node{
						&ast.StringNode{Key: k},
						&ast.OperatorNode{Value: "AND"},
						ast.StringNode{Key: k},
						&ast.OperatorNode{Value: "OR"},
						&ast.GroupNode{Key: k, Nodes: []ast.Node{
							&ast.StringNode{Key: k},
							&ast.OperatorNode{Value: "AND"},
							ast.StringNode{Key: k},
							&ast.OperatorNode{Value: "OR"},
							&ast.GroupNode{Key: k, Nodes: []ast.Node{
								&ast.StringNode{Key: k},
								&ast.OperatorNode{Value: "AND"},
								ast.StringNode{Key: k},
							}},
						}},
					}},
				},
				Want: []ast.Node{
					&ast.StringNode{Key: v},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: v},
					&ast.OperatorNode{Value: "AND"},
					&ast.DateTimeNode{Key: v},
					&ast.OperatorNode{Value: "OR"},
					&ast.DateTimeNode{Key: v},
					&ast.OperatorNode{Value: "OR"},
					&ast.BooleanNode{Key: v},
					&ast.OperatorNode{Value: "NOT"},
					&ast.BooleanNode{Key: v},
					&ast.OperatorNode{Value: "NOT"},
					&ast.GroupNode{Key: func() string {
						switch {
						case k == "":
							return k
						default:
							return v
						}
					}(), Nodes: []ast.Node{
						&ast.StringNode{Key: v},
						&ast.OperatorNode{Value: "AND"},
						&ast.StringNode{Key: v},
						&ast.OperatorNode{Value: "OR"},
						&ast.GroupNode{Key: func() string {
							switch {
							case k == "":
								return k
							default:
								return v
							}
						}(), Nodes: []ast.Node{
							&ast.StringNode{Key: v},
							&ast.OperatorNode{Value: "AND"},
							&ast.StringNode{Key: v},
							&ast.OperatorNode{Value: "OR"},
							&ast.GroupNode{Key: func() string {
								switch {
								case k == "":
									return k
								default:
									return v
								}
							}(), Nodes: []ast.Node{
								&ast.StringNode{Key: v},
								&ast.OperatorNode{Value: "AND"},
								&ast.StringNode{Key: v},
							}},
						}},
					}},
				},
			})
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				result, err := convert.ExpandKQL(test.Got)
				require.NoError(t, err)
				require.Equal(t, test.Want, result)
			})
		}
	})

	t.Run("lowercases some values", func(t *testing.T) {
		tests := []opensearchtest.TableTest[[]ast.Node, []ast.Node]{
			{
				Name: "!Hidden: StringNode -> stringnode",
				Got: []ast.Node{
					ast.StringNode{Key: "aBc", Value: "StringNode"},
					ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
						ast.StringNode{Key: "aBc", Value: "StringNode"},
					}},
				},
				Want: []ast.Node{
					&ast.StringNode{Key: "aBc", Value: "stringnode"},
					&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
						&ast.StringNode{Key: "aBc", Value: "stringnode"},
					}},
				},
			},
			{
				Name: "Hidden: StringNode -> StringNode",
				Got: []ast.Node{
					ast.StringNode{Key: "Hidden", Value: "StringNode"},
					ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
						ast.StringNode{Key: "Hidden", Value: "StringNode"},
					}},
				},
				Want: []ast.Node{
					&ast.StringNode{Key: "Hidden", Value: "StringNode"},
					&ast.GroupNode{Key: "GroupNode", Nodes: []ast.Node{
						&ast.StringNode{Key: "Hidden", Value: "StringNode"},
					}},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				result, err := convert.ExpandKQL(test.Got)
				require.NoError(t, err)
				require.Equal(t, test.Want, result)
			})
		}
	})

	t.Run("unfolds some values", func(t *testing.T) {
		tests := []opensearchtest.TableTest[[]ast.Node, []ast.Node]{
			{
				Name: "MimeType:unknown",
				Got: []ast.Node{
					&ast.StringNode{Key: "MimeType", Value: "unknown"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.StringNode{Key: "MimeType", Value: "unknown"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:file",
				Got: []ast.Node{
					&ast.StringNode{Key: "MimeType", Value: "file"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.OperatorNode{Value: "NOT"},
					&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:folder",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "folder"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:document",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "document"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "MimeType", Value: "application/msword"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.openxmlformats-officedocument.wordprocessingml.form"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.oasis.opendocument.text"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "text/plain"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "text/markdown"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/rtf"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.apple.pages"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:spreadsheet",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "spreadsheet"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.ms-excel"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.oasis.opendocument.spreadsheet"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "text/csv"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.oasis.opendocument.spreadshee"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.apple.numbers"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:presentation",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "presentation"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.oasis.opendocument.presentation"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.ms-powerpoint"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/vnd.apple.keynote"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:pdf",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "pdf"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "MimeType", Value: "application/pdf"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:image",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "image"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "MimeType", Value: "image/*"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:video",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "video"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "MimeType", Value: "video/*"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:audio",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "audio"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "MimeType", Value: "audio/*"},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
			{
				Name: "MimeType:archive",
				Got: []ast.Node{
					ast.BooleanNode{Key: "Deleted", Value: false},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Key: "MimeType", Value: "archive"},
					ast.OperatorNode{Value: "AND"},
					ast.StringNode{Value: "some-name"},
				},
				Want: []ast.Node{
					&ast.BooleanNode{Key: "Deleted", Value: false},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "MimeType", Value: "application/zip"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/gzip"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-gzip"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-7z-compressed"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-rar-compressed"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-tar"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-bzip2"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-bzip"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "MimeType", Value: "application/x-tgz"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.StringNode{Key: "Name", Value: `some-name`},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				if test.Skip {
					t.Skip("Skipping test due to known issue")
				}
				result, err := convert.ExpandKQL(test.Got)
				require.NoError(t, err)
				require.EqualValues(t, test.Want, result)
			})
		}
	})

	t.Run("different cases", func(t *testing.T) {
		tests := []opensearchtest.TableTest[[]ast.Node, []ast.Node]{
			{
				Name: "use the group node key as default key",
				Got: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Key: "a", Nodes: []ast.Node{
						&ast.StringNode{Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Key: "mediatype", Nodes: []ast.Node{
						&ast.StringNode{Value: "file"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "mediatype", Value: "file"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
				},
				Want: []ast.Node{
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.StringNode{Key: "Name", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Key: "a", Nodes: []ast.Node{
						&ast.StringNode{Key: "a", Value: "b"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Key: "MimeType", Nodes: []ast.Node{
						&ast.OperatorNode{Value: "NOT"},
						&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
					&ast.OperatorNode{Value: "AND"},
					&ast.GroupNode{Nodes: []ast.Node{
						&ast.OperatorNode{Value: "NOT"},
						&ast.StringNode{Key: "MimeType", Value: "httpd/unix-directory"},
						&ast.OperatorNode{Value: "OR"},
						&ast.StringNode{Key: "c", Value: "d"},
					}},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				if test.Skip {
					t.Skip("Skipping test due to known issue")
				}
				result, err := convert.ExpandKQL(test.Got)
				require.NoError(t, err)
				require.EqualValues(t, test.Want, result)
			})
		}
	})
}
