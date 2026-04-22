package l10n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslateStruct(t *testing.T) {

	type InnerStruct struct {
		Description string
		DisplayName *string
	}

	type TopLevelStruct struct {
		Description string
		DisplayName *string
		SubStruct   *InnerStruct
	}

	type WrapperStruct struct {
		Description string
		StructList  []*InnerStruct
	}

	toStrPointer := func(str string) *string {
		return &str
	}

	tests := []struct {
		name     string
		entity   any
		args     []TranslateOption
		expected any
		wantErr  bool
	}{
		{
			name: "top level slice of struct",
			entity: []*InnerStruct{
				{
					Description: "inner 1",
					DisplayName: toStrPointer("innerDisplayName 1"),
				},
				{
					Description: "inner 2",
					DisplayName: toStrPointer("innerDisplayName 2"),
				},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
			},
			expected: []*InnerStruct{
				{
					Description: "new Inner 1",
					DisplayName: toStrPointer("new InnerDisplayName 1"),
				},
				{
					Description: "new Inner 2",
					DisplayName: toStrPointer("new InnerDisplayName 2"),
				},
			},
		},
		{
			name: "top level slice of string",
			entity: []string{
				"inner 1",
				"inner 2",
			},
			expected: []string{
				"new Inner 1",
				"new Inner 2",
			},
		},
		{
			name: "top level slice of struct",
			entity: []*TopLevelStruct{
				{
					Description: "inner 1",
					DisplayName: toStrPointer("innerDisplayName 1"),
					SubStruct: &InnerStruct{
						Description: "inner",
						DisplayName: toStrPointer("innerDisplayName"),
					},
				},
				{
					Description: "inner 2",
					DisplayName: toStrPointer("innerDisplayName 2"),
				},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("SubStruct",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: []*TopLevelStruct{
				{
					Description: "new Inner 1",
					DisplayName: toStrPointer("new InnerDisplayName 1"),
					SubStruct: &InnerStruct{
						Description: "new Inner",
						DisplayName: toStrPointer("new InnerDisplayName"),
					},
				},
				{
					Description: "new Inner 2",
					DisplayName: toStrPointer("new InnerDisplayName 2"),
				},
			},
		},
		{
			name: "wrapped struct full",
			entity: &WrapperStruct{
				StructList: []*InnerStruct{
					{
						Description: "inner 1",
						DisplayName: toStrPointer("innerDisplayName 1"),
					},
					{
						Description: "inner 2",
						DisplayName: toStrPointer("innerDisplayName 2"),
					},
				},
			},
			args: []TranslateOption{
				TranslateEach("StructList",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &WrapperStruct{
				StructList: []*InnerStruct{
					{
						Description: "new Inner 1",
						DisplayName: toStrPointer("new InnerDisplayName 1"),
					},
					{
						Description: "new Inner 2",
						DisplayName: toStrPointer("new InnerDisplayName 2"),
					},
				},
			},
		},
		{
			name:   "empty struct, NotExistingSubStructName",
			entity: &TopLevelStruct{},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("NotExistingSubStructName",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &TopLevelStruct{},
		},
		{
			name:   "empty struct",
			entity: &TopLevelStruct{},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("SubStruct",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &TopLevelStruct{},
		},
		{
			name: "empty struct, not existing field",
			entity: &TopLevelStruct{
				Description: "description",
				DisplayName: toStrPointer("displayName"),
			},
			args: []TranslateOption{
				TranslateField("NotExistingFieldName"),
				TranslateStruct("SubStruct",
					TranslateField("NotExistingFieldName"),
				),
			},
			expected: &TopLevelStruct{
				Description: "description",
				DisplayName: toStrPointer("displayName"),
			},
		},
		{
			name: "inner struct DisplayName empy",
			entity: &TopLevelStruct{
				Description: "description",
				DisplayName: toStrPointer("displayName"),
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("SubStruct",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &TopLevelStruct{
				Description: "new Description",
				DisplayName: toStrPointer("new DisplayName"),
			},
		},
		{
			name: "inner struct full",
			entity: &TopLevelStruct{
				Description: "description",
				DisplayName: toStrPointer("displayName"),
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("SubStruct",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &TopLevelStruct{
				Description: "new Description",
				DisplayName: toStrPointer("new DisplayName"),
			},
		},
		{
			name: "full struct",
			entity: &TopLevelStruct{
				Description: "description",
				DisplayName: toStrPointer("displayName"),
				SubStruct: &InnerStruct{
					Description: "inner",
					DisplayName: toStrPointer("innerDisplayName"),
				},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
				TranslateStruct("SubStruct",
					TranslateField("Description"),
					TranslateField("DisplayName"),
				),
			},
			expected: &TopLevelStruct{
				Description: "new Description",
				DisplayName: toStrPointer("new DisplayName"),
				SubStruct: &InnerStruct{
					Description: "new Inner",
					DisplayName: toStrPointer("new InnerDisplayName"),
				},
			},
		},
		{
			name:    "nil",
			wantErr: true,
		},
		{
			name:    "empty slice",
			wantErr: true,
		},
		{
			name:     "string slice",
			entity:   []string{"description", "inner"},
			expected: []string{"new Description", "new Inner"},
		},
		{
			name: "string map",
			entity: map[string]string{
				"entryOne": "description",
				"entryTwo": "inner",
			},
			expected: map[string]string{
				"entryOne": "new Description",
				"entryTwo": "new Inner",
			},
		},
		{
			name: "pointer struct map",
			entity: map[string]*InnerStruct{
				"entryOne": {Description: "description", DisplayName: toStrPointer("displayName")},
				"entryTwo": {Description: "inner", DisplayName: toStrPointer("innerDisplayName")},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
			},
			expected: map[string]*InnerStruct{
				"entryOne": {Description: "new Description", DisplayName: toStrPointer("new DisplayName")},
				"entryTwo": {Description: "new Inner", DisplayName: toStrPointer("new InnerDisplayName")},
			},
		},
		/* FIXME: non pointer maps are currently not working
		{
			name: "struct map",
			entity: map[string]InnerStruct{
				"entryOne": {Description: "description", DisplayName: toStrPointer("displayName")},
				"entryTwo": {Description: "inner", DisplayName: toStrPointer("innerDisplayName")},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
			},
			expected: map[string]InnerStruct{
				"entryOne": {Description: "new Description", DisplayName: toStrPointer("new DisplayName")},
				"entryTwo": {Description: "new Inner", DisplayName: toStrPointer("new InnerDisplayName")},
			},
		},
		*/
		{
			name: "slice map",
			entity: map[string][]string{
				"entryOne": {"description", "inner"},
				"entryTwo": {"inner 2", "innerDisplayName 2"},
			},
			expected: map[string][]string{
				"entryOne": {"new Description", "new Inner"},
				"entryTwo": {"new Inner 2", "new InnerDisplayName 2"},
			},
		},
		{
			name: "double slice",
			entity: [][]string{
				{"description", "inner"},
				{"inner 2", "innerDisplayName 2"},
			},
			expected: [][]string{
				{"new Description", "new Inner"},
				{"new Inner 2", "new InnerDisplayName 2"},
			},
		},
		{
			name: "nested structs",
			entity: [][]*InnerStruct{
				{
					&InnerStruct{Description: "description", DisplayName: toStrPointer("displayName")},
					&InnerStruct{Description: "inner", DisplayName: toStrPointer("innerDisplayName")},
				},
				{
					&InnerStruct{Description: "inner 2", DisplayName: toStrPointer("innerDisplayName 2")},
				},
			},
			args: []TranslateOption{
				TranslateField("Description"),
				TranslateField("DisplayName"),
			},
			expected: [][]*InnerStruct{
				{
					&InnerStruct{Description: "new Description", DisplayName: toStrPointer("new DisplayName")},
					&InnerStruct{Description: "new Inner", DisplayName: toStrPointer("new InnerDisplayName")},
				},
				{
					&InnerStruct{Description: "new Inner 2", DisplayName: toStrPointer("new InnerDisplayName 2")},
				},
			},
		},
		{
			name: "double mapslices",
			entity: []map[string][]string{
				{
					"entryOne": {"inner 1", "innerDisplayName 1"},
					"entryTwo": {"inner 2", "innerDisplayName 2"},
				},
				{
					"entryOne": {"description", "displayName"},
				},
			},
			expected: []map[string][]string{
				{
					"entryOne": {"new Inner 1", "new InnerDisplayName 1"},
					"entryTwo": {"new Inner 2", "new InnerDisplayName 2"},
				},
				{
					"entryOne": {"new Description", "new DisplayName"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TranslateEntity(mock(), tt.entity, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("TranslateEntity() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.expected, tt.entity)
		})
	}
}

func mock() func(string, ...any) string {
	return func(s string, i ...any) string {
		switch s {
		case "description":
			return "new Description"
		case "displayName":
			return "new DisplayName"
		case "inner":
			return "new Inner"
		case "innerDisplayName":
			return "new InnerDisplayName"
		case "inner 1":
			return "new Inner 1"
		case "innerDisplayName 1":
			return "new InnerDisplayName 1"
		case "inner 2":
			return "new Inner 2"
		case "innerDisplayName 2":
			return "new InnerDisplayName 2"
		}
		return s
	}
}
