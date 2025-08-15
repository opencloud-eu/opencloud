package byteutil

// IsNumChar returns true if the given character is a numeric, otherwise false.
func IsNumChar(c byte) bool { return c >= '0' && c <= '9' }

// IsAlphaChar returns true if the given character is a alphabet, otherwise false.
func IsAlphaChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

