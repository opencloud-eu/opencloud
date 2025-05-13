package tw

// CellFormatting holds formatting options for table cells.
type CellFormatting struct {
	Alignment  Align // Text alignment within the cell (e.g., Left, Right, Center)
	AutoWrap   int   // Wrapping behavior (e.g., WrapTruncate, WrapNormal)
	AutoFormat bool  // Enables automatic formatting (e.g., title case for headers)
	// MaxWidth   int   // Maximum content width for the cell
	MergeMode int // Bitmask for merge behavior (e.g., MergeHorizontal, MergeVertical)
}

// CellPadding defines padding settings for table cells.
type CellPadding struct {
	Global    Padding   // Default padding applied to all cells
	PerColumn []Padding // Column-specific padding overrides
}

// CellFilter defines filtering functions for cell content.
type CellFilter struct {
	Global    func([]string) []string // Processes the entire row
	PerColumn []func(string) string   // Processes individual cells by column
}

// CellCallbacks holds callback functions for cell processing.
// Note: These are currently placeholders and not fully implemented.
type CellCallbacks struct {
	Global    func()   // Global callback applied to all cells
	PerColumn []func() // Column-specific callbacks
}

// CellConfig combines formatting, padding, and callback settings for a table section.
type CellConfig struct {
	Formatting   CellFormatting // Cell formatting options
	Padding      CellPadding    // Padding configuration
	Callbacks    CellCallbacks  // Callback functions (unused)
	Filter       CellFilter     // Function to filter cell content (renamed from Filter Filter)
	ColumnAligns []Align        // Per-column alignment overrides
	ColMaxWidths CellWidth      // Per-column maximum width overrides
}

type CellWidth struct {
	Global    int
	PerColumn Mapper[int, int]
}
