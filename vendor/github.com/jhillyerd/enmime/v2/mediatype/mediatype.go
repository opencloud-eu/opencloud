package mediatype

import (
	"fmt"
	"mime"
	"strings"
	_utf8 "unicode/utf8"

	"github.com/jhillyerd/enmime/v2/internal/coding"
	"github.com/jhillyerd/enmime/v2/internal/stringutil"
	"github.com/pkg/errors"
)

const (
	// Standard MIME content types
	ctAppPrefix       = "application/"
	ctAppOctetStream  = "application/octet-stream"
	ctMultipartMixed  = "multipart/mixed"
	ctMultipartPrefix = "multipart/"
	ctTextPrefix      = "text/"
	ctTextPlain       = "text/plain"

	// Used as a placeholder in case of malformed Content-Type headers
	ctPlaceholder = "x-not-a-mime-type/x-not-a-mime-type"
	// Used as a placeholder param value in case of malformed
	// Content-Type/Content-Disposition parameters that lack values.
	// E.g.: Content-Type: text/html;iso-8859-1
	pvPlaceholder = "not-a-param-value"

	utf8 = "utf-8"
)

// ParseOptions controls the parsing of content-type and media-type strings.
type ParseOptions struct {
	StripMediaTypeInvalidCharacters bool
}

// Parse is a more tolerant implementation of Go's mime.ParseMediaType function.
//
// Tolerances accounted for:
//   - Missing ';' between content-type and media parameters
//   - Repeating media parameters
//   - Unquoted values in media parameters containing 'tspecials' characters
//   - Newline characters
func Parse(ctype string) (mtype string, params map[string]string, invalidParams []string, err error) {
	return ParseWithOptions(ctype, ParseOptions{})
}

// ParseWithOptions parses media-type with additional options controlling the parsing behavior.
func ParseWithOptions(ctype string, options ParseOptions) (mtype string, params map[string]string, invalidParams []string, err error) {
	mtype, params, err = mime.ParseMediaType(
		fixNewlines(fixUnescapedQuotes(fixUnquotedSpecials(fixMangledMediaType(removeTrailingHTMLTags(ctype), ';', options)))))
	if err != nil {
		if err.Error() == "mime: no media type" {
			return "", nil, nil, nil
		}
		return "", nil, nil, errors.WithStack(err)
	}

	if mtype == ctPlaceholder {
		mtype = ""
	}
	for name, value := range params {
		if value != pvPlaceholder {
			continue
		}
		invalidParams = append(invalidParams, name)
		delete(params, name)
	}

	return mtype, params, invalidParams, err
}

// fixMangledMediaType is used to insert ; separators into media type strings that lack them, and
// remove repeated parameters.
func fixMangledMediaType(mtype string, sep rune, options ParseOptions) string {
	strsep := string([]rune{sep})
	if mtype == "" {
		return ""
	}

	parts := stringutil.SplitUnquoted(mtype, sep, '"')
	mtype = ""
	if strings.Contains(parts[0], "=") {
		// A parameter pair at this position indicates we are missing a content-type.
		parts[0] = fmt.Sprintf("%s%s %s", ctAppOctetStream, strsep, parts[0])
		parts = strings.Split(strings.Join(parts, strsep), strsep)
	}

	for i, p := range parts {
		switch i {
		case 0:
			if p == "" {
				// The content type is completely missing. Put in a placeholder.
				p = ctPlaceholder
			}
			// Remove invalid characters (specials)
			if options.StripMediaTypeInvalidCharacters {
				p = removeTypeSpecials(p)
			}
			// Check for missing token after slash.
			if strings.HasSuffix(p, "/") {
				switch p {
				case ctTextPrefix:
					p = ctTextPlain
				case ctAppPrefix:
					p = ctAppOctetStream
				case ctMultipartPrefix:
					p = ctMultipartMixed
				default:
					// Safe default
					p = ctAppOctetStream
				}
			}
			// Remove extra ctype parts
			if strings.Count(p, "/") > 1 {
				ps := strings.SplitN(p, "/", 3)
				p = strings.Join(ps[0:2], "/")
			}
		default:
			if len(p) == 0 {
				// Ignore trailing separators.
				continue
			}

			if len(strings.TrimSpace(p)) == 0 {
				// Ignore empty parameters.
				continue
			}

			if !strings.Contains(p, "=") {
				p = p + "=" + pvPlaceholder
			}

			// RFC-2047 encoded attribute name.
			p = coding.RFC2047Decode(p)

			pair := strings.SplitAfter(p, "=")

			if strings.TrimSpace(pair[0]) == "=" {
				// Ignore unnamed parameters.
				continue
			}

			if strings.Contains(mtype, strings.TrimSpace(pair[0])) {
				// Ignore repeated parameters.
				continue
			}

			if strings.ContainsAny(pair[0], "()<>@,;:\"\\/[]?") {
				// Attribute is a strict token and cannot be a quoted-string.  If any of the above
				// characters are present in a token it must be quoted and is therefor an invalid
				// attribute.  Discard the pair.
				continue
			}
		}

		mtype += p

		// Only terminate with semicolon if not the last parameter and if it doesn't already have a
		// semicolon.
		if i != len(parts)-1 && !strings.HasSuffix(mtype, ";") {
			// Remove whitespace between parameter=value and ;
			mtype = strings.TrimRight(mtype, " \t")
			mtype += ";"
		}
	}

	mtype = strings.TrimSuffix(mtype, ";")

	return mtype
}

// consumeParam takes the the parameter part of a Content-Type header, returns a clean version of
// the first parameter (quoted as necessary), and the remainder of the parameter part of the
// Content-Type header.
//
// Given this this header:
//
//	`Content-Type: text/calendar; charset=utf-8; method=text/calendar`
//
// `consumeParams` should be given this part:
//
//	` charset=utf-8; method=text/calendar`
//
// And returns (first pass):
//
//	`consumed = "charset=utf-8;"`
//	`rest     = " method=text/calendar"`
//
// Capture the `consumed` value (to build a clean Content-Type header value) and pass the value of
// `rest` back to `consumeParam`. That second call will return:
//
//	`consumed = " method=\"text/calendar\""`
//	`rest     = ""`
//
// Again, use the value of `consumed` to build a clean Content-Type header value. Given that `rest`
// is empty, all of the parameters have been consumed successfully.
//
// If `consumed` is returned empty and `rest` is not empty, then the value of `rest` does not
// begin with a parsable parameter. This does not necessarily indicate a problem. For example,
// if there is trailing whitespace, it would be returned here.
func consumeParam(s string) (consumed, rest string) {
	i := strings.IndexByte(s, '=')
	if i < 0 {
		return "", s
	}

	// Write out parameter name.
	param := strings.Builder{}
	param.WriteString(s[:i+1])
	s = s[i+1:]

	value := strings.Builder{}
	valueQuotedOriginally := false
	valueQuoteAdded := false
	valueQuoteNeeded := false
	rfc2047Needed := false

	var r rune
findValueStart:
	for i, r = range s {
		switch r {
		case ' ', '\t':
			// Do not preserve leading whitespace.

		case '"':
			valueQuotedOriginally = true
			valueQuoteAdded = true
			valueQuoteNeeded = true
			param.WriteRune(r)

			break findValueStart

		case ';':
			if value.Len() == 0 {
				// Value was empty, return immediately.
				param.WriteString(`"";`)
				return param.String(), s[i+1:]
			}

			break findValueStart

		default:
			if r > 127 {
				rfc2047Needed = true
			}

			valueQuotedOriginally = false
			valueQuoteAdded = false
			value.WriteRune(r)

			break findValueStart
		}
	}

	quoteIfUnquoted := func() {
		if !valueQuoteNeeded {
			if !valueQuoteAdded {
				param.WriteByte('"')

				valueQuoteAdded = true
			}

			valueQuoteNeeded = true
		}
	}

	if len(s)-i < 1 {
		// Parameter value starts at the end of the string, make empty
		// quoted string to play nice with mime.ParseMediaType.
		param.WriteString(`""`)
	} else {
		// The beginning of the value is not at the end of the string.
		for _, v := range []byte{'(', ')', '<', '>', '@', ',', ':', '/', '[', ']', '?', '='} {
			if s[0] == v {
				quoteIfUnquoted()
				break
			}
		}

		_, runeLength := _utf8.DecodeRuneInString(s[i:])
		s = s[i+runeLength:]
		escaped := r == '\\'

	findValueEnd:
		for i, r = range s {
			if escaped {
				value.WriteRune(r)
				escaped = false
				continue
			}

			switch r {
			case ';':
				if valueQuotedOriginally {
					// We're in a quoted string, so whitespace is allowed.
					value.WriteRune(r)
					break
				}

				// Otherwise, we've reached the end of an unquoted value.
				rest = s[i:]
				break findValueEnd

			case ' ', '\t':
				if valueQuotedOriginally {
					// We're in a quoted string, so whitespace is allowed.
					value.WriteRune(r)
					break
				}

				// This string contains whitespace, must be quoted.
				quoteIfUnquoted()
				value.WriteRune(r)

			case '"':
				if valueQuotedOriginally {
					// We're in a quoted value. This is the end of that value.
					rest = s[i:]
					break findValueEnd
				}

				quoteIfUnquoted()
				value.WriteByte('\\')
				value.WriteRune(r)

			case '\\':
				if i < len(s)-1 {
					// If next char is present, escape it with backslash.
					value.WriteRune(r)
					escaped = true
					quoteIfUnquoted()
				}

			case '(', ')', '<', '>', '@', ',', ':', '/', '[', ']', '?', '=':
				quoteIfUnquoted()
				fallthrough

			default:
				if r > 127 {
					rfc2047Needed = true
				}
				value.WriteRune(r)
			}
		}
	}

	if value.Len() > 0 {
		// Convert whole value to RFC2047 if it contains forbidden characters (ASCII > 127)
		val := value.String()
		if rfc2047Needed {
			val = mime.BEncoding.Encode(utf8, val)
			// RFC 2047 must be quoted
			quoteIfUnquoted()
		}

		// Write the value
		param.WriteString(val)
	}

	// Add final quote if required
	if valueQuoteNeeded {
		param.WriteByte('"')
	}

	// Write last parsed char if any
	if rest != "" {
		if rest[0] != '"' {
			// When last char is quote, valueQuotedOriginally is surely true and the quote was already written.
			// Otherwise output the character (; for example)
			param.WriteByte(rest[0])
		}

		// Focus the rest of the string
		rest = rest[1:]
	}

	return param.String(), rest
}

// fixUnquotedSpecials as defined in RFC 2045, section 5.1:
// https://tools.ietf.org/html/rfc2045#section-5.1
func fixUnquotedSpecials(s string) string {
	idx := strings.IndexByte(s, ';')
	if idx < 0 || idx == len(s) {
		// No parameters
		return s
	}

	clean := strings.Builder{}
	clean.WriteString(s[:idx+1])
	s = s[idx+1:]

	for len(s) > 0 {
		var consumed string
		consumed, s = consumeParam(s)

		if len(consumed) == 0 {
			clean.WriteString(s)
			return clean.String()
		}

		clean.WriteString(consumed)
	}

	return clean.String()
}

// fixUnescapedQuotes inspects for unescaped quotes inside of a quoted string and escapes them
//
//	Input:  application/rtf; charset=iso-8859-1; name=""V047411.rtf".rtf"
//	Output: application/rtf; charset=iso-8859-1; name="\"V047411.rtf\".rtf"
func fixUnescapedQuotes(hvalue string) string {
	params := stringutil.SplitAfterUnquoted(hvalue, ';', '"')
	sb := &strings.Builder{}

	for i := 0; i < len(params); i++ {
		// Inspect for "=" byte.
		eq := strings.IndexByte(params[i], '=')
		if eq < 0 {
			// No "=", must be the content-type or a comment.
			sb.WriteString(params[i])
			continue
		}
		sb.WriteString(params[i][:eq])
		param := params[i][eq:]
		startingQuote := strings.IndexByte(param, '"')
		closingQuote := strings.LastIndexByte(param, '"')

		// Opportunity to exit early if there are no quotes.
		if startingQuote < 0 && closingQuote < 0 {
			// This value is not quoted, write the value and carry on.
			sb.WriteString(param)
			continue
		}

		// Check if only one quote was found in the string.
		if closingQuote == startingQuote {
			// Append the next chunk of params here in case of a semicolon mid string.
			if len(params) > i+1 {
				param = fmt.Sprintf("%s%s", param, params[i+1])
			}
			closingQuote = strings.LastIndexByte(param, '"')
			i++
			if closingQuote == startingQuote {
				sb.WriteString("=\"\"")
				return sb.String()
			}
		}

		// Write the k/v separator back in along with everything up until the first quote.
		sb.WriteByte('=')
		sb.WriteByte('"') // Starting quote
		sb.WriteString(param[1:startingQuote])

		// Get the value, less the outer quotes.
		rest := param[closingQuote+1:]

		// If there is stuff after the last quote then we should escape the first quote.
		if len(rest) > 0 && rest != ";" {
			sb.WriteString("\\\"")
		}

		param = param[startingQuote+1 : closingQuote]
		escaped := false
		for strIdx := range []byte(param) {
			switch param[strIdx] {
			case '"':
				// We are inside of a quoted string, so lets escape this guy if it isn't already escaped.
				if !escaped {
					sb.WriteByte('\\')
					escaped = false
				}
				sb.WriteByte(param[strIdx])
			case '\\':
				// Something is getting escaped, a quote is the only char that needs
				// this, so lets assume the following char is a double-quote.
				escaped = true
				sb.WriteByte('\\')
			default:
				escaped = false
				sb.WriteByte(param[strIdx])
			}
		}

		// If there is stuff after the last quote then we should escape the last quote, apply the
		// rest and terminate with a quote.
		switch rest {
		case ";":
			sb.WriteByte('"')
			sb.WriteString(rest)
		case "":
			sb.WriteByte('"')
		default:
			sb.WriteByte('\\')
			sb.WriteByte('"')
			sb.WriteString(rest)
			sb.WriteByte('"')
		}
	}

	return sb.String()
}

// fixNewlines replaces \n with a space and removes \r
func fixNewlines(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", "")
	return value
}

// removeTrailingHTMLTags removes an unexpected HTML tags at the end of media type.
func removeTrailingHTMLTags(value string) string {
	tagStart := 0
	closeTags := 0

loop:
	for i := len(value) - 1; i > 0; i-- {
		c := value[i]
		switch c {
		case '"':
			if closeTags == 0 { // quotes started outside the tag, aborting
				break loop
			}
		case '>':
			closeTags++
		case '<':
			closeTags--
			if closeTags == 0 {
				tagStart = i
			}
		}
	}

	if tagStart != 0 {
		return value[0:tagStart]
	}

	return value
}

func removeTypeSpecials(value string) string {
	for _, r := range []string{"(", ")", "<", ">", "@", ",", ":", "\\", "\"", "[", "]", "?", "="} {
		value = strings.ReplaceAll(value, r, "")
	}

	return value
}
