package tablewriter

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter/tw"
)

// Recognized keys for the `tw` struct tag. Centralizing them here means the
// parser switch, and anything that documents or tests the tag vocabulary,
// all refer back to one definition instead of duplicating string literals.
//
// Note: Go struct tags are raw string literals, so a tag written on a field
// (e.g. `tw:"align=left"`) can never reference these constants directly —
// that part of the duplication is a language limitation, not a design
// choice. What these constants buy us is refactor-safety on the parsing
// side: renaming a key is a one-line change instead of a grep-and-pray.
const (
	twTagSkip = "-" // valid as the entire tag, or as a bare key/part

	twKeyName        = "name"
	twKeyAlign       = "align"
	twKeyHeaderAlign = "header_align"
	twKeyMaxWidth    = "max_width"
	twKeyPadLeft     = "pad_left"
	twKeyPadRight    = "pad_right"
	twKeyTrimSpace   = "trim_space"
	twKeyTrimTab     = "trim_tab"
	twKeyAutoFormat  = "auto_format"
	twKeyWrap        = "wrap"
)

// extractHeadersFromStruct is a thin wrapper around the unified extraction
// function below. It only cares about the header names.
func (t *Table) extractHeadersFromStruct(sample interface{}) []string {
	headers, _ := t.extractFieldsAndValuesFromStruct(sample)
	return headers
}

// extractFieldsAndValuesFromStruct is the single source of truth for struct
// reflection. It initiates recursive extraction starting at column offset 0.
func (t *Table) extractFieldsAndValuesFromStruct(sample interface{}) ([]string, []string) {
	return t.extractFieldsAndValuesFromStructWithOffset(sample, 0)
}

// extractFieldsAndValuesFromStructWithOffset recursively processes a struct,
// handling pointers and embedded structs, and natively parses the readable
// 'tw' struct tag. colOffset is the absolute column index of this struct's
// first field, used to keep embedded structs' per-column config (alignment,
// width, padding) addressed correctly within the parent's column layout.
func (t *Table) extractFieldsAndValuesFromStructWithOffset(sample interface{}, colOffset int) ([]string, []string) {
	v := reflect.ValueOf(sample)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil
	}

	typ := v.Type()
	headers := make([]string, 0, typ.NumField())
	values := make([]string, 0, typ.NumField())

	// Apply structural table configuration (alignment, width, padding, ...)
	// only on the first row, to avoid redundant work on every appended row
	// in Bulk(). Field tags are identical across rows of the same struct
	// type, so re-applying them on every row would be wasted reflection and
	// map writes, not a behavior change.
	isFirstPass := len(t.rows) == 0

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Handle embedded structs recursively, tracking absolute column offset
		if field.Anonymous {
			h, val := t.extractFieldsAndValuesFromStructWithOffset(fieldValue.Interface(), colOffset+len(headers))
			if h != nil {
				headers = append(headers, h...)
				values = append(values, val...)
			}
			continue
		}

		var legacyTagName string
		skipField := false
		twTag := field.Tag.Get("tw")

		// Check legacy priority tags (e.g., json, db) for fallback headers
		for _, tagKey := range t.config.Behavior.Structs.Tags {
			tagValue := field.Tag.Get(tagKey)
			if tagValue != "" {
				if tagValue == "-" {
					skipField = true
					break
				}
				legacyTagName = tagValue
				break
			}
		}

		// Determine base header name (Fallback: legacy tag -> field name).
		// Title-case it now, before any 'tw' name= override is applied below.
		// This matters for multi-word Go field names like "KeepMe": titling
		// it here turns it into "KEEPME" (one word, already uppercase), so
		// the render-time AutoFormat pass (which splits camelCase on case
		// transitions to insert spaces) has no lowercase-to-uppercase
		// transition left to find and won't turn it into "KEEP ME". An
		// explicit tw:"name=..." value is taken verbatim instead, after
		// this point, and intentionally skips this step to preserve casing.
		headerName := field.Name
		if legacyTagName != "" {
			headerName = strings.Split(legacyTagName, ",")[0]
		}
		headerName = tw.Title(headerName)

		// Absolute column index this field will occupy in the rendered table
		colIdx := colOffset + len(headers)

		// Parse the 'tw' struct tag schema
		if twTag != "" {
			if strings.TrimSpace(twTag) == twTagSkip {
				continue
			}

			parts := strings.Split(twTag, ",")

			// Pass 1: resolve skip first, before any part is allowed to
			// mutate table config. A tag like "align=left,-" must not leave
			// behind an alignment write for colIdx; if this field is going
			// to be dropped, the part that comes before "-" in the tag
			// string shouldn't matter, and a stale config write would leak
			// onto whichever field ends up occupying this column index.
			for _, p := range parts {
				kv := strings.SplitN(p, "=", 2)
				if strings.TrimSpace(kv[0]) == twTagSkip {
					skipField = true
					break
				}
			}

			// Pass 2: only now apply side effects, and only if the field
			// survived pass 1.
			if !skipField {
				for _, p := range parts {
					kv := strings.SplitN(p, "=", 2)
					key := strings.TrimSpace(kv[0])

					if len(kv) != 2 {
						continue
					}

					val := kv[1] // Do not TrimSpace val yet, to preserve padding spaces
					valTrimmed := strings.TrimSpace(val)

					switch key {
					case twKeyName:
						headerName = val // Use exact string, preserve casing

					// Per-Column Configurations
					case twKeyAlign:
						if isFirstPass {
							align := t.parseTwAlign(valTrimmed)
							t.setTwColumnAlignment(tw.Row, colIdx, align)
							t.setTwColumnAlignment(tw.Footer, colIdx, align)
						}
					case twKeyHeaderAlign:
						if isFirstPass {
							t.setTwColumnAlignment(tw.Header, colIdx, t.parseTwAlign(valTrimmed))
						}
					case twKeyMaxWidth:
						if isFirstPass {
							if mw, err := strconv.Atoi(valTrimmed); err == nil && mw > 0 {
								t.setTwColumnMaxWidth(colIdx, mw)
							}
						}
					case twKeyPadLeft:
						if isFirstPass {
							t.setTwColumnPadding(colIdx, val, true)
						}
					case twKeyPadRight:
						if isFirstPass {
							t.setTwColumnPadding(colIdx, val, false)
						}

					// Table-wide configurations. These cannot be scoped to a
					// single column: TrimSpace/TrimTab are a single flag on
					// t.config.Behavior used for every cell in the table
					// (header, row and footer alike), and Header.Formatting
					// .AutoFormat is one flag per section, not a per-column
					// array like Alignment.PerColumn is. Setting one of
					// these on a field changes that behavior for every
					// other column too, including ones processed earlier.
					// Whichever field sets a given key last "wins" for the
					// whole table. If genuinely independent per-column
					// control is needed, these settings don't belong in a
					// per-field tag — that would require making
					// CellFormatting per-column-aware first.
					case twKeyTrimSpace:
						if isFirstPass {
							t.config.Behavior.TrimSpace = t.parseTwState(valTrimmed)
						}
					case twKeyTrimTab:
						if isFirstPass {
							t.config.Behavior.TrimTab = t.parseTwState(valTrimmed)
						}
					case twKeyAutoFormat:
						if isFirstPass {
							t.config.Header.Formatting.AutoFormat = t.parseTwState(valTrimmed)
						}
					case twKeyWrap:
						if isFirstPass {
							switch valTrimmed {
							case "none":
								t.config.Row.Formatting.AutoWrap = tw.WrapNone
							case "normal":
								t.config.Row.Formatting.AutoWrap = tw.WrapNormal
							case "truncate":
								t.config.Row.Formatting.AutoWrap = tw.WrapTruncate
							case "break":
								t.config.Row.Formatting.AutoWrap = tw.WrapBreak
							}
						}
					}
				}
			}
		}

		if skipField {
			continue
		}

		headers = append(headers, headerName)

		// Extract value
		value := ""
		if !strings.Contains(legacyTagName, ",omitempty") || !fieldValue.IsZero() {
			value = t.convertToString(fieldValue.Interface())
		}
		values = append(values, value)
	}

	return headers, values
}

// Configuration Helper Methods

func (t *Table) parseTwAlign(val string) tw.Align {
	switch strings.ToLower(val) {
	case "center":
		return tw.AlignCenter
	case "left":
		return tw.AlignLeft
	case "right":
		return tw.AlignRight
	default:
		return tw.AlignDefault
	}
}

func (t *Table) parseTwState(val string) tw.State {
	if strings.ToLower(val) == "true" {
		return tw.On
	}
	return tw.Off
}

func (t *Table) setTwColumnAlignment(pos tw.Position, colIdx int, align tw.Align) {
	var alignConfig *[]tw.Align
	switch pos {
	case tw.Header:
		alignConfig = &t.config.Header.Alignment.PerColumn
	case tw.Row:
		alignConfig = &t.config.Row.Alignment.PerColumn
	case tw.Footer:
		alignConfig = &t.config.Footer.Alignment.PerColumn
	}

	if alignConfig != nil {
		if *alignConfig == nil {
			*alignConfig = make([]tw.Align, colIdx+1)
		}
		for len(*alignConfig) <= colIdx {
			*alignConfig = append(*alignConfig, tw.Skip)
		}
		(*alignConfig)[colIdx] = align
	}
}

// setTwColumnMaxWidth records a per-column max width for the header, row,
// and footer sections. This must operate on pointers to the actual config
// fields, not copies: tw.CellWidth.PerColumn is a map, and on a table that
// has never had a per-column width set before, that map is nil. Ranging
// over a []tw.CellWidth of *values* would initialize the map only on a
// local copy of the struct, which is discarded at the end of the loop body
// — the real t.config.*.ColMaxWidths.PerColumn would stay nil and the tag
// would silently do nothing. See setTwColumnPadding below for the same
// pattern done correctly, which this mirrors.
func (t *Table) setTwColumnMaxWidth(colIdx, mw int) {
	configs := []*tw.CellWidth{
		&t.config.Header.ColMaxWidths,
		&t.config.Row.ColMaxWidths,
		&t.config.Footer.ColMaxWidths,
	}
	for _, cfg := range configs {
		if cfg.PerColumn == nil {
			cfg.PerColumn = tw.NewMapper[int, int]()
		}
		cfg.PerColumn.Set(colIdx, mw)
	}
}

func (t *Table) setTwColumnPadding(colIdx int, padStr string, isLeft bool) {
	configs := []*tw.CellPadding{&t.config.Header.Padding, &t.config.Row.Padding, &t.config.Footer.Padding}

	for _, p := range configs {
		if p.PerColumn == nil {
			p.PerColumn = make([]tw.Padding, colIdx+1)
		}
		for len(p.PerColumn) <= colIdx {
			p.PerColumn = append(p.PerColumn, tw.Padding{})
		}

		if isLeft {
			p.PerColumn[colIdx].Left = padStr
		} else {
			p.PerColumn[colIdx].Right = padStr
		}
		p.PerColumn[colIdx].Overwrite = true
	}
}
