package displaywidth

// propertyWidths is a jump table of sorts, instead of a switch
var propertyWidths = [5]int{
	_Default:              1,
	_Zero_Width:           0,
	_East_Asian_Wide:      2,
	_East_Asian_Ambiguous: 1,
	_Emoji:                2,
}

// asciiWidths is a lookup table for single-byte character widths. Printable
// ASCII characters have width 1, control characters have width 0.
//
// It is intended for valid single-byte UTF-8, which means <128.
//
// If you look up an index >= 128, that is either:
//   - invalid UTF-8, or
//   - a multi-byte UTF-8 sequence, in which case you should be operating on
//     the grapheme cluster, and not using this table
//
// We will return a default value of 1 in those cases, so as not to panic.
var asciiWidths = [256]int8{
	// Control characters (0x00-0x1F): width 0
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	// Printable ASCII (0x20-0x7E): width 1
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	// DEL (0x7F): width 0
	0,
	// >= 128
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
}

// asciiProperties is a lookup table for single-byte character properties.
// It is intended for valid single-byte UTF-8, which means <128.
//
// If you look up an index >= 128, that is either:
//   - invalid UTF-8, or
//   - a multi-byte UTF-8 sequence, in which case you should be operating on
//     the grapheme cluster, and not using this table
//
// We will return a default value of _Default in those cases, so as not to
// panic.
var asciiProperties = [256]property{
	// Control characters (0x00-0x1F): _Zero_Width
	_Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width,
	_Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width,
	_Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width,
	_Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width, _Zero_Width,
	// Printable ASCII (0x20-0x7E): _Default
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default,
	// DEL (0x7F): _Zero_Width
	_Zero_Width,
	// >= 128
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
	_Default, _Default, _Default, _Default, _Default, _Default, _Default, _Default,
}
