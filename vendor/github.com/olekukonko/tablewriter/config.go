// Package tablewriter provides functionality for creating and formatting tables with customizable configurations.
package tablewriter

import (
	"github.com/olekukonko/ll"             // Logging library for debug output
	"github.com/olekukonko/tablewriter/tw" // Tablewriter core types and utilities
	"io"                                   // Input/output interfaces
	"reflect"                              // Reflection for type handling
)

// Behavior defines table behavior settings that control features like auto-hiding columns and trimming spaces.
type Behavior struct {
	AutoHide  tw.State // Controls whether empty columns are automatically hidden (ignored in streaming mode)
	TrimSpace tw.State // Controls whether leading/trailing spaces are trimmed from cell content
}

// ColumnConfigBuilder is used to configure settings for a specific column across all table sections (header, row, footer).
type ColumnConfigBuilder struct {
	parent *ConfigBuilder // Reference to the parent ConfigBuilder for chaining
	col    int            // Index of the column being configured
}

// Build returns the parent ConfigBuilder to allow method chaining.
func (c *ColumnConfigBuilder) Build() *ConfigBuilder {
	return c.parent
}

// WithAlignment sets the text alignment for a specific column in the header section only.
// Invalid alignments are ignored, and the method returns the builder for chaining.
func (c *ColumnConfigBuilder) WithAlignment(align tw.Align) *ColumnConfigBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return c
	}
	// Ensure the ColumnAligns slice is large enough to accommodate the column index
	if len(c.parent.config.Header.ColumnAligns) <= c.col {
		newAligns := make([]tw.Align, c.col+1)
		copy(newAligns, c.parent.config.Header.ColumnAligns)
		c.parent.config.Header.ColumnAligns = newAligns
	}
	c.parent.config.Header.ColumnAligns[c.col] = align
	return c
}

// WithMaxWidth sets the maximum width for a specific column across all sections (header, row, footer).
// Negative widths are ignored, and the method returns the builder for chaining.
func (c *ColumnConfigBuilder) WithMaxWidth(width int) *ColumnConfigBuilder {
	if width < 0 {
		return c
	}
	// Initialize PerColumn maps if they don't exist
	if c.parent.config.Header.ColMaxWidths.PerColumn == nil {
		c.parent.config.Header.ColMaxWidths.PerColumn = make(map[int]int)
		c.parent.config.Row.ColMaxWidths.PerColumn = make(map[int]int)
		c.parent.config.Footer.ColMaxWidths.PerColumn = make(map[int]int)
	}
	c.parent.config.Header.ColMaxWidths.PerColumn[c.col] = width
	c.parent.config.Row.ColMaxWidths.PerColumn[c.col] = width
	c.parent.config.Footer.ColMaxWidths.PerColumn[c.col] = width
	return c
}

// Config represents the overall configuration for a table, including settings for header, rows, footer, and behavior.
type Config struct {
	MaxWidth int             // Maximum width of the entire table (0 for unlimited)
	Header   tw.CellConfig   // Configuration for the header section
	Row      tw.CellConfig   // Configuration for the row section
	Footer   tw.CellConfig   // Configuration for the footer section
	Debug    bool            // Enables debug logging when true
	Stream   tw.StreamConfig // Configuration specific to streaming mode
	Behavior Behavior        // Behavioral settings like auto-hiding and trimming
}

// ConfigBuilder provides a fluent interface for building a Config struct with both direct and nested configuration methods.
type ConfigBuilder struct {
	config Config // The configuration being built
}

// Build finalizes and returns the Config struct after all modifications.
func (b *ConfigBuilder) Build() Config {
	return b.config
}

// Footer returns a builder for advanced configuration of the footer section.
func (b *ConfigBuilder) Footer() *FooterConfigBuilder {
	return &FooterConfigBuilder{
		parent:  b,
		config:  &b.config.Footer,
		section: "footer",
	}
}

// ForColumn returns a builder for configuring a specific column across all sections.
func (b *ConfigBuilder) ForColumn(col int) *ColumnConfigBuilder {
	return &ColumnConfigBuilder{
		parent: b,
		col:    col,
	}
}

// Header returns a builder for advanced configuration of the header section.
func (b *ConfigBuilder) Header() *HeaderConfigBuilder {
	return &HeaderConfigBuilder{
		parent:  b,
		config:  &b.config.Header,
		section: "header",
	}
}

// Row returns a builder for advanced configuration of the row section.
func (b *ConfigBuilder) Row() *RowConfigBuilder {
	return &RowConfigBuilder{
		parent:  b,
		config:  &b.config.Row,
		section: "row",
	}
}

// WithAutoHide enables or disables automatic hiding of empty columns (ignored in streaming mode).
func (b *ConfigBuilder) WithAutoHide(state tw.State) *ConfigBuilder {
	b.config.Behavior.AutoHide = state
	return b
}

// WithDebug enables or disables debug logging for the table.
func (b *ConfigBuilder) WithDebug(debug bool) *ConfigBuilder {
	b.config.Debug = debug
	return b
}

// WithFooterAlignment sets the text alignment for all footer cells.
// Invalid alignments are ignored.
func (b *ConfigBuilder) WithFooterAlignment(align tw.Align) *ConfigBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return b
	}
	b.config.Footer.Formatting.Alignment = align
	return b
}

// WithFooterAutoFormat enables or disables automatic formatting (e.g., title case) for footer cells.
func (b *ConfigBuilder) WithFooterAutoFormat(autoFormat bool) *ConfigBuilder {
	b.config.Footer.Formatting.AutoFormat = autoFormat
	return b
}

// WithFooterAutoWrap sets the wrapping behavior for footer cells (e.g., truncate, normal, break).
// Invalid wrap modes are ignored.
func (b *ConfigBuilder) WithFooterAutoWrap(autoWrap int) *ConfigBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return b
	}
	b.config.Footer.Formatting.AutoWrap = autoWrap
	return b
}

// WithFooterGlobalPadding sets the global padding for all footer cells.
func (b *ConfigBuilder) WithFooterGlobalPadding(padding tw.Padding) *ConfigBuilder {
	b.config.Footer.Padding.Global = padding
	return b
}

// WithFooterMaxWidth sets the maximum content width for footer cells.
// Negative values are ignored.
func (b *ConfigBuilder) WithFooterMaxWidth(maxWidth int) *ConfigBuilder {
	if maxWidth < 0 {
		return b
	}
	b.config.Footer.ColMaxWidths.Global = maxWidth
	return b
}

// WithFooterMergeMode sets the merge behavior for footer cells (e.g., horizontal, hierarchical).
// Invalid merge modes are ignored.
func (b *ConfigBuilder) WithFooterMergeMode(mergeMode int) *ConfigBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return b
	}
	b.config.Footer.Formatting.MergeMode = mergeMode
	return b
}

// WithHeaderAlignment sets the text alignment for all header cells.
// Invalid alignments are ignored.
func (b *ConfigBuilder) WithHeaderAlignment(align tw.Align) *ConfigBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return b
	}
	b.config.Header.Formatting.Alignment = align
	return b
}

// WithHeaderAutoFormat enables or disables automatic formatting (e.g., title case) for header cells.
func (b *ConfigBuilder) WithHeaderAutoFormat(autoFormat bool) *ConfigBuilder {
	b.config.Header.Formatting.AutoFormat = autoFormat
	return b
}

// WithHeaderAutoWrap sets the wrapping behavior for header cells (e.g., truncate, normal).
// Invalid wrap modes are ignored.
func (b *ConfigBuilder) WithHeaderAutoWrap(autoWrap int) *ConfigBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return b
	}
	b.config.Header.Formatting.AutoWrap = autoWrap
	return b
}

// WithHeaderGlobalPadding sets the global padding for all header cells.
func (b *ConfigBuilder) WithHeaderGlobalPadding(padding tw.Padding) *ConfigBuilder {
	b.config.Header.Padding.Global = padding
	return b
}

// WithHeaderMaxWidth sets the maximum content width for header cells.
// Negative values are ignored.
func (b *ConfigBuilder) WithHeaderMaxWidth(maxWidth int) *ConfigBuilder {
	if maxWidth < 0 {
		return b
	}
	b.config.Header.ColMaxWidths.Global = maxWidth
	return b
}

// WithHeaderMergeMode sets the merge behavior for header cells (e.g., horizontal, vertical).
// Invalid merge modes are ignored.
func (b *ConfigBuilder) WithHeaderMergeMode(mergeMode int) *ConfigBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return b
	}
	b.config.Header.Formatting.MergeMode = mergeMode
	return b
}

// WithMaxWidth sets the maximum width for the entire table (0 means unlimited).
// Negative values are treated as 0.
func (b *ConfigBuilder) WithMaxWidth(width int) *ConfigBuilder {
	if width < 0 {
		b.config.MaxWidth = 0
	} else {
		b.config.MaxWidth = width
	}
	return b
}

// WithRowAlignment sets the text alignment for all row cells.
// Invalid alignments are ignored.
func (b *ConfigBuilder) WithRowAlignment(align tw.Align) *ConfigBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return b
	}
	b.config.Row.Formatting.Alignment = align
	return b
}

// WithRowAutoFormat enables or disables automatic formatting for row cells.
func (b *ConfigBuilder) WithRowAutoFormat(autoFormat bool) *ConfigBuilder {
	b.config.Row.Formatting.AutoFormat = autoFormat
	return b
}

// WithRowAutoWrap sets the wrapping behavior for row cells (e.g., truncate, normal).
// Invalid wrap modes are ignored.
func (b *ConfigBuilder) WithRowAutoWrap(autoWrap int) *ConfigBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return b
	}
	b.config.Row.Formatting.AutoWrap = autoWrap
	return b
}

// WithRowGlobalPadding sets the global padding for all row cells.
func (b *ConfigBuilder) WithRowGlobalPadding(padding tw.Padding) *ConfigBuilder {
	b.config.Row.Padding.Global = padding
	return b
}

// WithRowMaxWidth sets the maximum content width for row cells.
// Negative values are ignored.
func (b *ConfigBuilder) WithRowMaxWidth(maxWidth int) *ConfigBuilder {
	if maxWidth < 0 {
		return b
	}
	b.config.Row.ColMaxWidths.Global = maxWidth
	return b
}

// WithRowMergeMode sets the merge behavior for row cells (e.g., horizontal, hierarchical).
// Invalid merge modes are ignored.
func (b *ConfigBuilder) WithRowMergeMode(mergeMode int) *ConfigBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return b
	}
	b.config.Row.Formatting.MergeMode = mergeMode
	return b
}

// WithTrimSpace enables or disables automatic trimming of leading/trailing spaces.
// Ignored in streaming mode.
func (b *ConfigBuilder) WithTrimSpace(state tw.State) *ConfigBuilder {
	b.config.Behavior.TrimSpace = state
	return b
}

// FooterConfigBuilder provides advanced configuration options for the footer section.
type FooterConfigBuilder struct {
	parent  *ConfigBuilder // Reference to the parent ConfigBuilder
	config  *tw.CellConfig // Footer configuration being modified
	section string         // Section name for logging/debugging
}

// Build returns the parent ConfigBuilder for chaining.
func (f *FooterConfigBuilder) Build() *ConfigBuilder {
	return f.parent
}

// Formatting returns a builder for configuring footer formatting settings.
func (f *FooterConfigBuilder) Formatting() *FooterFormattingBuilder {
	return &FooterFormattingBuilder{
		parent:  f,
		config:  &f.config.Formatting,
		section: f.section,
	}
}

// Padding returns a builder for configuring footer padding settings.
func (f *FooterConfigBuilder) Padding() *FooterPaddingBuilder {
	return &FooterPaddingBuilder{
		parent:  f,
		config:  &f.config.Padding,
		section: f.section,
	}
}

// FooterFormattingBuilder configures formatting options for the footer section.
type FooterFormattingBuilder struct {
	parent  *FooterConfigBuilder // Reference to the parent FooterConfigBuilder
	config  *tw.CellFormatting   // Formatting configuration being modified
	section string               // Section name for logging/debugging
}

// Build returns the parent FooterConfigBuilder for chaining.
func (ff *FooterFormattingBuilder) Build() *FooterConfigBuilder {
	return ff.parent
}

// WithAlignment sets the text alignment for footer cells.
// Invalid alignments are ignored.
func (ff *FooterFormattingBuilder) WithAlignment(align tw.Align) *FooterFormattingBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return ff
	}
	ff.config.Alignment = align
	return ff
}

// WithAutoFormat enables or disables automatic formatting for footer cells.
func (ff *FooterFormattingBuilder) WithAutoFormat(autoFormat bool) *FooterFormattingBuilder {
	ff.config.AutoFormat = autoFormat
	return ff
}

// WithAutoWrap sets the wrapping behavior for footer cells.
// Invalid wrap modes are ignored.
func (ff *FooterFormattingBuilder) WithAutoWrap(autoWrap int) *FooterFormattingBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return ff
	}
	ff.config.AutoWrap = autoWrap
	return ff
}

// WithMaxWidth sets the maximum content width for footer cells.
// Negative values are ignored.
//func (ff *FooterFormattingBuilder) WithMaxWidth(maxWidth int) *FooterFormattingBuilder {
//	if maxWidth < 0 {
//		return ff
//	}
//	ff.config.Foo = maxWidth
//	return ff
//}

// WithNewarkMode sets the merge behavior for footer cells (likely a typo, should be WithMergeMode).
// Invalid merge modes are ignored.
func (ff *FooterFormattingBuilder) WithNewarkMode(mergeMode int) *FooterFormattingBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return ff
	}
	ff.config.MergeMode = mergeMode
	return ff
}

// FooterPaddingBuilder configures padding options for the footer section.
type FooterPaddingBuilder struct {
	parent  *FooterConfigBuilder // Reference to the parent FooterConfigBuilder
	config  *tw.CellPadding      // Padding configuration being modified
	section string               // Section name for logging/debugging
}

// AddColumnPadding adds padding for a specific column in the footer.
func (fp *FooterPaddingBuilder) AddColumnPadding(padding tw.Padding) *FooterPaddingBuilder {
	fp.config.PerColumn = append(fp.config.PerColumn, padding)
	return fp
}

// Build returns the parent FooterConfigBuilder for chaining.
func (fp *FooterPaddingBuilder) Build() *FooterConfigBuilder {
	return fp.parent
}

// WithGlobal sets the global padding for all footer cells.
func (fp *FooterPaddingBuilder) WithGlobal(padding tw.Padding) *FooterPaddingBuilder {
	fp.config.Global = padding
	return fp
}

// WithPerColumn sets per-column padding for the footer.
func (fp *FooterPaddingBuilder) WithPerColumn(padding []tw.Padding) *FooterPaddingBuilder {
	fp.config.PerColumn = padding
	return fp
}

// HeaderConfigBuilder provides advanced configuration options for the header section.
type HeaderConfigBuilder struct {
	parent  *ConfigBuilder // Reference to the parent ConfigBuilder
	config  *tw.CellConfig // Header configuration being modified
	section string         // Section name for logging/debugging
}

// Build returns the parent ConfigBuilder for chaining.
func (h *HeaderConfigBuilder) Build() *ConfigBuilder {
	return h.parent
}

// Formatting returns a builder for configuring header formatting settings.
func (h *HeaderConfigBuilder) Formatting() *HeaderFormattingBuilder {
	return &HeaderFormattingBuilder{
		parent:  h,
		config:  &h.config.Formatting,
		section: h.section,
	}
}

// Padding returns a builder for configuring header padding settings.
func (h *HeaderConfigBuilder) Padding() *HeaderPaddingBuilder {
	return &HeaderPaddingBuilder{
		parent:  h,
		config:  &h.config.Padding,
		section: h.section,
	}
}

// HeaderFormattingBuilder configures formatting options for the header section.
type HeaderFormattingBuilder struct {
	parent  *HeaderConfigBuilder // Reference to the parent HeaderConfigBuilder
	config  *tw.CellFormatting   // Formatting configuration being modified
	section string               // Section name for logging/debugging
}

// Build returns the parent HeaderConfigBuilder for chaining.
func (hf *HeaderFormattingBuilder) Build() *HeaderConfigBuilder {
	return hf.parent
}

// WithAlignment sets the text alignment for header cells.
// Invalid alignments are ignored.
func (hf *HeaderFormattingBuilder) WithAlignment(align tw.Align) *HeaderFormattingBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return hf
	}
	hf.config.Alignment = align
	return hf
}

// WithAutoFormat enables or disables automatic formatting for header cells.
func (hf *HeaderFormattingBuilder) WithAutoFormat(autoFormat bool) *HeaderFormattingBuilder {
	hf.config.AutoFormat = autoFormat
	return hf
}

// WithAutoWrap sets the wrapping behavior for header cells.
// Invalid wrap modes are ignored.
func (hf *HeaderFormattingBuilder) WithAutoWrap(autoWrap int) *HeaderFormattingBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return hf
	}
	hf.config.AutoWrap = autoWrap
	return hf
}

// WithMaxWidth sets the maximum content width for header cells.
// Negative values are ignored.
//func (hf *HeaderFormattingBuilder) WithMaxWidth(maxWidth int) *HeaderFormattingBuilder {
//	if maxWidth < 0 {
//		return hf
//	}
//	hf.config.MaxWidth = maxWidth
//	return hf
//}

// WithMergeMode sets the merge behavior for header cells.
// Invalid merge modes are ignored.
func (hf *HeaderFormattingBuilder) WithMergeMode(mergeMode int) *HeaderFormattingBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return hf
	}
	hf.config.MergeMode = mergeMode
	return hf
}

// HeaderPaddingBuilder configures padding options for the header section.
type HeaderPaddingBuilder struct {
	parent  *HeaderConfigBuilder // Reference to the parent HeaderConfigBuilder
	config  *tw.CellPadding      // Padding configuration being modified
	section string               // Section name for logging/debugging
}

// AddColumnPadding adds padding for a specific column in the header.
func (hp *HeaderPaddingBuilder) AddColumnPadding(padding tw.Padding) *HeaderPaddingBuilder {
	hp.config.PerColumn = append(hp.config.PerColumn, padding)
	return hp
}

// Build returns the parent HeaderConfigBuilder for chaining.
func (hp *HeaderPaddingBuilder) Build() *HeaderConfigBuilder {
	return hp.parent
}

// WithGlobal sets the global padding for all header cells.
func (hp *HeaderPaddingBuilder) WithGlobal(padding tw.Padding) *HeaderPaddingBuilder {
	hp.config.Global = padding
	return hp
}

// WithPerColumn sets per-column padding for the header.
func (hp *HeaderPaddingBuilder) WithPerColumn(padding []tw.Padding) *HeaderPaddingBuilder {
	hp.config.PerColumn = padding
	return hp
}

// Option defines a function type for configuring a Table instance.
type Option func(target *Table)

// RowConfigBuilder provides advanced configuration options for the row section.
type RowConfigBuilder struct {
	parent  *ConfigBuilder // Reference to the parent ConfigBuilder
	config  *tw.CellConfig // Row configuration being modified
	section string         // Section name for logging/debugging
}

// Build returns the parent ConfigBuilder for chaining.
func (r *RowConfigBuilder) Build() *ConfigBuilder {
	return r.parent
}

// Formatting returns a builder for configuring row formatting settings.
func (r *RowConfigBuilder) Formatting() *RowFormattingBuilder {
	return &RowFormattingBuilder{
		parent:  r,
		config:  &r.config.Formatting,
		section: r.section,
	}
}

// Padding returns a builder for configuring row padding settings.
func (r *RowConfigBuilder) Padding() *RowPaddingBuilder {
	return &RowPaddingBuilder{
		parent:  r,
		config:  &r.config.Padding,
		section: r.section,
	}
}

// RowFormattingBuilder configures formatting options for the row section.
type RowFormattingBuilder struct {
	parent  *RowConfigBuilder  // Reference to the parent RowConfigBuilder
	config  *tw.CellFormatting // Formatting configuration being modified
	section string             // Section name for logging/debugging
}

// Build returns the parent RowConfigBuilder for chaining.
func (rf *RowFormattingBuilder) Build() *RowConfigBuilder {
	return rf.parent
}

// WithAlignment sets the text alignment for row cells.
// Invalid alignments are ignored.
func (rf *RowFormattingBuilder) WithAlignment(align tw.Align) *RowFormattingBuilder {
	if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
		return rf
	}
	rf.config.Alignment = align
	return rf
}

// WithAutoFormat enables or disables automatic formatting for row cells.
func (rf *RowFormattingBuilder) WithAutoFormat(autoFormat bool) *RowFormattingBuilder {
	rf.config.AutoFormat = autoFormat
	return rf
}

// WithAutoWrap sets the wrapping behavior for row cells.
// Invalid wrap modes are ignored.
func (rf *RowFormattingBuilder) WithAutoWrap(autoWrap int) *RowFormattingBuilder {
	if autoWrap < tw.WrapNone || autoWrap > tw.WrapBreak {
		return rf
	}
	rf.config.AutoWrap = autoWrap
	return rf
}

// WithMergeMode sets the merge behavior for row cells.
// Invalid merge modes are ignored.
func (rf *RowFormattingBuilder) WithMergeMode(mergeMode int) *RowFormattingBuilder {
	if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
		return rf
	}
	rf.config.MergeMode = mergeMode
	return rf
}

// RowPaddingBuilder configures padding options for the row section.
type RowPaddingBuilder struct {
	parent  *RowConfigBuilder // Reference to the parent RowConfigBuilder
	config  *tw.CellPadding   // Padding configuration being modified
	section string            // Section name for logging/debugging
}

// AddColumnPadding adds padding for a specific column in the rows.
func (rp *RowPaddingBuilder) AddColumnPadding(padding tw.Padding) *RowPaddingBuilder {
	rp.config.PerColumn = append(rp.config.PerColumn, padding)
	return rp
}

// Build returns the parent RowConfigBuilder for chaining.
func (rp *RowPaddingBuilder) Build() *RowConfigBuilder {
	return rp.parent
}

// WithGlobal sets the global padding for all row cells.
func (rp *RowPaddingBuilder) WithGlobal(padding tw.Padding) *RowPaddingBuilder {
	rp.config.Global = padding
	return rp
}

// WithPerColumn sets per-column padding for the rows.
func (rp *RowPaddingBuilder) WithPerColumn(padding []tw.Padding) *RowPaddingBuilder {
	rp.config.PerColumn = padding
	return rp
}

// NewConfigBuilder creates a new ConfigBuilder initialized with default settings.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: defaultConfig(),
	}
}

// NewWriter creates a new table with default settings for backward compatibility.
// It logs the creation if debugging is enabled.
func NewWriter(w io.Writer) *Table {
	t := NewTable(w)
	if t.logger != nil {
		t.logger.Debug("NewWriter created buffered Table")
	}
	return t
}

// WithAutoHide enables or disables automatic hiding of columns with empty data rows.
// Logs the change if debugging is enabled.
func WithAutoHide(state tw.State) Option {
	return func(target *Table) {
		target.config.Behavior.AutoHide = state
		if target.logger != nil {
			target.logger.Debugf("Option: WithAutoHide applied to Table: %v", state)
		}
	}
}

// WithBorders sets the border configuration for the table and updates the renderer's configuration.
// Logs the change if debugging is enabled.
func WithBorders(borders tw.Border) Option {
	return func(target *Table) {
		if target.renderer != nil {
			cfg := target.renderer.Config()
			cfg.Borders = borders
			if target.logger != nil {
				target.logger.Debugf("Option: WithBorders applied to Table: %+v", borders)
			}
		}
	}
}

// WithColumnMax sets a global maximum column width for the table in streaming mode.
// Negative values are ignored, and the change is logged if debugging is enabled.
func WithColumnMax(width int) Option {
	return func(target *Table) {
		if width < 0 {
			return
		}
		target.config.Stream.Widths.Global = width
		if target.logger != nil {
			target.logger.Debugf("Option: WithColumnMax applied to Table: %v", width)
		}
	}
}

// WithTableMax sets a global maximum table width for the table
// Negative values are ignored, and the change is logged if debugging is enabled.
func WithTableMax(width int) Option {
	return func(target *Table) {
		if width < 0 {
			return
		}
		target.config.MaxWidth = width
		if target.logger != nil {
			target.logger.Debugf("Option: WithTableMax applied to Table: %v", width)
		}
	}
}

// WithColumnWidths sets per-column widths for the table in streaming mode.
// Negative widths are removed, and the change is logged if debugging is enabled.
func WithColumnWidths(widths map[int]int) Option {
	return func(target *Table) {
		for k, v := range widths {
			if v < 0 {
				delete(widths, k)
			}
		}
		target.config.Stream.Widths.PerColumn = widths
		if target.logger != nil {
			target.logger.Debugf("Option: WithColumnWidths applied to Table: %v", widths)
		}
	}
}

// WithConfig applies a custom configuration to the table by merging it with the default configuration.
func WithConfig(cfg Config) Option {
	return func(target *Table) {
		target.config = mergeConfig(defaultConfig(), cfg)
	}
}

// WithDebug enables or disables debug logging and adjusts the logger level accordingly.
// Logs the change if debugging is enabled.
func WithDebug(debug bool) Option {
	return func(target *Table) {
		target.config.Debug = debug
	}
}

// WithFooter sets the table footers by calling the Footer method.
func WithFooter(footers []string) Option {
	return func(target *Table) {
		target.Footer(footers)
	}
}

// WithFooterConfig applies a full footer configuration to the table.
// Logs the change if debugging is enabled.
func WithFooterConfig(config tw.CellConfig) Option {
	return func(target *Table) {
		target.config.Footer = config
		if target.logger != nil {
			target.logger.Debug("Option: WithFooterConfig applied to Table.")
		}
	}
}

// WithFooterMergeMode sets the merge mode for footer cells.
// Invalid merge modes are ignored, and the change is logged if debugging is enabled.
func WithFooterMergeMode(mergeMode int) Option {
	return func(target *Table) {
		if mergeMode < tw.MergeNone || mergeMode > tw.MergeHierarchical {
			return
		}
		target.config.Footer.Formatting.MergeMode = mergeMode
		if target.logger != nil {
			target.logger.Debugf("Option: WithFooterMergeMode applied to Table: %v", mergeMode)
		}
	}
}

// WithHeader sets the table headers by calling the Header method.
func WithHeader(headers []string) Option {
	return func(target *Table) {
		target.Header(headers)
	}
}

// WithHeaderAlignment sets the text alignment for header cells.
// Invalid alignments are ignored, and the change is logged if debugging is enabled.
func WithHeaderAlignment(align tw.Align) Option {
	return func(target *Table) {
		if align != tw.AlignLeft && align != tw.AlignRight && align != tw.AlignCenter && align != tw.AlignNone {
			return
		}
		target.config.Header.Formatting.Alignment = align
		if target.logger != nil {
			target.logger.Debugf("Option: WithHeaderAlignment applied to Table: %v", align)
		}
	}
}

// WithHeaderConfig applies a full header configuration to the table.
// Logs the change if debugging is enabled.
func WithHeaderConfig(config tw.CellConfig) Option {
	return func(target *Table) {
		target.config.Header = config
		if target.logger != nil {
			target.logger.Debug("Option: WithHeaderConfig applied to Table.")
		}
	}
}

// WithLogger sets a custom logger for the table and updates the renderer if present.
// Logs the change if debugging is enabled.
func WithLogger(logger *ll.Logger) Option {
	return func(target *Table) {
		target.logger = logger
		if target.logger != nil {
			target.logger.Debug("Option: WithLogger applied to Table.")
			if target.renderer != nil {
				target.renderer.Logger(target.logger)
			}
		}
	}
}

// WithRenderer sets a custom renderer for the table and attaches the logger if present.
// Logs the change if debugging is enabled.
func WithRenderer(f tw.Renderer) Option {
	return func(target *Table) {
		target.renderer = f
		if target.logger != nil {
			target.logger.Debugf("Option: WithRenderer applied to Table: %T", f)
			f.Logger(target.logger)
		}
	}
}

// WithRendererSettings updates the renderer's settings (e.g., separators, lines).
// Logs the change if debugging is enabled.
func WithRendererSettings(settings tw.Settings) Option {
	return func(target *Table) {
		if target.renderer != nil {
			cfg := target.renderer.Config()
			cfg.Settings = settings
			if target.logger != nil {
				target.logger.Debugf("Option: WithRendererSettings applied to Table: %+v", settings)
			}
		}
	}
}

// WithRowConfig applies a full row configuration to the table.
// Logs the change if debugging is enabled.
func WithRowConfig(config tw.CellConfig) Option {
	return func(target *Table) {
		target.config.Row = config
		if target.logger != nil {
			target.logger.Debug("Option: WithRowConfig applied to Table.")
		}
	}
}

// WithRowMaxWidth sets the maximum content width for row cells.
// Negative values are ignored, and the change is logged if debugging is enabled.
func WithRowMaxWidth(maxWidth int) Option {
	return func(target *Table) {
		if maxWidth < 0 {
			return
		}
		target.config.Row.ColMaxWidths.Global = maxWidth
		if target.logger != nil {
			target.logger.Debugf("Option: WithRowMaxWidth applied to Table: %v", maxWidth)
		}
	}
}

// WithStreaming applies a streaming configuration to the table by merging it with the existing configuration.
// Logs the change if debugging is enabled.
func WithStreaming(c tw.StreamConfig) Option {
	return func(target *Table) {
		target.config.Stream = mergeStreamConfig(target.config.Stream, c)
		if target.logger != nil {
			target.logger.Debug("Option: WithStreaming applied to Table.")
		}
	}
}

// WithStringer sets a custom stringer function for converting row data and clears the stringer cache.
// Logs the change if debugging is enabled.
func WithStringer(stringer interface{}) Option {
	return func(t *Table) {
		t.stringer = stringer
		t.stringerCacheMu.Lock()
		t.stringerCache = make(map[reflect.Type]reflect.Value)
		t.stringerCacheMu.Unlock()
		t.logger.Debug("Stringer updated, cache cleared")
	}
}

// WithStringerCache enables caching for the stringer function.
func WithStringerCache() Option {
	return func(t *Table) {
		t.stringerCacheEnabled = true
	}
}

// WithSymbols sets the symbols used for table drawing and updates the renderer's configuration.
// Logs the change if debugging is enabled.
func WithSymbols(symbols tw.Symbols) Option {
	return func(target *Table) {
		if target.renderer != nil {
			cfg := target.renderer.Config()
			cfg.Symbols = symbols
			if target.logger != nil {
				target.logger.Debug("Option: WithSymbols applied to Table.")
			}
		}
	}
}

// WithTrimSpace enables or disables automatic trimming of leading/trailing spaces.
// Logs the change if debugging is enabled.
func WithTrimSpace(state tw.State) Option {
	return func(target *Table) {
		target.config.Behavior.TrimSpace = state
		if target.logger != nil {
			target.logger.Debugf("Option: WithAutoHide applied to Table: %v", state)
		}
	}
}

// WithAlignment helps to set default alignments
func WithAlignment(alignment tw.Alignment) Option {
	return func(target *Table) {
		target.config.Header.ColumnAligns = alignment
		target.config.Row.ColumnAligns = alignment
		target.config.Footer.ColumnAligns = alignment
	}
}

// WithRendition allows updating the active renderer's rendition configuration
// by merging the provided rendition.
// If the renderer does not implement tw.Renditioning, a warning is logged.
func WithRendition(rendition tw.Rendition) Option {
	return func(target *Table) {
		if target.renderer == nil {
			target.logger.Warn("Option: WithRendition: No renderer set on table.")
			return
		}

		if ru, ok := target.renderer.(tw.Renditioning); ok {
			ru.Rendition(rendition)
			target.logger.Debugf("Option: WithRendition: Applied to renderer via Renditioning.SetRendition(): %+v", rendition)
		} else {
			target.logger.Warnf("Option: WithRendition: Current renderer type %T does not implement tw.Renditioning. Rendition may not be applied as expected.", target.renderer)
		}
	}
}

// defaultConfig returns a default Config with sensible settings for headers, rows, footers, and behavior.
func defaultConfig() Config {
	defaultPadding := tw.Padding{Left: tw.Space, Right: tw.Space, Top: tw.Empty, Bottom: tw.Empty}
	return Config{
		MaxWidth: 0,
		Header: tw.CellConfig{
			Formatting: tw.CellFormatting{
				AutoWrap:   tw.WrapTruncate,
				Alignment:  tw.AlignCenter,
				AutoFormat: true,
				MergeMode:  tw.MergeNone,
			},
			Padding: tw.CellPadding{
				Global: defaultPadding,
			},
		},
		Row: tw.CellConfig{
			Formatting: tw.CellFormatting{
				AutoWrap:   tw.WrapNormal,
				Alignment:  tw.AlignLeft,
				AutoFormat: false,
				MergeMode:  tw.MergeNone,
			},
			Padding: tw.CellPadding{
				Global: defaultPadding,
			},
		},
		Footer: tw.CellConfig{
			Formatting: tw.CellFormatting{
				AutoWrap:   tw.WrapNormal,
				Alignment:  tw.AlignRight,
				AutoFormat: false,
				MergeMode:  tw.MergeNone,
			},
			Padding: tw.CellPadding{
				Global: defaultPadding,
			},
		},
		Stream: tw.StreamConfig{},
		Debug:  false,
		Behavior: Behavior{
			AutoHide:  tw.Off,
			TrimSpace: tw.On,
		},
	}
}

// mergeCellConfig merges a source CellConfig into a destination CellConfig, prioritizing non-default source values.
// It handles deep merging for complex fields like padding and callbacks.
func mergeCellConfig(dst, src tw.CellConfig) tw.CellConfig {
	if src.Formatting.Alignment != tw.Empty {
		dst.Formatting.Alignment = src.Formatting.Alignment
	}
	if src.Formatting.AutoWrap != 0 {
		dst.Formatting.AutoWrap = src.Formatting.AutoWrap
	}
	if src.ColMaxWidths.Global != 0 {
		dst.ColMaxWidths.Global = src.ColMaxWidths.Global
	}
	if src.Formatting.MergeMode != 0 {
		dst.Formatting.MergeMode = src.Formatting.MergeMode
	}
	dst.Formatting.AutoFormat = src.Formatting.AutoFormat || dst.Formatting.AutoFormat
	if src.Padding.Global != (tw.Padding{}) {
		dst.Padding.Global = src.Padding.Global
	}
	if len(src.Padding.PerColumn) > 0 {
		if dst.Padding.PerColumn == nil {
			dst.Padding.PerColumn = make([]tw.Padding, len(src.Padding.PerColumn))
		} else if len(src.Padding.PerColumn) > len(dst.Padding.PerColumn) {
			dst.Padding.PerColumn = append(dst.Padding.PerColumn, make([]tw.Padding, len(src.Padding.PerColumn)-len(dst.Padding.PerColumn))...)
		}
		for i, pad := range src.Padding.PerColumn {
			if pad != (tw.Padding{}) {
				dst.Padding.PerColumn[i] = pad
			}
		}
	}
	if src.Callbacks.Global != nil {
		dst.Callbacks.Global = src.Callbacks.Global
	}
	if len(src.Callbacks.PerColumn) > 0 {
		if dst.Callbacks.PerColumn == nil {
			dst.Callbacks.PerColumn = make([]func(), len(src.Callbacks.PerColumn))
		} else if len(src.Callbacks.PerColumn) > len(dst.Callbacks.PerColumn) {
			dst.Callbacks.PerColumn = append(dst.Callbacks.PerColumn, make([]func(), len(src.Callbacks.PerColumn)-len(dst.Callbacks.PerColumn))...)
		}
		for i, cb := range src.Callbacks.PerColumn {
			if cb != nil {
				dst.Callbacks.PerColumn[i] = cb
			}
		}
	}
	if src.Filter.Global != nil {
		dst.Filter.Global = src.Filter.Global
	}
	if len(src.Filter.PerColumn) > 0 {
		if dst.Filter.PerColumn == nil {
			dst.Filter.PerColumn = make([]func(string) string, len(src.Filter.PerColumn))
		} else if len(src.Filter.PerColumn) > len(dst.Filter.PerColumn) {
			dst.Filter.PerColumn = append(dst.Filter.PerColumn, make([]func(string) string, len(src.Filter.PerColumn)-len(dst.Filter.PerColumn))...)
		}
		for i, filter := range src.Filter.PerColumn {
			if filter != nil {
				dst.Filter.PerColumn[i] = filter
			}
		}
	}
	if len(src.ColumnAligns) > 0 {
		if dst.ColumnAligns == nil {
			dst.ColumnAligns = make([]tw.Align, len(src.ColumnAligns))
		} else if len(src.ColumnAligns) > len(dst.ColumnAligns) {
			dst.ColumnAligns = append(dst.ColumnAligns, make([]tw.Align, len(src.ColumnAligns)-len(dst.ColumnAligns))...)
		}
		for i, align := range src.ColumnAligns {
			if align != tw.Empty && align != tw.Skip {
				dst.ColumnAligns[i] = align
			}
		}
	}
	if len(src.ColMaxWidths.PerColumn) > 0 {
		if dst.ColMaxWidths.PerColumn == nil {
			dst.ColMaxWidths.PerColumn = make(map[int]int)
		}
		for k, v := range src.ColMaxWidths.PerColumn {
			if v != 0 {
				dst.ColMaxWidths.PerColumn[k] = v
			}
		}
	}
	return dst
}

// mergeConfig merges a source Config into a destination Config, prioritizing non-default source values.
// It performs deep merging for complex types like Header, Row, Footer, and Stream.
func mergeConfig(dst, src Config) Config {
	if src.MaxWidth != 0 {
		dst.MaxWidth = src.MaxWidth
	}
	dst.Debug = src.Debug || dst.Debug
	dst.Behavior.AutoHide = src.Behavior.AutoHide
	dst.Behavior.TrimSpace = src.Behavior.TrimSpace
	dst.Header = mergeCellConfig(dst.Header, src.Header)
	dst.Row = mergeCellConfig(dst.Row, src.Row)
	dst.Footer = mergeCellConfig(dst.Footer, src.Footer)
	dst.Stream = mergeStreamConfig(dst.Stream, src.Stream)
	return dst
}

// mergeStreamConfig merges a source StreamConfig into a destination StreamConfig, prioritizing non-default source values.
func mergeStreamConfig(dst, src tw.StreamConfig) tw.StreamConfig {
	if src.Enable {
		dst.Enable = true
	}
	if src.Widths.Global != 0 {
		dst.Widths.Global = src.Widths.Global
	}
	if len(src.Widths.PerColumn) > 0 {
		if dst.Widths.PerColumn == nil {
			dst.Widths.PerColumn = make(map[int]int)
		}
		for k, v := range src.Widths.PerColumn {
			if v != 0 {
				dst.Widths.PerColumn[k] = v
			}
		}
	}
	return dst
}

// padLine pads a line to the specified column count by appending empty strings as needed.
func padLine(line []string, numCols int) []string {
	if len(line) >= numCols {
		return line
	}
	padded := make([]string, numCols)
	copy(padded, line)
	for i := len(line); i < numCols; i++ {
		padded[i] = tw.Empty
	}
	return padded
}
