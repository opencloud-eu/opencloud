package stringutil

// FindUnquoted returns the indexes of the instance of v in s, or empty slice if v is not present in s.
// It ignores v present inside quoted runs.
func FindUnquoted(s string, v rune, quote rune) []int {
	escaped := false
	quoted := false
	indexes := make([]int, 0)
	quotedIndexes := make([]int, 0)

	for i := 0; i < len(s); i++ {
		switch rune(s[i]) {
		case escape:
			escaped = !escaped // escape can escape itself.
		case quote:
			if escaped {
				escaped = false
				continue
			}

			quoted = !quoted
			if !quoted {
				quotedIndexes = quotedIndexes[:0] // drop possible indices inside quoted segment
			}
		case v:
			escaped = false
			if quoted {
				quotedIndexes = append(quotedIndexes, i)
			} else {
				indexes = append(indexes, i)
			}
		default:
			escaped = false
		}
	}

	return append(indexes, quotedIndexes...)
}
