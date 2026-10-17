package preprocessor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	inputs = [18]string{
		"basic latin",
		"trailing tab	",
		"Small text. \"$\", \"£\" and \"¥\" are currencies.",
		"latin with 🖖",
		"기본 한국어",
		"基本的な日本語",
		"ウーロン茶",
		"私はエンジニアです",
		"ティー私はエンジニアです",
		"私はエンジニアです ティー",
		"आधारभूत देवनागरी",
		"mixed 언어 传入 🚀!",
		"/k͜p/",
		// ä and a + ¨
		"ä ä",
		"базовый русский", // cyrillic script isn't part of our default
		"latin русский",   // latin + cyrillic (cyrillic not supported)
		" space justified ",
		"",
	}
)

func TestAnalyzeString(t *testing.T) {
	defaultOpts := AnalysisOpts{
		UseMergeMap: true,
		MergeMap:    DefaultMergeMap,
	}

	tables := []struct {
		input string
		opts  AnalysisOpts
		eOut  TextAnalysis
	}{
		{
			input: inputs[0],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 10, Spaces: []int{5}, TargetScript: "Latin", RuneCount: 11},
				},
				RuneCount: map[string]int{
					"Latin": 11,
				},
				Text: inputs[0],
			},
		},
		{
			input: inputs[1],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 12, Spaces: []int{8, 12}, TargetScript: "Latin", RuneCount: 13},
				},
				RuneCount: map[string]int{
					"Latin": 13,
				},
				Text: inputs[1],
			},
		},
		{
			input: inputs[2],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 45, Spaces: []int{5, 11, 16, 21, 25, 30, 34}, TargetScript: "Latin", RuneCount: 44},
				},
				RuneCount: map[string]int{
					"Latin": 44,
				},
				Text: inputs[2],
			},
		},
		{
			input: inputs[3],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 14, Spaces: []int{5, 10}, TargetScript: "Latin", RuneCount: 12},
				},
				RuneCount: map[string]int{
					"Latin": 12,
				},
				Text: inputs[3],
			},
		},
		{
			input: inputs[4],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 15, Spaces: []int{6}, TargetScript: "Hangul", RuneCount: 6},
				},
				RuneCount: map[string]int{
					"Hangul": 6,
				},
				Text: inputs[4],
			},
		},
		{
			input: inputs[5],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 20, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 7},
				},
				RuneCount: map[string]int{
					"Hiragana": 7,
				},
				Text: inputs[5],
			},
		},
		{
			input: inputs[6],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 14, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 5},
				},
				RuneCount: map[string]int{
					"Katakana": 5,
				},
				Text: inputs[6],
			},
		},
		{
			input: inputs[7],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 26, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 9},
				},
				RuneCount: map[string]int{
					"Hiragana": 9,
				},
				Text: inputs[7],
			},
		},
		{
			input: inputs[8],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 35, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 12},
				},
				RuneCount: map[string]int{
					"Hiragana": 12,
				},
				Text: inputs[8],
			},
		},
		{
			input: inputs[9],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 36, Spaces: []int{27}, TargetScript: "Hiragana", RuneCount: 13},
				},
				RuneCount: map[string]int{
					"Hiragana": 13,
				},
				Text: inputs[9],
			},
		},
		{
			input: inputs[10],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 45, Spaces: []int{21}, TargetScript: "Devanagari", RuneCount: 16},
				},
				RuneCount: map[string]int{
					"Devanagari": 16,
				},
				Text: inputs[10],
			},
		},
		{
			input: inputs[11],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{5}, TargetScript: "Latin", RuneCount: 6},
					{Low: 6, High: 12, Spaces: []int{12}, TargetScript: "Hangul", RuneCount: 3},
					{Low: 13, High: 24, Spaces: []int{19}, TargetScript: "Han", RuneCount: 5}, // 🚀 and ! are "Common" script and will be merged with "Han"
				},
				RuneCount: map[string]int{
					"Latin":  6,
					"Hangul": 3,
					"Han":    5,
				},
				Text: inputs[11],
			},
		},
		{
			input: inputs[12],
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
				},
				RuneCount: map[string]int{
					"Latin": 5,
				},
				Text: inputs[12],
			},
		},
		{
			input: inputs[13], // ä and a + ¨
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{2}, TargetScript: "Latin", RuneCount: 4},
				},
				RuneCount: map[string]int{
					"Latin": 4,
				},
				Text: inputs[13],
			},
		},
		{
			input: inputs[14], // cyrillic script isn't part of our default
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 28, Spaces: []int{14}, TargetScript: "_unknown", RuneCount: 15},
				},
				RuneCount: map[string]int{
					"_unknown": 15,
				},
				Text: inputs[14],
			},
		},
		{
			input: inputs[15], // latin + cyrillic (cyrillic script isn't part of our default)
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{5}, TargetScript: "Latin", RuneCount: 6},
					{Low: 6, High: 19, Spaces: []int{}, TargetScript: "_unknown", RuneCount: 7},
				},
				RuneCount: map[string]int{
					"Latin":    6,
					"_unknown": 7,
				},
				Text: inputs[15],
			},
		},
		{
			input: inputs[16], // latin + cyrillic (cyrillic script isn't part of our default)
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 16, Spaces: []int{0, 6, 16}, TargetScript: "Latin", RuneCount: 17},
				},
				RuneCount: map[string]int{
					"Latin": 17,
				},
				Text: inputs[16],
			},
		},
		{
			input: inputs[17], // latin + cyrillic (cyrillic script isn't part of our default)
			opts:  defaultOpts,
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{},
				RuneCount:    map[string]int{},
				Text:         inputs[17],
			},
		},
	}

	for _, table := range tables {
		testname := fmt.Sprintf("Analyzing \"%s\" string", table.input)
		t.Run(testname, func(t *testing.T) {
			ta := NewTextAnalyzer(DefaultScripts)
			result := ta.AnalyzeString(table.input, table.opts)
			if table.opts.UseMergeMap {
				result.MergeCommon(table.opts.MergeMap)
			} else {
				result.MergeCommon(MergeMap{})
			}
			assert.Equal(t, table.eOut, result)
		})
	}
}

func TestAnalyzeStringRaw(t *testing.T) {
	tables := []struct {
		input string
		eOut  TextAnalysis
	}{
		{
			input: inputs[0],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 5, High: 5, Spaces: []int{5}, TargetScript: "Common", RuneCount: 1},
					{Low: 6, High: 10, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
				},
				RuneCount: map[string]int{
					"Latin":  10,
					"Common": 1,
				},
				Text: inputs[0],
			},
		},
		{
			input: inputs[1],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 7, Spaces: []int{}, TargetScript: "Latin", RuneCount: 8},
					{Low: 8, High: 8, Spaces: []int{8}, TargetScript: "Common", RuneCount: 1},
					{Low: 9, High: 11, Spaces: []int{}, TargetScript: "Latin", RuneCount: 3},
					{Low: 12, High: 12, Spaces: []int{12}, TargetScript: "Common", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Latin":  11,
					"Common": 2,
				},
				Text: inputs[1],
			},
		},
		{
			input: inputs[2],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 5, High: 5, Spaces: []int{5}, TargetScript: "Common", RuneCount: 1},
					{Low: 6, High: 9, Spaces: []int{}, TargetScript: "Latin", RuneCount: 4},
					{Low: 10, High: 21, Spaces: []int{11, 16, 21}, TargetScript: "Common", RuneCount: 11}, // £ takes 2 bytes
					{Low: 22, High: 24, Spaces: []int{}, TargetScript: "Latin", RuneCount: 3},
					{Low: 25, High: 30, Spaces: []int{25, 30}, TargetScript: "Common", RuneCount: 5}, // ¥ takes 2 bytes
					{Low: 31, High: 33, Spaces: []int{}, TargetScript: "Latin", RuneCount: 3},
					{Low: 34, High: 34, Spaces: []int{34}, TargetScript: "Common", RuneCount: 1},
					{Low: 35, High: 44, Spaces: []int{}, TargetScript: "Latin", RuneCount: 10},
					{Low: 45, High: 45, Spaces: []int{}, TargetScript: "Common", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Latin":  25,
					"Common": 19,
				},
				Text: inputs[2],
			},
		},
		{
			input: inputs[3],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 5, High: 5, Spaces: []int{5}, TargetScript: "Common", RuneCount: 1},
					{Low: 6, High: 9, Spaces: []int{}, TargetScript: "Latin", RuneCount: 4},
					{Low: 10, High: 14, Spaces: []int{10}, TargetScript: "Common", RuneCount: 2},
				},
				RuneCount: map[string]int{
					"Latin":  9,
					"Common": 3,
				},
				Text: inputs[3],
			},
		},
		{
			input: inputs[4],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{}, TargetScript: "Hangul", RuneCount: 2},
					{Low: 6, High: 6, Spaces: []int{6}, TargetScript: "Common", RuneCount: 1},
					{Low: 7, High: 15, Spaces: []int{}, TargetScript: "Hangul", RuneCount: 3},
				},
				RuneCount: map[string]int{
					"Hangul": 5,
					"Common": 1,
				},
				Text: inputs[4],
			},
		},
		{
			input: inputs[5],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 8, Spaces: []int{}, TargetScript: "Han", RuneCount: 3},
					{Low: 9, High: 11, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 1},
					{Low: 12, High: 20, Spaces: []int{}, TargetScript: "Han", RuneCount: 3},
				},
				RuneCount: map[string]int{
					"Hiragana": 1,
					"Han":      6,
				},
				Text: inputs[5],
			},
		},
		{
			input: inputs[6],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 2, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 1},
					{Low: 3, High: 5, Spaces: []int{}, TargetScript: "Common", RuneCount: 1}, // ー U+30FC (KATAKANA-HIRAGANA PROLONGED SOUND MARK) seems to be counted as Common
					{Low: 6, High: 11, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 2},
					{Low: 12, High: 14, Spaces: []int{}, TargetScript: "Han", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Katakana": 3,
					"Common":   1,
					"Han":      1,
				},
				Text: inputs[6],
			},
		},
		{
			input: inputs[7],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 2, Spaces: []int{}, TargetScript: "Han", RuneCount: 1},
					{Low: 3, High: 5, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 1},
					{Low: 6, High: 20, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 5},
					{Low: 21, High: 26, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 2},
				},
				RuneCount: map[string]int{
					"Han":      1,
					"Hiragana": 3,
					"Katakana": 5,
				},
				Text: inputs[7],
			},
		},
		{
			input: inputs[8],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 5, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 2},
					{Low: 6, High: 8, Spaces: []int{}, TargetScript: "Common", RuneCount: 1},
					{Low: 9, High: 11, Spaces: []int{}, TargetScript: "Han", RuneCount: 1},
					{Low: 12, High: 14, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 1},
					{Low: 15, High: 29, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 5},
					{Low: 30, High: 35, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 2},
				},
				RuneCount: map[string]int{
					"Han":      1,
					"Hiragana": 3,
					"Katakana": 7,
					"Common":   1,
				},
				Text: inputs[8],
			},
		},
		{
			input: inputs[9],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 2, Spaces: []int{}, TargetScript: "Han", RuneCount: 1},
					{Low: 3, High: 5, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 1},
					{Low: 6, High: 20, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 5},
					{Low: 21, High: 26, Spaces: []int{}, TargetScript: "Hiragana", RuneCount: 2},
					{Low: 27, High: 27, Spaces: []int{27}, TargetScript: "Common", RuneCount: 1},
					{Low: 28, High: 33, Spaces: []int{}, TargetScript: "Katakana", RuneCount: 2},
					{Low: 34, High: 36, Spaces: []int{}, TargetScript: "Common", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Han":      1,
					"Hiragana": 3,
					"Katakana": 7,
					"Common":   2,
				},
				Text: inputs[9],
			},
		},
		{
			input: inputs[10],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 20, Spaces: []int{}, TargetScript: "Devanagari", RuneCount: 7},
					{Low: 21, High: 21, Spaces: []int{21}, TargetScript: "Common", RuneCount: 1},
					{Low: 22, High: 45, Spaces: []int{}, TargetScript: "Devanagari", RuneCount: 8},
				},
				RuneCount: map[string]int{
					"Devanagari": 15,
					"Common":     1,
				},
				Text: inputs[10],
			},
		},
		{
			input: inputs[11],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 5, High: 5, Spaces: []int{5}, TargetScript: "Common", RuneCount: 1},
					{Low: 6, High: 11, Spaces: []int{}, TargetScript: "Hangul", RuneCount: 2},
					{Low: 12, High: 12, Spaces: []int{12}, TargetScript: "Common", RuneCount: 1},
					{Low: 13, High: 18, Spaces: []int{}, TargetScript: "Han", RuneCount: 2},
					{Low: 19, High: 24, Spaces: []int{19}, TargetScript: "Common", RuneCount: 3},
				},
				RuneCount: map[string]int{
					"Latin":  5,
					"Hangul": 2,
					"Han":    2,
					"Common": 5,
				},
				Text: inputs[11],
			},
		},
		{
			input: inputs[12],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 0, Spaces: []int{}, TargetScript: "Common", RuneCount: 1},
					{Low: 1, High: 1, Spaces: []int{}, TargetScript: "Latin", RuneCount: 1},
					{Low: 2, High: 3, Spaces: []int{}, TargetScript: "Inherited", RuneCount: 1},
					{Low: 4, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 1},
					{Low: 5, High: 5, Spaces: []int{}, TargetScript: "Common", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Latin":     2,
					"Common":    2,
					"Inherited": 1,
				},
				Text: inputs[12],
			},
		},
		{
			input: inputs[13], // ä and a + ¨
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 1, Spaces: []int{}, TargetScript: "Latin", RuneCount: 1},
					{Low: 2, High: 2, Spaces: []int{2}, TargetScript: "Common", RuneCount: 1},
					{Low: 3, High: 3, Spaces: []int{}, TargetScript: "Latin", RuneCount: 1},
					{Low: 4, High: 5, Spaces: []int{}, TargetScript: "Inherited", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Latin":     2,
					"Common":    1,
					"Inherited": 1,
				},
				Text: inputs[13],
			},
		},
		{
			input: inputs[14], // cyrillic script isn't part of our default
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 13, Spaces: []int{}, TargetScript: "_unknown", RuneCount: 7},
					{Low: 14, High: 14, Spaces: []int{14}, TargetScript: "Common", RuneCount: 1},
					{Low: 15, High: 28, Spaces: []int{}, TargetScript: "_unknown", RuneCount: 7},
				},
				RuneCount: map[string]int{
					"_unknown": 14,
					"Common":   1,
				},
				Text: inputs[14],
			},
		},
		{
			input: inputs[15], // latin + cyrillic (cyrillic script isn't part of our default)
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 4, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 5, High: 5, Spaces: []int{5}, TargetScript: "Common", RuneCount: 1},
					{Low: 6, High: 19, Spaces: []int{}, TargetScript: "_unknown", RuneCount: 7},
				},
				RuneCount: map[string]int{
					"Latin":    5,
					"Common":   1,
					"_unknown": 7,
				},
				Text: inputs[15],
			},
		},
		{
			input: inputs[16],
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{
					{Low: 0, High: 0, Spaces: []int{0}, TargetScript: "Common", RuneCount: 1},
					{Low: 1, High: 5, Spaces: []int{}, TargetScript: "Latin", RuneCount: 5},
					{Low: 6, High: 6, Spaces: []int{6}, TargetScript: "Common", RuneCount: 1},
					{Low: 7, High: 15, Spaces: []int{}, TargetScript: "Latin", RuneCount: 9},
					{Low: 16, High: 16, Spaces: []int{16}, TargetScript: "Common", RuneCount: 1},
				},
				RuneCount: map[string]int{
					"Latin":  14,
					"Common": 3,
				},
				Text: inputs[16],
			},
		},
		{
			input: inputs[17], // empty string
			eOut: TextAnalysis{
				ScriptRanges: []ScriptRange{},
				RuneCount:    map[string]int{},
				Text:         inputs[17],
			},
		},
	}

	for _, table := range tables {
		testname := fmt.Sprintf("Raw-Analyzing \"%s\" string", table.input)
		t.Run(testname, func(t *testing.T) {
			ta := NewTextAnalyzer(DefaultScripts)
			result := ta.AnalyzeString(table.input, AnalysisOpts{})

			assert.Equal(t, table.eOut, result)
		})
	}
}
