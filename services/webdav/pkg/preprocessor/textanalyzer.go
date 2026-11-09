package preprocessor

import (
	"unicode"
	"unicode/utf8"
)

// Default list of scripts to be analyzed within the string.
//
// Scripts that aren't present in the list will be considered as part
// of the last "known" script. For example, if "Avestan" script (which isn't
// present) is preceeded by "Arabic" script, then the "Avestan" script will
// be considered as "Arabic"
//
// Punctuation symbols are usually considered part of the "Common" script
var DefaultScripts = []string{
	"Arabic",
	"Common",
	"Devanagari",
	"Han",
	"Hangul",
	"Hiragana",
	"Inherited",
	"Katakana",
	"Latin",
}

// Convenient map[string]map[string]string type used to merge multiple
// scripts into one. This is mainly used for japanese language which uses
// "Han", "Hiragana" and "Katakana" scripts.
//
// The map contains the expected previous script as first key, the expected
// current script as second key, and the resulting script (if both keys
// match) as value
type MergeMap map[string]map[string]string

// The default mergeMap containing info for the japanese scripts
var DefaultMergeMap = MergeMap{
	"Han": map[string]string{
		"Hiragana": "Hiragana",
		"Katakana": "Katakana",
	},
	"Hiragana": map[string]string{
		"Han":      "Hiragana",
		"Katakana": "Hiragana",
	},
	"Katakana": map[string]string{
		"Han":      "Katakana",
		"Hiragana": "Hiragana",
	},
}

// Analysis options.
type AnalysisOpts struct {
	UseMergeMap bool
	MergeMap    MergeMap
}

// A script range. The range should be attached to a string which could contain
// multiple scripts. The "TargetScript" will go from bytes "Low" to "High"
// (both inclusive), and contains a "RuneCount" number of runes or chars
// (mostly for debugging purposes).
// The Space contains the bytes (inside the range) that are considered as
// white space.
type ScriptRange struct {
	Low, High    int
	Spaces       []int
	TargetScript string
	RuneCount    int
}

// The result of a text analysis. It contains the analyzed text, a list of
// script ranges (see the ScriptRange type) and a map containing how many
// runes have been detected for a particular script.
type TextAnalysis struct {
	ScriptRanges []ScriptRange
	RuneCount    map[string]int
	Text         string
}

// The TextAnalyzer object contains private members. It should be created via
// "NewTextAnalyzer" function.
type TextAnalyzer struct {
	scripts         map[string]*unicode.RangeTable
	scriptListCache []string
}

// Create a new TextAnalyzer. A list of scripts must be provided.
// You can use the "DefaultScripts" variable for a default list,
// although it doesn't contain all the available scripts.
// See the unicode.Scripts variable (in the unicode package) for a
// full list. Note that using invalid scripts will cause an undefined
// behavior
func NewTextAnalyzer(scriptList []string) TextAnalyzer {
	scriptRanges := make(map[string]*unicode.RangeTable, len(scriptList))
	for _, script := range scriptList {
		scriptRanges[script] = unicode.Scripts[script]
	}
	return TextAnalyzer{
		scripts:         scriptRanges,
		scriptListCache: scriptList,
	}
}

// Analyze the target string using the specified options.
// A TextAnalysis will be returned with the result of the analysis.
func (ta *TextAnalyzer) AnalyzeString(word string, opts AnalysisOpts) TextAnalysis {
	analysis := TextAnalysis{
		ScriptRanges: []ScriptRange{},
		RuneCount:    make(map[string]int),
		Text:         word,
	}

	if len(word) < 1 {
		return analysis
	}

	firstRune, runeLen := utf8.DecodeRuneInString(word)

	lastRange := &ScriptRange{
		Low:          0,
		Spaces:       make([]int, 0),
		TargetScript: ta.chooseScriptFor(firstRune),
	}
	firstRuneIsWhiteSpace := unicode.Is(unicode.White_Space, firstRune)
	if firstRuneIsWhiteSpace {
		lastRange.Spaces = append(lastRange.Spaces, 0)
	}

	runeCount := 1
	for wordIndex, char := range word[runeLen:] {
		wordIndex += runeLen // shifted from the original string
		script := ta.chooseScriptFor(char)

		isWhiteSpace := unicode.Is(unicode.White_Space, char)
		if script != lastRange.TargetScript {
			if mapScript, isOk := ta.getMergeMapValue(opts, lastRange.TargetScript, script); isOk {
				lastRange.TargetScript = mapScript
				if isWhiteSpace {
					// TODO: Check if this is dead code.
					// whitespace should be part of the "Common" script, and the Common
					// script shouldn't be part of a mergeMap
					lastRange.Spaces = append(lastRange.Spaces, wordIndex)
				}
				runeCount++
				continue
			}

			lastRange.High = wordIndex - 1
			lastRange.RuneCount = runeCount
			analysis.ScriptRanges = append(analysis.ScriptRanges, *lastRange)
			if _, exists := analysis.RuneCount[lastRange.TargetScript]; !exists {
				analysis.RuneCount[lastRange.TargetScript] = 0
			}
			analysis.RuneCount[lastRange.TargetScript] += runeCount
			lastRange = &ScriptRange{
				Low:          wordIndex,
				Spaces:       make([]int, 0),
				TargetScript: script,
			}
			runeCount = 0
		}
		runeCount++
		if isWhiteSpace {
			lastRange.Spaces = append(lastRange.Spaces, wordIndex)
		}
	}

	// close the last range
	lastRange.High = len(word) - 1
	lastRange.RuneCount = runeCount
	analysis.RuneCount[lastRange.TargetScript] += runeCount
	analysis.ScriptRanges = append(analysis.ScriptRanges, *lastRange)

	return analysis
}

func (ta *TextAnalyzer) chooseScriptFor(char rune) string {
	script := "_unknown"
	for scriptIndex, scriptFound := range ta.scriptListCache {
		// if we can't match with a known script, do nothing and jump to the next char
		if unicode.Is(ta.scripts[scriptFound], char) {
			if scriptIndex > 3 {
				// we might expect more chars with the same script
				// so move the script first to match it faster next time
				ta.reorderScriptList(scriptFound)
			}
			return scriptFound
		}
	}
	return script
}

// Reorder the scriptListCache in the TextAnalyzer in order to speed up
// the next script searches. A "Latin" script is expected to be surrounded
// by "Latin" chars, although "Common" script chars might be present too
func (ta *TextAnalyzer) reorderScriptList(matchedScript string) {
	for index, script := range ta.scriptListCache {
		if script == matchedScript {
			if index != 0 {
				// move the script to the first position for a faster matching
				newList := append([]string{script}, ta.scriptListCache[:index]...)
				ta.scriptListCache = append(newList, ta.scriptListCache[index+1:]...)
			}
			// if index == 0 there is nothing to do: the element is already the first
			break
		}
	}
}

// Get the value from the merge map based on the previous and current scripts.
// The information about using the merge map and the actual merge map will be
// gotten from the AnalysisOpts passed as parameter
func (ta *TextAnalyzer) getMergeMapValue(opts AnalysisOpts, previous, current string) (string, bool) {
	if opts.UseMergeMap {
		// This option mainly target japanese chars; multiple scripts can be used
		// in the same piece of text (Han, Hiragana and Katakana)
		// Instead of starting a new range, adjust the target script of the last range
		if expCurrent, currentOk := opts.MergeMap[previous]; currentOk {
			if expFinal, finalOk := expCurrent[current]; finalOk {
				return expFinal, finalOk
			}
		}
	}
	return "", false
}

// Change the "Common" script to the one used in the previous script range.
// The ranges will be readjusted and merged if they're adjacent.
// This naive approach should be good enough for normal use cases
//
// The MergeMap is needed in case of the japanese language: the ranges
// "Han"-"Common"-"Katakana" might be replaced to "Han"-"Hiragana"-"Katakana"
// However, the ranges should be merged together into a big "Hiragana" range.
// If the MergeMap isn't needed, use an empty one
func (tr *TextAnalysis) MergeCommon(mergeMap MergeMap) {
	var finalRanges []ScriptRange

	if len(tr.ScriptRanges) < 1 {
		// no ranges -> nothing to do
		return
	}

	previousRange := &ScriptRange{}
	*previousRange = tr.ScriptRanges[0]
	for _, sRange := range tr.ScriptRanges[1:] {
		if previousRange.TargetScript == sRange.TargetScript {
			previousRange.High = sRange.High
			previousRange.Spaces = append(previousRange.Spaces, sRange.Spaces...)
			previousRange.RuneCount += sRange.RuneCount
		} else if sRange.TargetScript == "Common" || sRange.TargetScript == "Inherited" {
			// new range will be absorbed into the previous one
			previousRange.High = sRange.High
			previousRange.Spaces = append(previousRange.Spaces, sRange.Spaces...)
			previousRange.RuneCount += sRange.RuneCount
			tr.RuneCount[previousRange.TargetScript] += sRange.RuneCount
			tr.RuneCount[sRange.TargetScript] -= sRange.RuneCount
		} else if previousRange.TargetScript == "Common" || previousRange.TargetScript == "Inherited" {
			// might happen if the text starts with a Common script
			previousRange.High = sRange.High
			previousRange.Spaces = append(previousRange.Spaces, sRange.Spaces...)
			tr.RuneCount[sRange.TargetScript] += previousRange.RuneCount
			tr.RuneCount[previousRange.TargetScript] -= previousRange.RuneCount
			previousRange.RuneCount += sRange.RuneCount
			previousRange.TargetScript = sRange.TargetScript
		} else {
			if mapScript, isOk := tr.getMergeMapValue(mergeMap, previousRange.TargetScript, sRange.TargetScript); isOk {
				if sRange.TargetScript == mapScript {
					// the previous range has changed the target script
					tr.RuneCount[previousRange.TargetScript] -= previousRange.RuneCount
					tr.RuneCount[sRange.TargetScript] += previousRange.RuneCount
				} else {
					// new range has been absorbed
					tr.RuneCount[sRange.TargetScript] -= sRange.RuneCount
					tr.RuneCount[previousRange.TargetScript] += sRange.RuneCount
				}
				previousRange.TargetScript = mapScript
				previousRange.High = sRange.High
				previousRange.Spaces = append(previousRange.Spaces, sRange.Spaces...)
				previousRange.RuneCount += sRange.RuneCount
				continue
			}
			finalRanges = append(finalRanges, *previousRange)
			*previousRange = sRange
		}
	}

	finalRanges = append(finalRanges, *previousRange)
	tr.ScriptRanges = finalRanges
	delete(tr.RuneCount, "Common")
	delete(tr.RuneCount, "Inherited")
	for index, rCount := range tr.RuneCount {
		if rCount == 0 {
			delete(tr.RuneCount, index)
		}
	}
}

func (tr *TextAnalysis) getMergeMapValue(mMap MergeMap, previous, current string) (string, bool) {
	// This option mainly target japanese chars; multiple scripts can be used
	// in the same piece of text (Han, Hiragana and Katakana)
	// Instead of starting a new range, adjust the target script of the last range
	if expCurrent, currentOk := mMap[previous]; currentOk {
		if expFinal, finalOk := expCurrent[current]; finalOk {
			return expFinal, finalOk
		}
	}
	return "", false
}
