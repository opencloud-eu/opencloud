package tw

// Padding defines custom padding characters for a cell
type Padding struct {
	Left   string
	Right  string
	Top    string
	Bottom string
}

// Symbols defines the interface for table border symbols
type Symbols interface {
	// Name returns the style name
	Name() string

	// Basic component symbols
	Center() string // Junction symbol (where lines cross)
	Row() string    // Horizontal line symbol
	Column() string // Vertical line symbol

	// Corner and junction symbols
	TopLeft() string     // LevelHeader-left corner
	TopMid() string      // LevelHeader junction
	TopRight() string    // LevelHeader-right corner
	MidLeft() string     // Left junction
	MidRight() string    // Right junction
	BottomLeft() string  // LevelFooter-left corner
	BottomMid() string   // LevelFooter junction
	BottomRight() string // LevelFooter-right corner

	// Optional header-specific symbols
	HeaderLeft() string
	HeaderMid() string
	HeaderRight() string
}

// BorderStyle defines different border styling options
type BorderStyle int

// Border style constants
const (
	StyleNone BorderStyle = iota
	StyleASCII
	StyleLight
	StyleHeavy
	StyleDouble
	StyleLightHeavy
	StyleHeavyLight
	StyleLightDouble
	StyleDoubleLight
	StyleRounded
	StyleMarkdown
	StyleGraphical
	StyleMerger
	StyleDefault
)

// String returns the string representation of a border style
func (s BorderStyle) String() string {
	return [...]string{
		"None",
		"SymbolASCII",
		"Light",
		"Heavy",
		"Double",
		"LightHeavy",
		"HeavyLight",
		"LightDouble",
		"DoubleLight",
		"Rounded",
		"SymbolMarkdown",
		"SymbolGraphical",
		"SymbolMerger",
	}[s]
}

// NewSymbols creates a new Symbols instance with the specified style
func NewSymbols(style BorderStyle) Symbols {
	switch style {
	case StyleASCII:
		return &SymbolASCII{}
	case StyleLight, StyleDefault:
		return &SymbolUnicode{
			row:    "─",
			column: "│",
			center: "┼",
			corners: [9]string{
				"┌", "┬", "┐",
				"├", "┼", "┤",
				"└", "┴", "┘",
			},
		}
	case StyleHeavy:
		return &SymbolUnicode{
			row:    "━",
			column: "┃",
			center: "╋",
			corners: [9]string{
				"┏", "┳", "┓",
				"┣", "╋", "┫",
				"┗", "┻", "┛",
			},
		}
	case StyleDouble:
		return &SymbolUnicode{
			row:    "═",
			column: "║",
			center: "╬",
			corners: [9]string{
				"╔", "╦", "╗",
				"╠", "╬", "╣",
				"╚", "╩", "╝",
			},
		}
	case StyleLightHeavy:
		return &SymbolUnicode{
			row:    "─",
			column: "┃",
			center: "╂",
			corners: [9]string{
				"┍", "┯", "┑",
				"┝", "╂", "┥",
				"┕", "┷", "┙",
			},
		}
	case StyleHeavyLight:
		return &SymbolUnicode{
			row:    "━",
			column: "│",
			center: "┿",
			corners: [9]string{
				"┎", "┰", "┒",
				"┠", "┿", "┨",
				"┖", "┸", "┚",
			},
		}
	case StyleLightDouble:
		return &SymbolUnicode{
			row:    "─",
			column: "║",
			center: "╫",
			corners: [9]string{
				"╓", "╥", "╖",
				"╟", "╫", "╢",
				"╙", "╨", "╜",
			},
		}
	case StyleDoubleLight:
		return &SymbolUnicode{
			row:    "═",
			column: "│",
			center: "╪",
			corners: [9]string{
				"╒", "╤", "╕",
				"╞", "╪", "╡",
				"╘", "╧", "╛",
			},
		}
	case StyleRounded:
		return &SymbolUnicode{
			row:    "─",
			column: "│",
			center: "┼",
			corners: [9]string{
				"╭", "┬", "╮",
				"├", "┼", "┤",
				"╰", "┴", "╯",
			},
		}
	case StyleMarkdown:
		return &SymbolMarkdown{}
	case StyleGraphical:
		return &SymbolGraphical{}
	case StyleMerger:
		// Private: Custom style for merged table rendering
		return &SymbolMerger{
			row:    "─", // Light row
			column: "│", // Light column
			center: "+", // Simplified junction
			corners: [9]string{
				"┌", "┬", "┐",
				"├", "┼", "┤",
				"└", "┴", "┘",
			},
		}
	default:
		return &SymbolNothing{}
	}
}

// Private: Style name constants
const (
	NameASCII     = "ascii"
	NameUnicode   = "unicode"
	NameNothing   = "nothing"
	NameMarkdown  = "markdown"
	NameGraphical = "graphical"
	NameMerger    = "merger"
)

// SymbolASCII provides basic SymbolASCII border symbols
type SymbolASCII struct{}

// SymbolASCII symbol methods
func (s *SymbolASCII) Name() string        { return NameASCII }
func (s *SymbolASCII) Center() string      { return "+" }
func (s *SymbolASCII) Row() string         { return "-" }
func (s *SymbolASCII) Column() string      { return "|" }
func (s *SymbolASCII) TopLeft() string     { return "+" }
func (s *SymbolASCII) TopMid() string      { return "+" }
func (s *SymbolASCII) TopRight() string    { return "+" }
func (s *SymbolASCII) MidLeft() string     { return "+" }
func (s *SymbolASCII) MidRight() string    { return "+" }
func (s *SymbolASCII) BottomLeft() string  { return "+" }
func (s *SymbolASCII) BottomMid() string   { return "+" }
func (s *SymbolASCII) BottomRight() string { return "+" }
func (s *SymbolASCII) HeaderLeft() string  { return "+" }
func (s *SymbolASCII) HeaderMid() string   { return "+" }
func (s *SymbolASCII) HeaderRight() string { return "+" }

// SymbolUnicode provides configurable SymbolUnicode border symbols
type SymbolUnicode struct {
	row     string
	column  string
	center  string
	corners [9]string // [topLeft, topMid, topRight, midLeft, center, midRight, bottomLeft, bottomMid, bottomRight]
}

// SymbolUnicode symbol methods
func (s *SymbolUnicode) Name() string        { return NameUnicode }
func (s *SymbolUnicode) Center() string      { return s.center }
func (s *SymbolUnicode) Row() string         { return s.row }
func (s *SymbolUnicode) Column() string      { return s.column }
func (s *SymbolUnicode) TopLeft() string     { return s.corners[0] }
func (s *SymbolUnicode) TopMid() string      { return s.corners[1] }
func (s *SymbolUnicode) TopRight() string    { return s.corners[2] }
func (s *SymbolUnicode) MidLeft() string     { return s.corners[3] }
func (s *SymbolUnicode) MidRight() string    { return s.corners[5] }
func (s *SymbolUnicode) BottomLeft() string  { return s.corners[6] }
func (s *SymbolUnicode) BottomMid() string   { return s.corners[7] }
func (s *SymbolUnicode) BottomRight() string { return s.corners[8] }
func (s *SymbolUnicode) HeaderLeft() string  { return s.MidLeft() }
func (s *SymbolUnicode) HeaderMid() string   { return s.Center() }
func (s *SymbolUnicode) HeaderRight() string { return s.MidRight() }

// SymbolMarkdown provides symbols for SymbolMarkdown-style tables
type SymbolMarkdown struct{}

// SymbolMarkdown symbol methods
func (s *SymbolMarkdown) Name() string        { return NameMarkdown }
func (s *SymbolMarkdown) Center() string      { return "|" }
func (s *SymbolMarkdown) Row() string         { return "-" }
func (s *SymbolMarkdown) Column() string      { return "|" }
func (s *SymbolMarkdown) TopLeft() string     { return "" }
func (s *SymbolMarkdown) TopMid() string      { return "" }
func (s *SymbolMarkdown) TopRight() string    { return "" }
func (s *SymbolMarkdown) MidLeft() string     { return "|" }
func (s *SymbolMarkdown) MidRight() string    { return "|" }
func (s *SymbolMarkdown) BottomLeft() string  { return "" }
func (s *SymbolMarkdown) BottomMid() string   { return "" }
func (s *SymbolMarkdown) BottomRight() string { return "" }
func (s *SymbolMarkdown) HeaderLeft() string  { return "|" }
func (s *SymbolMarkdown) HeaderMid() string   { return "|" }
func (s *SymbolMarkdown) HeaderRight() string { return "|" }

// SymbolNothing provides no border symbols (invisible borders)
type SymbolNothing struct{}

// SymbolNothing symbol methods
func (s *SymbolNothing) Name() string        { return NameNothing }
func (s *SymbolNothing) Center() string      { return "" }
func (s *SymbolNothing) Row() string         { return "" }
func (s *SymbolNothing) Column() string      { return "" }
func (s *SymbolNothing) TopLeft() string     { return "" }
func (s *SymbolNothing) TopMid() string      { return "" }
func (s *SymbolNothing) TopRight() string    { return "" }
func (s *SymbolNothing) MidLeft() string     { return "" }
func (s *SymbolNothing) MidRight() string    { return "" }
func (s *SymbolNothing) BottomLeft() string  { return "" }
func (s *SymbolNothing) BottomMid() string   { return "" }
func (s *SymbolNothing) BottomRight() string { return "" }
func (s *SymbolNothing) HeaderLeft() string  { return "" }
func (s *SymbolNothing) HeaderMid() string   { return "" }
func (s *SymbolNothing) HeaderRight() string { return "" }

// SymbolGraphical provides border symbols using emoji/emoticons
type SymbolGraphical struct{}

// SymbolGraphical symbol methods
func (s *SymbolGraphical) Name() string        { return NameGraphical }
func (s *SymbolGraphical) Center() string      { return "➕" }  // Cross
func (s *SymbolGraphical) Row() string         { return "➖" }  // Horizontal line
func (s *SymbolGraphical) Column() string      { return "➡️" } // Vertical line (using right arrow)
func (s *SymbolGraphical) TopLeft() string     { return "↖️" } // LevelHeader-left corner
func (s *SymbolGraphical) TopMid() string      { return "⬆️" } // LevelHeader junction
func (s *SymbolGraphical) TopRight() string    { return "↗️" } // LevelHeader-right corner
func (s *SymbolGraphical) MidLeft() string     { return "⬅️" } // Left junction
func (s *SymbolGraphical) MidRight() string    { return "➡️" } // Right junction
func (s *SymbolGraphical) BottomLeft() string  { return "↙️" } // LevelFooter-left corner
func (s *SymbolGraphical) BottomMid() string   { return "⬇️" } // LevelFooter junction
func (s *SymbolGraphical) BottomRight() string { return "↘️" } // LevelFooter-right corner
func (s *SymbolGraphical) HeaderLeft() string  { return "⏩" }  // Header left
func (s *SymbolGraphical) HeaderMid() string   { return "⏺️" } // Header middle
func (s *SymbolGraphical) HeaderRight() string { return "⏪" }  // Header right

// SymbolMerger provides custom symbols for merged table rendering
type SymbolMerger struct {
	row     string
	column  string
	center  string
	corners [9]string // [TL, TM, TR, ML, CenterIdx(unused), MR, BL, BM, BR]
}

// SymbolMerger symbol methods
func (s *SymbolMerger) Name() string        { return NameMerger }
func (s *SymbolMerger) Center() string      { return s.center } // Main crossing symbol
func (s *SymbolMerger) Row() string         { return s.row }
func (s *SymbolMerger) Column() string      { return s.column }
func (s *SymbolMerger) TopLeft() string     { return s.corners[0] }
func (s *SymbolMerger) TopMid() string      { return s.corners[1] } // LevelHeader junction
func (s *SymbolMerger) TopRight() string    { return s.corners[2] }
func (s *SymbolMerger) MidLeft() string     { return s.corners[3] } // Left junction
func (s *SymbolMerger) MidRight() string    { return s.corners[5] } // Right junction
func (s *SymbolMerger) BottomLeft() string  { return s.corners[6] }
func (s *SymbolMerger) BottomMid() string   { return s.corners[7] } // LevelFooter junction
func (s *SymbolMerger) BottomRight() string { return s.corners[8] }
func (s *SymbolMerger) HeaderLeft() string  { return s.MidLeft() }
func (s *SymbolMerger) HeaderMid() string   { return s.Center() }
func (s *SymbolMerger) HeaderRight() string { return s.MidRight() }

// SymbolCustom implements the Symbols interface with fully configurable symbols
type SymbolCustom struct {
	name        string
	center      string
	row         string
	column      string
	topLeft     string
	topMid      string
	topRight    string
	midLeft     string
	midRight    string
	bottomLeft  string
	bottomMid   string
	bottomRight string
	headerLeft  string
	headerMid   string
	headerRight string
}

// NewSymbolCustom creates a new customizable border style
func NewSymbolCustom(name string) *SymbolCustom {
	return &SymbolCustom{
		name:   name,
		center: "+",
		row:    "-",
		column: "|",
	}
}

// Implement all Symbols interface methods
func (c *SymbolCustom) Name() string        { return c.name }
func (c *SymbolCustom) Center() string      { return c.center }
func (c *SymbolCustom) Row() string         { return c.row }
func (c *SymbolCustom) Column() string      { return c.column }
func (c *SymbolCustom) TopLeft() string     { return c.topLeft }
func (c *SymbolCustom) TopMid() string      { return c.topMid }
func (c *SymbolCustom) TopRight() string    { return c.topRight }
func (c *SymbolCustom) MidLeft() string     { return c.midLeft }
func (c *SymbolCustom) MidRight() string    { return c.midRight }
func (c *SymbolCustom) BottomLeft() string  { return c.bottomLeft }
func (c *SymbolCustom) BottomMid() string   { return c.bottomMid }
func (c *SymbolCustom) BottomRight() string { return c.bottomRight }
func (c *SymbolCustom) HeaderLeft() string  { return c.headerLeft }
func (c *SymbolCustom) HeaderMid() string   { return c.headerMid }
func (c *SymbolCustom) HeaderRight() string { return c.headerRight }

// Builder methods for fluent configuration
func (c *SymbolCustom) WithCenter(s string) *SymbolCustom      { c.center = s; return c }
func (c *SymbolCustom) WithRow(s string) *SymbolCustom         { c.row = s; return c }
func (c *SymbolCustom) WithColumn(s string) *SymbolCustom      { c.column = s; return c }
func (c *SymbolCustom) WithTopLeft(s string) *SymbolCustom     { c.topLeft = s; return c }
func (c *SymbolCustom) WithTopMid(s string) *SymbolCustom      { c.topMid = s; return c }
func (c *SymbolCustom) WithTopRight(s string) *SymbolCustom    { c.topRight = s; return c }
func (c *SymbolCustom) WithMidLeft(s string) *SymbolCustom     { c.midLeft = s; return c }
func (c *SymbolCustom) WithMidRight(s string) *SymbolCustom    { c.midRight = s; return c }
func (c *SymbolCustom) WithBottomLeft(s string) *SymbolCustom  { c.bottomLeft = s; return c }
func (c *SymbolCustom) WithBottomMid(s string) *SymbolCustom   { c.bottomMid = s; return c }
func (c *SymbolCustom) WithBottomRight(s string) *SymbolCustom { c.bottomRight = s; return c }
func (c *SymbolCustom) WithHeaderLeft(s string) *SymbolCustom  { c.headerLeft = s; return c }
func (c *SymbolCustom) WithHeaderMid(s string) *SymbolCustom   { c.headerMid = s; return c }
func (c *SymbolCustom) WithHeaderRight(s string) *SymbolCustom { c.headerRight = s; return c }
