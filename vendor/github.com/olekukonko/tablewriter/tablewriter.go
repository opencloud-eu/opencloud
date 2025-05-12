package tablewriter

import (
	"bytes"
	"fmt"
	"github.com/olekukonko/errors"
	"github.com/olekukonko/ll"
	"github.com/olekukonko/ll/lh"
	"github.com/olekukonko/tablewriter/pkg/twwarp"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"io"
	"reflect"
	"strings"
	"sync"
)

// Table represents a table instance with content and rendering capabilities.
type Table struct {
	writer       io.Writer           // Destination for table output
	rows         [][][]string        // Row data, supporting multi-line cells
	headers      [][]string          // Header content
	footers      [][]string          // Footer content
	headerWidths tw.Mapper[int, int] // Computed widths for header columns
	rowWidths    tw.Mapper[int, int] // Computed widths for row columns
	footerWidths tw.Mapper[int, int] // Computed widths for footer columns
	renderer     tw.Renderer         // Engine for rendering the table
	config       Config              // Table configuration settings
	stringer     any                 // Function to convert rows to strings
	newLine      string              // Newline character (e.g., "\n")
	hasPrinted   bool                // Indicates if the table has been rendered
	logger       *ll.Logger          // Debug trace log
	trace        *bytes.Buffer       // Debug trace log

	// streaming
	streamWidths            tw.Mapper[int, int]           // Fixed column widths for streaming mode, calculated once
	streamFooterLines       [][]string                    // Processed footer lines for streaming, stored until Close().
	headerRendered          bool                          // Tracks if header has been rendered in streaming mode
	firstRowRendered        bool                          // Tracks if the first data row has been rendered in streaming mode
	lastRenderedLineContent []string                      // Content of the very last line rendered (for Previous context in streaming)
	lastRenderedMergeState  tw.Mapper[int, tw.MergeState] // Merge state of the very last line rendered (for Previous context in streaming)
	lastRenderedPosition    tw.Position                   // Position (Header/Row/Footer/Separator) of the last line rendered (for Previous context in streaming)
	streamNumCols           int                           // The derived number of columns in streaming mode
	streamRowCounter        int                           // Counter for rows rendered in streaming mode (0-indexed logical rows)

	// cache
	stringerCache        map[reflect.Type]reflect.Value // Cache for stringer reflection
	stringerCacheMu      sync.RWMutex                   // Mutex for thread-safe cache access
	stringerCacheEnabled bool                           // Flag to enable/disable caching
}

// renderContext holds the core state for rendering the table.
type renderContext struct {
	table           *Table                                      // Reference to the table instance
	renderer        tw.Renderer                                 // Renderer instance
	cfg             tw.Rendition                                // Renderer configuration
	numCols         int                                         // Total number of columns
	headerLines     [][]string                                  // Processed header lines
	rowLines        [][][]string                                // Processed row lines
	footerLines     [][]string                                  // Processed footer lines
	widths          tw.Mapper[tw.Position, tw.Mapper[int, int]] // Widths per section
	footerPrepared  bool                                        // Tracks if footer is prepared
	emptyColumns    []bool                                      // Tracks which original columns are empty (true if empty)
	visibleColCount int                                         // Count of columns that are NOT empty
	logger          *ll.Logger                                  // Debug trace log
}

// mergeContext holds state related to cell merging.
type mergeContext struct {
	headerMerges map[int]tw.MergeState        // Merge states for header columns
	rowMerges    []map[int]tw.MergeState      // Merge states for each row
	footerMerges map[int]tw.MergeState        // Merge states for footer columns
	horzMerges   map[tw.Position]map[int]bool // Tracks horizontal merges (unused)
}

// helperContext holds additional data for rendering helpers.
type helperContext struct {
	position tw.Position // Section being processed (Header, Row, Footer)
	rowIdx   int         // Row index within section
	lineIdx  int         // Line index within row
	location tw.Location // Boundary location (First, Middle, End)
	line     []string    // Current line content
}

// renderMergeResponse holds cell context data from rendering operations.
type renderMergeResponse struct {
	cells        map[int]tw.CellContext // Current line cells
	prevCells    map[int]tw.CellContext // Previous line cells
	nextCells    map[int]tw.CellContext // Next line cells
	location     tw.Location            // Determined Location for this line
	cellsContent []string
}

// NewTable creates a new table instance with specified writer and options.
// Parameters include writer for output and optional configuration options.
// Returns a pointer to the initialized Table instance.
func NewTable(w io.Writer, opts ...Option) *Table {
	t := &Table{
		writer:       w,
		headerWidths: tw.NewMapper[int, int](),
		rowWidths:    tw.NewMapper[int, int](),
		footerWidths: tw.NewMapper[int, int](),
		renderer:     renderer.NewBlueprint(),
		config:       defaultConfig(),
		newLine:      tw.NewLine,
		trace:        &bytes.Buffer{},

		// --- ADDED INITIALIZATION START ---
		streamWidths:           tw.NewMapper[int, int](), // Initialize empty mapper for streaming widths
		lastRenderedMergeState: tw.NewMapper[int, tw.MergeState](),
		headerRendered:         false,
		firstRowRendered:       false,
		lastRenderedPosition:   "",
		streamNumCols:          0,
		streamRowCounter:       0,

		//  Cache
		stringerCache:        make(map[reflect.Type]reflect.Value),
		stringerCacheEnabled: false, // Disabled by default
	}

	// add logger
	t.logger = ll.New("table").Handler(lh.NewTextHandler(t.trace))

	for _, opt := range opts {
		opt(t)
	}

	// force debugging mode if set
	// This should  be move away form WithDebug
	if t.config.Debug == true {
		t.logger.Enable()
	} else {
		t.logger.Disable()
	}

	// send logger to renderer
	// this will overwrite the default logger
	t.renderer.Logger(t.logger)
	t.logger.Info("Table initialized with %d options", len(opts))
	return t
}

// Append adds rows to the table, supporting various input types.
// Parameter rows accepts one or more rows, with stringer for custom types.
// Returns an error if any row fails to append.
func (t *Table) Append(rows ...interface{}) error {
	t.ensureInitialized() // Ensure initialized regardless of mode

	// Check if streaming is enabled and started
	if t.config.Stream.Enable && t.hasPrinted {
		t.logger.Debug("Append() called in streaming mode with %d rows", len(rows))
		for i, row := range rows {
			// Delegate rendering of each row to the streaming helper
			if err := t.streamAppendRow(row); err != nil {
				t.logger.Error("Error rendering streaming row %d: %v", i, err)
				// Decide error handling: stop appending? log and continue?
				// For now, stop and return the error.
				return fmt.Errorf("failed to stream append row %d: %w", i, err)
			}
		}
		t.logger.Debug("Append() completed in streaming mode, %d rows processed.", len(rows))
		return nil // Exit the function after handling streaming
	}

	// Existing batch rendering logic:
	t.logger.Debug("Starting batch Append operation with %d rows", len(rows))
	for i, row := range rows {
		if err := t.appendSingle(row); err != nil {
			t.logger.Debug("Append failed at index %d (batch mode): %v", i, err)
			return err
		}
	}
	t.logger.Debug("Append completed (batch mode), total rows: %d", len(t.rows))
	return nil
}

// Bulk adds multiple rows from a slice to the table (legacy method).
// Parameter rows must be a slice compatible with stringer or []string.
// Returns an error if the input is invalid or appending fails.
func (t *Table) Bulk(rows interface{}) error {
	t.logger.Debug("Starting Bulk operation")
	rv := reflect.ValueOf(rows)
	if rv.Kind() != reflect.Slice {
		err := errors.Newf("Bulk expects a slice, got %T", rows)
		t.logger.Debug("Bulk error: %v", err)
		return err
	}
	for i := 0; i < rv.Len(); i++ {
		row := rv.Index(i).Interface()
		t.logger.Debug("Processing bulk row %d: %v", i, row)
		if err := t.appendSingle(row); err != nil {
			t.logger.Debug("Bulk append failed at index %d: %v", i, err)
			return err
		}
	}
	t.logger.Debug("Bulk completed, processed %d rows", rv.Len())
	return nil
}

// Config returns the current table configuration.
// No parameters are required.
// Returns the Config struct with current settings.
func (t *Table) Config() Config {
	return t.config
}

// Configure updates the table's configuration using a provided function.
// Parameter fn is a function that modifies the Config struct.
// Returns the Table instance for method chaining.
func (t *Table) Configure(fn func(*Config)) *Table {
	fn(&t.config)
	return t
}

// Debug retrieves the accumulated debug trace logs.
// No parameters are required.
// Returns a slice of debug messages including renderer logs.
func (t *Table) Debug() *bytes.Buffer {
	return t.trace
}

// Footer sets the table's footer content, padding to match column count.
// Parameter footers is a slice of strings for footer content.
// No return value.
// Footer sets the table's footer content.
// Parameter footers is a slice of strings for footer content.
// In streaming mode, this processes and stores the footer for rendering by Close().
func (t *Table) Footer(elements ...any) {
	t.ensureInitialized()
	t.logger.Debug("Footer() method called with raw variadic elements: %v (len %d). Streaming: %v, Started: %v",
		elements, len(elements), t.config.Stream.Enable, t.hasPrinted)

	if t.config.Stream.Enable && t.hasPrinted {
		// --- Streaming Path ---
		actualCellsToProcess := t.processVariadicElements(elements)
		footersAsStrings, err := t.convertCellsToStrings(actualCellsToProcess, t.config.Footer)
		if err != nil {
			t.logger.Error("Footer(): Failed to convert footer elements to strings for streaming: %v", err)
			footersAsStrings = []string{} // Use empty on error
		}
		errStream := t.streamStoreFooter(footersAsStrings) // streamStoreFooter handles padding to streamNumCols internally
		if errStream != nil {
			t.logger.Error("Error processing streaming footer: %v", errStream)
		}
		return
	}

	// --- Batch Path ---
	actualCellsToProcess := t.processVariadicElements(elements)
	t.logger.Debug("Footer() (Batch): Effective cells to process: %v", actualCellsToProcess)

	footersAsStrings, err := t.convertCellsToStrings(actualCellsToProcess, t.config.Footer)
	if err != nil {
		t.logger.Error("Footer() (Batch): Failed to convert to strings: %v", err)
		t.footers = [][]string{} // Set to empty on error
		return
	}

	preparedFooterLines := t.prepareContent(footersAsStrings, t.config.Footer)
	t.footers = preparedFooterLines // Store directly. Padding to t.maxColumns() will happen in prepareContexts.

	t.logger.Debug("Footer set (batch mode), lines stored: %d. First line if exists: %v",
		len(t.footers), func() []string {
			if len(t.footers) > 0 {
				return t.footers[0]
			} else {
				return nil
			}
		}())
}

// Render triggers the table rendering process to the configured writer.
// No parameters are required.
// Returns an error if rendering fails.
func (t *Table) Render() error {
	return t.render()
}

// render generates the table output using the configured renderer.
// No parameters are required.
// Returns an error if rendering fails in any section.
func (t *Table) render() error {
	t.ensureInitialized() // This is needed in both modes, good place to keep it.

	// --- ADDED LOGIC START ---
	// If streaming is enabled, this batch rendering path should not be used.
	// The Start/Append/Close methods handle streaming.
	if t.config.Stream.Enable {
		t.logger.Warn("Internal render() method called when streaming is enabled. This indicates incorrect usage of Render() instead of Start/Append/Close.")
		// It's safer to just return an error here, as the table state is likely not set up for batch.
		return errors.New("internal batch render called in streaming mode")
	}
	// --- ADDED LOGIC END ---

	// The rest of the original batch rendering logic follows:
	ctx, mctx, err := t.prepareContexts() // This is batch-specific
	if err != nil {
		t.logger.Error("prepareContexts failed: %v", err)
		return err
	}

	ctx.logger.Debug("Calling renderer Start()")
	// In batch mode, renderer.Start is called here.
	if err := ctx.renderer.Start(t.writer); err != nil {
		ctx.logger.Debug("Renderer Start() error: %v", err)
		return fmt.Errorf("renderer start failed: %w", err)
	}

	renderError := false
	// Render sections (Header, Row, Footer) sequentially using batch context
	for _, renderFn := range []func(*renderContext, *mergeContext) error{
		t.renderHeader,
		t.renderRow,
		t.renderFooter,
	} {
		if err := renderFn(ctx, mctx); err != nil {
			ctx.logger.Error("Renderer section error: %v", err)
			renderError = true
		}
	}

	ctx.logger.Debug("Calling renderer Close()")
	// In batch mode, renderer.Close is called here.
	closeErr := ctx.renderer.Close(t.writer)
	if closeErr != nil {
		ctx.logger.Error("Renderer Close() error: %v", closeErr)
		if !renderError {
			return fmt.Errorf("renderer close failed: %w", closeErr)
		}
	}

	if renderError {
		return errors.New("table rendering failed in one or more sections")
	}

	// This flag needs careful consideration with streaming.
	// For now, it marks that rendering finished (batch mode).
	t.hasPrinted = true
	ctx.logger.Debug("Render completed")
	return nil
}

// appendSingle adds a single row to the table's row data.
// Parameter row is the data to append, converted via stringer if needed.
// Returns an error if conversion or appending fails.
func (t *Table) appendSingle(row interface{}) error {
	t.ensureInitialized() // Already here

	if t.config.Stream.Enable && t.hasPrinted { // If streaming is active
		t.logger.Debug("appendSingle: Dispatching to streamAppendRow for row: %v", row)
		return t.streamAppendRow(row) // Call the streaming render function
	}
	// Existing batch logic:
	t.logger.Debug("appendSingle: Processing for batch mode, row: %v", row)
	// toStringLines now uses the new convertCellsToStrings internally, then prepareContent.
	// This is fine for batch.
	lines, err := t.toStringLines(row, t.config.Row)
	if err != nil {
		t.logger.Debug("Error in toStringLines (batch mode): %v", err)
		return err
	}
	t.rows = append(t.rows, lines) // Add to batch storage
	t.logger.Debug("Row appended to batch t.rows, total batch rows: %d", len(t.rows))
	return nil
}

// buildAligns constructs a map of column alignments from configuration.
// Parameter config provides alignment settings for the section.
// Returns a map of column indices to alignment settings.
func (t *Table) buildAligns(config tw.CellConfig) map[int]tw.Align {
	t.logger.Debug("buildAligns INPUT: config.Formatting.Alignment=%s, config.ColumnAligns=%v", config.Formatting.Alignment, config.ColumnAligns)
	numColsToUse := t.getNumColsToUse()
	colAlignsResult := make(map[int]tw.Align)
	for i := 0; i < numColsToUse; i++ {
		currentAlign := config.Formatting.Alignment
		if i < len(config.ColumnAligns) {
			colSpecificAlign := config.ColumnAligns[i]
			if colSpecificAlign != tw.Empty && colSpecificAlign != tw.Skip {
				currentAlign = colSpecificAlign
			}
		}
		colAlignsResult[i] = currentAlign
	}
	t.logger.Debug("Aligns built: %v (length %d)", colAlignsResult, len(colAlignsResult))
	return colAlignsResult
}

// buildPadding constructs a map of column padding settings from configuration.
// Parameter padding provides padding settings for the section.
// Returns a map of column indices to padding settings.
func (t *Table) buildPadding(padding tw.CellPadding) map[int]tw.Padding {
	numColsToUse := t.getNumColsToUse()
	colPadding := make(map[int]tw.Padding)
	for i := 0; i < numColsToUse; i++ {
		if i < len(padding.PerColumn) && padding.PerColumn[i] != (tw.Padding{}) {
			colPadding[i] = padding.PerColumn[i]
		} else {
			colPadding[i] = padding.Global
		}
	}
	t.logger.Debug("Padding built: %v (length %d)", colPadding, len(colPadding))
	return colPadding
}

// ensureInitialized initializes required fields before use.
// No parameters are required.
// No return value.
func (t *Table) ensureInitialized() {
	if t.headerWidths == nil {
		t.headerWidths = tw.NewMapper[int, int]()
	}
	if t.rowWidths == nil {
		t.rowWidths = tw.NewMapper[int, int]()
	}
	if t.footerWidths == nil {
		t.footerWidths = tw.NewMapper[int, int]()
	}
	if t.renderer == nil {
		t.renderer = renderer.NewBlueprint()
	}
	t.logger.Debug("ensureInitialized called")
}

// finalizeHierarchicalMergeBlock sets Span and End for hierarchical merges.
// Parameters include ctx, mctx, col, startRow, and endRow.
// No return value.
func (t *Table) finalizeHierarchicalMergeBlock(ctx *renderContext, mctx *mergeContext, col, startRow, endRow int) {
	if endRow < startRow {
		ctx.logger.Debug("Hierarchical merge FINALIZE WARNING: Invalid block col %d, start %d > end %d", col, startRow, endRow)
		return
	}
	if startRow < 0 || endRow < 0 {
		ctx.logger.Debug("Hierarchical merge FINALIZE WARNING: Negative row indices col %d, start %d, end %d", col, startRow, endRow)
		return
	}
	requiredLen := endRow + 1
	if requiredLen > len(mctx.rowMerges) {
		ctx.logger.Debug("Hierarchical merge FINALIZE WARNING: rowMerges slice too short (len %d) for endRow %d", len(mctx.rowMerges), endRow)
		return
	}
	if mctx.rowMerges[startRow] == nil {
		mctx.rowMerges[startRow] = make(map[int]tw.MergeState)
	}
	if mctx.rowMerges[endRow] == nil {
		mctx.rowMerges[endRow] = make(map[int]tw.MergeState)
	}

	finalSpan := (endRow - startRow) + 1
	ctx.logger.Debug("Finalizing H-merge block: col=%d, startRow=%d, endRow=%d, span=%d", col, startRow, endRow, finalSpan)

	startState := mctx.rowMerges[startRow][col]
	if startState.Hierarchical.Present && startState.Hierarchical.Start {
		startState.Hierarchical.Span = finalSpan
		startState.Hierarchical.End = finalSpan == 1
		mctx.rowMerges[startRow][col] = startState
		ctx.logger.Debug(" -> Updated start state: %+v", startState.Hierarchical)
	} else {
		ctx.logger.Debug("Hierarchical merge FINALIZE WARNING: col %d, startRow %d was not marked as Present/Start? Current state: %+v. Attempting recovery.", col, startRow, startState.Hierarchical)
		startState.Hierarchical.Present = true
		startState.Hierarchical.Start = true
		startState.Hierarchical.Span = finalSpan
		startState.Hierarchical.End = (finalSpan == 1)
		mctx.rowMerges[startRow][col] = startState
	}

	if endRow > startRow {
		endState := mctx.rowMerges[endRow][col]
		if endState.Hierarchical.Present && !endState.Hierarchical.Start {
			endState.Hierarchical.End = true
			endState.Hierarchical.Span = finalSpan
			mctx.rowMerges[endRow][col] = endState
			ctx.logger.Debug(" -> Updated end state: %+v", endState.Hierarchical)
		} else {
			ctx.logger.Debug("Hierarchical merge FINALIZE WARNING: col %d, endRow %d was not marked as Present/Continuation? Current state: %+v. Attempting recovery.", col, endRow, endState.Hierarchical)
			endState.Hierarchical.Present = true
			endState.Hierarchical.Start = false
			endState.Hierarchical.End = true
			endState.Hierarchical.Span = finalSpan
			mctx.rowMerges[endRow][col] = endState
		}
	} else {
		ctx.logger.Debug(" -> Span is 1, startRow is also endRow.")
	}
}

// getLevel maps a position to its rendering level.
// Parameter position specifies the section (Header, Row, Footer).
// Returns the corresponding tw.Level (Header, Body, Footer).
func (t *Table) getLevel(position tw.Position) tw.Level {
	switch position {
	case tw.Header:
		return tw.LevelHeader
	case tw.Row:
		return tw.LevelBody
	case tw.Footer:
		return tw.LevelFooter
	default:
		return tw.LevelBody
	}
}

// hasFooterElements checks if the footer has renderable elements.
// No parameters are required.
// Returns true if footer has content or padding, false otherwise.
func (t *Table) hasFooterElements() bool {
	hasContent := len(t.footers) > 0
	hasTopPadding := t.config.Footer.Padding.Global.Top != tw.Empty
	hasBottomPaddingConfig := t.config.Footer.Padding.Global.Bottom != tw.Empty || t.hasPerColumnBottomPadding()
	return hasContent || hasTopPadding || hasBottomPaddingConfig
}

// hasPerColumnBottomPadding checks for per-column bottom padding in footer.
// No parameters are required.
// Returns true if any per-column bottom padding is defined.
func (t *Table) hasPerColumnBottomPadding() bool {
	if t.config.Footer.Padding.PerColumn == nil {
		return false
	}
	for _, pad := range t.config.Footer.Padding.PerColumn {
		if pad.Bottom != tw.Empty {
			return true
		}
	}
	return false
}

// Header sets the table's header content, padding to match column count.
// Parameter elements is a slice of strings for header content.
// No return value.
// In streaming mode, this processes and renders the header immediately.
func (t *Table) Header(elements ...any) {
	t.ensureInitialized()
	t.logger.Debug("Header() method called with raw variadic elements: %v (len %d). Streaming: %v, Started: %v",
		elements, len(elements), t.config.Stream.Enable, t.hasPrinted)

	if t.config.Stream.Enable && t.hasPrinted {
		// --- Streaming Path ---
		actualCellsToProcess := t.processVariadicElements(elements)
		headersAsStrings, err := t.convertCellsToStrings(actualCellsToProcess, t.config.Header)
		if err != nil {
			t.logger.Error("Header(): Failed to convert header elements to strings for streaming: %v", err)
			headersAsStrings = []string{} // Use empty on error
		}
		errStream := t.streamRenderHeader(headersAsStrings) // streamRenderHeader handles padding to streamNumCols internally
		if errStream != nil {
			t.logger.Error("Error rendering streaming header: %v", errStream)
		}
		return
	}

	// --- Batch Path ---
	actualCellsToProcess := t.processVariadicElements(elements)
	t.logger.Debug("Header() (Batch): Effective cells to process: %v", actualCellsToProcess)

	headersAsStrings, err := t.convertCellsToStrings(actualCellsToProcess, t.config.Header)
	if err != nil {
		t.logger.Error("Header() (Batch): Failed to convert to strings: %v", err)
		t.headers = [][]string{} // Set to empty on error
		return
	}

	// prepareContent uses t.config.Header for AutoFormat and MaxWidth constraints.
	// It processes based on the number of columns in headersAsStrings.
	preparedHeaderLines := t.prepareContent(headersAsStrings, t.config.Header)
	t.headers = preparedHeaderLines // Store directly. Padding to t.maxColumns() will happen in prepareContexts.

	t.logger.Debug("Header set (batch mode), lines stored: %d. First line if exists: %v",
		len(t.headers), func() []string {
			if len(t.headers) > 0 {
				return t.headers[0]
			} else {
				return nil
			}
		}())
}

// Logger retrieves the table's logger instance.
// No parameters are required.
// Returns the ll.Logger instance used for debug tracing.
func (t *Table) Logger() *ll.Logger {
	return t.logger
}

// Renderer retrieves the current renderer instance used by the table.
// No parameters are required.
// Returns the tw.Renderer interface instance.
func (t *Table) Renderer() tw.Renderer {
	t.logger.Debug("Renderer requested")
	return t.renderer
}

// maxColumns calculates the maximum column count across sections.
// No parameters are required.
// Returns the highest number of columns found.
func (t *Table) maxColumns() int {
	m := 0
	if len(t.headers) > 0 && len(t.headers[0]) > m {
		m = len(t.headers[0])
	}
	for _, row := range t.rows {
		if len(row) > 0 && len(row[0]) > m {
			m = len(row[0])
		}
	}
	if len(t.footers) > 0 && len(t.footers[0]) > m {
		m = len(t.footers[0])
	}
	t.logger.Debug("Max columns: %d", m)
	return m
}

// prepareContent processes cell content with formatting and wrapping.
// Parameters include cells to process and config for formatting rules.
// Returns a slice of string slices representing processed lines.
func (t *Table) prepareContent(cells []string, config tw.CellConfig) [][]string {
	isStreaming := t.config.Stream.Enable && t.hasPrinted
	t.logger.Debug("prepareContent: Processing cells=%v (streaming: %v)", cells, isStreaming)
	initialInputCellCount := len(cells)
	result := make([][]string, 0)

	effectiveNumCols := initialInputCellCount
	if isStreaming {
		if t.streamNumCols > 0 {
			effectiveNumCols = t.streamNumCols
			t.logger.Debug("prepareContent: Streaming mode, using fixed streamNumCols: %d", effectiveNumCols)
			if len(cells) != effectiveNumCols {
				t.logger.Warn("prepareContent: Streaming mode, input cell count (%d) does not match streamNumCols (%d). Input cells will be padded/truncated.", len(cells), effectiveNumCols)
				if len(cells) < effectiveNumCols {
					paddedCells := make([]string, effectiveNumCols)
					copy(paddedCells, cells)
					for i := len(cells); i < effectiveNumCols; i++ {
						paddedCells[i] = tw.Empty
					}
					cells = paddedCells
				} else if len(cells) > effectiveNumCols {
					cells = cells[:effectiveNumCols]
				}
			}
		} else {
			t.logger.Warn("prepareContent: Streaming mode enabled but streamNumCols is 0. Using input cell count %d. Stream widths may not be available.", effectiveNumCols)
		}
	}

	for i := 0; i < effectiveNumCols; i++ {
		cellContent := ""
		if i < len(cells) {
			cellContent = cells[i]
		} else {
			cellContent = tw.Empty
		}

		if t.config.Behavior.TrimSpace.Enabled() { // Access the global table config for TrimSpace behavior
			cellContent = strings.TrimSpace(cellContent)
		}

		colPad := config.Padding.Global
		if i < len(config.Padding.PerColumn) && config.Padding.PerColumn[i] != (tw.Padding{}) {
			colPad = config.Padding.PerColumn[i]
		}
		padLeftWidth := tw.DisplayWidth(colPad.Left)
		padRightWidth := tw.DisplayWidth(colPad.Right)

		effectiveContentMaxWidth := t.calculateContentMaxWidth(i, config, padLeftWidth, padRightWidth, isStreaming)

		if config.Formatting.AutoFormat {
			cellContent = tw.Title(strings.Join(tw.SplitCamelCase(cellContent), tw.Space))
		}

		lines := strings.Split(cellContent, "\n")
		finalLinesForCell := make([]string, 0)
		for _, line := range lines {
			if effectiveContentMaxWidth > 0 {
				switch config.Formatting.AutoWrap {
				case tw.WrapNormal:
					wrapped, _ := twwarp.WrapString(line, effectiveContentMaxWidth)
					finalLinesForCell = append(finalLinesForCell, wrapped...)
				case tw.WrapTruncate:
					if tw.DisplayWidth(line) > effectiveContentMaxWidth {
						ellipsisWidth := tw.DisplayWidth(tw.CharEllipsis)
						if effectiveContentMaxWidth >= ellipsisWidth {
							finalLinesForCell = append(finalLinesForCell, tw.TruncateString(line, effectiveContentMaxWidth-ellipsisWidth, tw.CharEllipsis))
						} else {
							finalLinesForCell = append(finalLinesForCell, tw.TruncateString(line, effectiveContentMaxWidth, ""))
						}
					} else {
						finalLinesForCell = append(finalLinesForCell, line)
					}
				case tw.WrapBreak:
					wrapped := make([]string, 0)
					currentLine := line
					for tw.DisplayWidth(currentLine) > effectiveContentMaxWidth {
						breakPoint := tw.BreakPoint(currentLine, effectiveContentMaxWidth)
						if breakPoint <= 0 {
							t.logger.Warn("prepareContent: WrapBreak - BreakPoint <= 0 for line '%s' at width %d. Attempting manual break.", currentLine, effectiveContentMaxWidth)
							runes := []rune(currentLine)
							actualBreakRuneCount := 0
							tempWidth := 0
							for charIdx, r := range currentLine {
								runeStr := string(r)
								rw := tw.DisplayWidth(runeStr)
								if tempWidth+rw > effectiveContentMaxWidth && charIdx > 0 {
									break
								}
								tempWidth += rw
								actualBreakRuneCount = charIdx + 1
								if tempWidth >= effectiveContentMaxWidth && charIdx == 0 {
									break
								}
							}
							if actualBreakRuneCount == 0 && len(runes) > 0 {
								actualBreakRuneCount = 1
							}

							if actualBreakRuneCount > 0 && actualBreakRuneCount <= len(runes) {
								wrapped = append(wrapped, string(runes[:actualBreakRuneCount])+tw.CharBreak)
								currentLine = string(runes[actualBreakRuneCount:])
							} else {
								if tw.DisplayWidth(currentLine) > 0 {
									wrapped = append(wrapped, currentLine)
									currentLine = ""
								}
								break
							}
						} else {
							runes := []rune(currentLine)
							if breakPoint <= len(runes) {
								wrapped = append(wrapped, string(runes[:breakPoint])+tw.CharBreak)
								currentLine = string(runes[breakPoint:])
							} else {
								t.logger.Warn("prepareContent: WrapBreak - BreakPoint (%d) out of bounds for line runes (%d). Adding full line.", breakPoint, len(runes))
								wrapped = append(wrapped, currentLine)
								currentLine = ""
								break
							}
						}
					}
					if tw.DisplayWidth(currentLine) > 0 {
						wrapped = append(wrapped, currentLine)
					}
					if len(wrapped) == 0 && tw.DisplayWidth(line) > 0 && len(finalLinesForCell) == 0 {
						finalLinesForCell = append(finalLinesForCell, line)
					} else {
						finalLinesForCell = append(finalLinesForCell, wrapped...)
					}
				default:
					finalLinesForCell = append(finalLinesForCell, line)
				}
			} else {
				finalLinesForCell = append(finalLinesForCell, line)
			}
		}

		for len(result) < len(finalLinesForCell) {
			newRow := make([]string, effectiveNumCols)
			for j := range newRow {
				newRow[j] = tw.Empty
			}
			result = append(result, newRow)
		}

		for j := 0; j < len(result); j++ {
			cellLineContent := tw.Empty
			if j < len(finalLinesForCell) {
				cellLineContent = finalLinesForCell[j]
			}
			if i < len(result[j]) {
				result[j][i] = cellLineContent
			} else {
				t.logger.Warn("prepareContent: Column index %d out of bounds (%d) during result matrix population.", i, len(result[j]))
			}
		}
	}

	t.logger.Debug("prepareContent: Content prepared, result %d lines.", len(result))
	return result
}

// prepareContexts initializes rendering and merge contexts.
// No parameters are required.
// Returns renderContext, mergeContext, and an error if initialization fails.
func (t *Table) prepareContexts() (*renderContext, *mergeContext, error) {
	numOriginalCols := t.maxColumns()
	t.logger.Debug("prepareContexts: Original number of columns: %d", numOriginalCols)

	ctx := &renderContext{
		table:    t,
		renderer: t.renderer,
		cfg:      t.renderer.Config(),
		numCols:  numOriginalCols,
		widths: map[tw.Position]tw.Mapper[int, int]{
			tw.Header: tw.NewMapper[int, int](),
			tw.Row:    tw.NewMapper[int, int](),
			tw.Footer: tw.NewMapper[int, int](),
		},
		logger: t.logger,
	}

	isEmpty, visibleCount := t.getEmptyColumnInfo(numOriginalCols)
	ctx.emptyColumns = isEmpty
	ctx.visibleColCount = visibleCount

	mctx := &mergeContext{
		headerMerges: make(map[int]tw.MergeState),
		rowMerges:    make([]map[int]tw.MergeState, len(t.rows)),
		footerMerges: make(map[int]tw.MergeState),
		horzMerges:   make(map[tw.Position]map[int]bool),
	}
	for i := range mctx.rowMerges {
		mctx.rowMerges[i] = make(map[int]tw.MergeState)
	}

	ctx.headerLines = t.headers
	ctx.rowLines = t.rows
	ctx.footerLines = t.footers

	if err := t.calculateAndNormalizeWidths(ctx); err != nil {
		t.logger.Debug("Error during initial width calculation: %v", err)
		return nil, nil, err
	}
	t.logger.Debug("Initial normalized widths (before hiding): H=%v, R=%v, F=%v",
		ctx.widths[tw.Header], ctx.widths[tw.Row], ctx.widths[tw.Footer])

	preparedHeaderLines, headerMerges, _ := t.prepareWithMerges(ctx.headerLines, t.config.Header, tw.Header)
	ctx.headerLines = preparedHeaderLines
	mctx.headerMerges = headerMerges

	processedRowLines := make([][][]string, len(ctx.rowLines))
	for i, row := range ctx.rowLines {
		if mctx.rowMerges[i] == nil {
			mctx.rowMerges[i] = make(map[int]tw.MergeState)
		}
		processedRowLines[i], mctx.rowMerges[i], _ = t.prepareWithMerges(row, t.config.Row, tw.Row)
	}
	ctx.rowLines = processedRowLines

	t.applyHorizontalMergeWidths(tw.Header, ctx, mctx.headerMerges)

	if t.config.Row.Formatting.MergeMode&tw.MergeVertical != 0 {
		t.applyVerticalMerges(ctx, mctx)
	}
	if t.config.Row.Formatting.MergeMode&tw.MergeHierarchical != 0 {
		t.applyHierarchicalMerges(ctx, mctx)
	}

	t.prepareFooter(ctx, mctx)
	t.logger.Debug("Footer prepared. Widths before hiding: H=%v, R=%v, F=%v",
		ctx.widths[tw.Header], ctx.widths[tw.Row], ctx.widths[tw.Footer])

	if t.config.Behavior.AutoHide.Enabled() {
		t.logger.Debug("Applying AutoHide: Adjusting widths for empty columns.")
		if ctx.emptyColumns == nil {
			t.logger.Debug("Warning: ctx.emptyColumns is nil during width adjustment.")
		} else if len(ctx.emptyColumns) != ctx.numCols {
			t.logger.Debug("Warning: Length mismatch between emptyColumns (%d) and numCols (%d). Skipping adjustment.", len(ctx.emptyColumns), ctx.numCols)
		} else {
			for colIdx := 0; colIdx < ctx.numCols; colIdx++ {
				if ctx.emptyColumns[colIdx] {
					t.logger.Debug("AutoHide: Hiding column %d by setting width to 0.", colIdx)
					ctx.widths[tw.Header].Set(colIdx, 0)
					ctx.widths[tw.Row].Set(colIdx, 0)
					ctx.widths[tw.Footer].Set(colIdx, 0)
				}
			}
			t.logger.Debug("Widths after AutoHide adjustment: H=%v, R=%v, F=%v",
				ctx.widths[tw.Header], ctx.widths[tw.Row], ctx.widths[tw.Footer])
		}
	} else {
		t.logger.Debug("AutoHide is disabled, skipping width adjustment.")
	}
	t.logger.Debug("prepareContexts completed all stages.")
	return ctx, mctx, nil
}

// prepareFooter processes footer content and applies merges.
// Parameters ctx and mctx hold rendering and merge state.
// No return value.
func (t *Table) prepareFooter(ctx *renderContext, mctx *mergeContext) {
	if len(t.footers) == 0 {
		ctx.logger.Debug("Skipping footer preparation - no footer data")
		if ctx.widths[tw.Footer] == nil {
			ctx.widths[tw.Footer] = tw.NewMapper[int, int]()
		}
		numCols := ctx.numCols
		for i := 0; i < numCols; i++ {
			ctx.widths[tw.Footer].Set(i, ctx.widths[tw.Row].Get(i))
		}
		t.logger.Debug("Initialized empty footer widths based on row widths: %v", ctx.widths[tw.Footer])
		ctx.footerPrepared = true
		return
	}

	t.logger.Debug("Preparing footer with merge mode: %d", t.config.Footer.Formatting.MergeMode)
	preparedLines, mergeStates, _ := t.prepareWithMerges(t.footers, t.config.Footer, tw.Footer)
	t.footers = preparedLines
	mctx.footerMerges = mergeStates
	ctx.footerLines = t.footers
	t.logger.Debug("Base footer widths (normalized from rows/header): %v", ctx.widths[tw.Footer])
	t.applyHorizontalMergeWidths(tw.Footer, ctx, mctx.footerMerges)
	ctx.footerPrepared = true
	t.logger.Debug("Footer preparation completed. Final footer widths: %v", ctx.widths[tw.Footer])
}

// prepareWithMerges processes content and detects horizontal merges.
// Parameters include content, config, and position (Header, Row, Footer).
// Returns processed lines, merge states, and horizontal merge map.
func (t *Table) prepareWithMerges(content [][]string, config tw.CellConfig, position tw.Position) ([][]string, map[int]tw.MergeState, map[int]bool) {
	t.logger.Debug("PrepareWithMerges START: position=%s, mergeMode=%d", position, config.Formatting.MergeMode)
	if len(content) == 0 {
		t.logger.Debug("PrepareWithMerges END: No content.")
		return content, nil, nil
	}

	numCols := 0
	if len(content) > 0 && len(content[0]) > 0 { // Assumes content[0] exists and has items
		numCols = len(content[0])
	} else { // Fallback if first line is empty or content is empty
		for _, line := range content { // Find max columns from any line
			if len(line) > numCols {
				numCols = len(line)
			}
		}
		if numCols == 0 { // If still 0, try table-wide max (batch mode context)
			numCols = t.maxColumns()
		}
	}

	if numCols == 0 {
		t.logger.Debug("PrepareWithMerges END: numCols is zero.")
		return content, nil, nil
	}

	horzMergeMap := make(map[int]bool)      // Tracks if a column is part of any horizontal merge for this logical row
	mergeMap := make(map[int]tw.MergeState) // Final merge states for this logical row

	// Ensure all lines in 'content' are padded to numCols for consistent processing
	// This result is what will be modified and returned.
	result := make([][]string, len(content))
	for i := range content {
		result[i] = padLine(content[i], numCols)
	}

	if config.Formatting.MergeMode&tw.MergeHorizontal != 0 {
		t.logger.Debug("Checking for horizontal merges (logical cell comparison) for %d visual lines, %d columns", len(content), numCols)

		// Special handling for footer lead merge (often for "TOTAL" spanning empty cells)
		// This logic only applies if it's a footer and typically to the first (often only) visual line.
		if position == tw.Footer && len(content) > 0 {
			lineIdx := 0                                       // Assume footer lead merge applies to the first visual line primarily
			originalLine := padLine(content[lineIdx], numCols) // Use original content for decision
			currentLineResult := result[lineIdx]               // Modify the result line

			firstContentIdx := -1
			var firstContent string
			for c := 0; c < numCols; c++ {
				if c >= len(originalLine) {
					break
				}
				trimmedVal := strings.TrimSpace(originalLine[c])
				if trimmedVal != "" && trimmedVal != "-" { // "-" is often a placeholder not to merge over
					firstContentIdx = c
					firstContent = originalLine[c] // Store the raw content for placement
					break
				} else if trimmedVal == "-" { // Stop if we hit a hard non-mergeable placeholder
					break
				}
			}

			if firstContentIdx > 0 { // If content starts after the first column
				span := firstContentIdx + 1 // Merge from col 0 up to and including firstContentIdx
				startCol := 0

				allEmptyBefore := true
				for c := 0; c < firstContentIdx; c++ {
					if c >= len(originalLine) || strings.TrimSpace(originalLine[c]) != "" {
						allEmptyBefore = false
						break
					}
				}

				if allEmptyBefore {
					t.logger.Debug("Footer lead-merge applied line %d: content '%s' from col %d moved to col %d, span %d",
						lineIdx, firstContent, firstContentIdx, startCol, span)

					if startCol < len(currentLineResult) {
						currentLineResult[startCol] = firstContent // Place the original content
					}
					for k := startCol + 1; k < startCol+span; k++ { // Clear out other cells in the span
						if k < len(currentLineResult) {
							currentLineResult[k] = tw.Empty
						}
					}

					// Update mergeMap for all visual lines of this logical row
					for visualLine := 0; visualLine < len(result); visualLine++ {
						// Only apply the data move to the line where it was detected,
						// but the merge state should apply to the logical cell (all its visual lines).
						if visualLine != lineIdx { // For other visual lines, just clear the cells in the span
							if startCol < len(result[visualLine]) {
								result[visualLine][startCol] = tw.Empty // Typically empty for other lines in a lead merge
							}
							for k := startCol + 1; k < startCol+span; k++ {
								if k < len(result[visualLine]) {
									result[visualLine][k] = tw.Empty
								}
							}
						}
					}

					// Set merge state for the starting column
					startState := mergeMap[startCol]
					startState.Horizontal = tw.MergeStateOption{Present: true, Span: span, Start: true, End: (span == 1)}
					mergeMap[startCol] = startState
					horzMergeMap[startCol] = true // Mark this column as processed by a merge

					// Set merge state for subsequent columns in the span
					for k := startCol + 1; k < startCol+span; k++ {
						colState := mergeMap[k]
						colState.Horizontal = tw.MergeStateOption{Present: true, Span: span, Start: false, End: k == startCol+span-1}
						mergeMap[k] = colState
						horzMergeMap[k] = true // Mark as processed
					}
				}
			}
		}

		// Standard horizontal merge logic based on full logical cell content
		col := 0
		for col < numCols {
			if horzMergeMap[col] { // If already part of a footer lead-merge, skip
				col++
				continue
			}

			// Get full content of logical cell 'col'
			var currentLogicalCellContentBuilder strings.Builder
			for lineIdx := 0; lineIdx < len(content); lineIdx++ {
				if col < len(content[lineIdx]) {
					currentLogicalCellContentBuilder.WriteString(content[lineIdx][col])
				}
			}
			currentLogicalCellTrimmed := strings.TrimSpace(currentLogicalCellContentBuilder.String())

			if currentLogicalCellTrimmed == "" || currentLogicalCellTrimmed == "-" {
				col++
				continue
			}

			span := 1
			for nextCol := col + 1; nextCol < numCols; nextCol++ {
				if horzMergeMap[nextCol] { // Don't merge into an already merged (e.g. footer lead) column
					break
				}
				var nextLogicalCellContentBuilder strings.Builder
				for lineIdx := 0; lineIdx < len(content); lineIdx++ {
					if nextCol < len(content[lineIdx]) {
						nextLogicalCellContentBuilder.WriteString(content[lineIdx][nextCol])
					}
				}
				nextLogicalCellTrimmed := strings.TrimSpace(nextLogicalCellContentBuilder.String())

				if currentLogicalCellTrimmed == nextLogicalCellTrimmed && nextLogicalCellTrimmed != "-" {
					span++
				} else {
					break
				}
			}

			if span > 1 {
				t.logger.Debug("Standard horizontal merge (logical cell): startCol %d, span %d for content '%s'", col, span, currentLogicalCellTrimmed)
				startState := mergeMap[col]
				startState.Horizontal = tw.MergeStateOption{Present: true, Span: span, Start: true, End: (span == 1)}
				mergeMap[col] = startState
				horzMergeMap[col] = true

				// For all visual lines, clear out the content of the merged-over cells
				for lineIdx := 0; lineIdx < len(result); lineIdx++ {
					for k := col + 1; k < col+span; k++ {
						if k < len(result[lineIdx]) {
							result[lineIdx][k] = tw.Empty
						}
					}
				}

				// Set merge state for subsequent columns in the span
				for k := col + 1; k < col+span; k++ {
					colState := mergeMap[k]
					colState.Horizontal = tw.MergeStateOption{Present: true, Span: span, Start: false, End: k == col+span-1}
					mergeMap[k] = colState
					horzMergeMap[k] = true
				}
				col += span
			} else {
				col++
			}
		}
	}

	t.logger.Debug("PrepareWithMerges END: position=%s, lines=%d, mergeMapH: %v", position, len(result), func() map[int]tw.MergeStateOption {
		m := make(map[int]tw.MergeStateOption)
		for k, v := range mergeMap {
			m[k] = v.Horizontal
		}
		return m
	}())
	return result, mergeMap, horzMergeMap
}

// renderFooter renders the table's footer section with borders and padding.
// Parameters ctx and mctx hold rendering and merge state.
// Returns an error if rendering fails.
func (t *Table) renderFooter(ctx *renderContext, mctx *mergeContext) error {
	if !ctx.footerPrepared {
		t.prepareFooter(ctx, mctx)
	}

	f := ctx.renderer
	cfg := ctx.cfg

	hasContent := len(ctx.footerLines) > 0
	hasTopPadding := t.config.Footer.Padding.Global.Top != tw.Empty
	hasBottomPaddingConfig := t.config.Footer.Padding.Global.Bottom != tw.Empty || t.hasPerColumnBottomPadding()
	hasAnyFooterElement := hasContent || hasTopPadding || hasBottomPaddingConfig

	if !hasAnyFooterElement {
		hasContentAbove := len(ctx.rowLines) > 0 || len(ctx.headerLines) > 0
		if hasContentAbove && cfg.Borders.Bottom.Enabled() && cfg.Settings.Lines.ShowBottom.Enabled() {
			ctx.logger.Debug("Footer is empty, rendering table bottom border based on last row/header")
			var lastLineAboveCtx *helperContext
			var lastLineAligns map[int]tw.Align
			var lastLinePadding map[int]tw.Padding

			if len(ctx.rowLines) > 0 {
				lastRowIdx := len(ctx.rowLines) - 1
				lastRowLineIdx := -1
				var lastRowLine []string
				if lastRowIdx >= 0 && len(ctx.rowLines[lastRowIdx]) > 0 {
					lastRowLineIdx = len(ctx.rowLines[lastRowIdx]) - 1
					lastRowLine = padLine(ctx.rowLines[lastRowIdx][lastRowLineIdx], ctx.numCols)
				} else {
					lastRowLine = make([]string, ctx.numCols)
				}
				lastLineAboveCtx = &helperContext{
					position: tw.Row,
					rowIdx:   lastRowIdx,
					lineIdx:  lastRowLineIdx,
					line:     lastRowLine,
					location: tw.LocationEnd,
				}
				lastLineAligns = t.buildAligns(t.config.Row)
				lastLinePadding = t.buildPadding(t.config.Row.Padding)
			} else {
				lastHeaderLineIdx := -1
				var lastHeaderLine []string
				if len(ctx.headerLines) > 0 {
					lastHeaderLineIdx = len(ctx.headerLines) - 1
					lastHeaderLine = padLine(ctx.headerLines[lastHeaderLineIdx], ctx.numCols)
				} else {
					lastHeaderLine = make([]string, ctx.numCols)
				}
				lastLineAboveCtx = &helperContext{
					position: tw.Header,
					rowIdx:   0,
					lineIdx:  lastHeaderLineIdx,
					line:     lastHeaderLine,
					location: tw.LocationEnd,
				}
				lastLineAligns = t.buildAligns(t.config.Header)
				lastLinePadding = t.buildPadding(t.config.Header.Padding)
			}

			resp := t.buildCellContexts(ctx, mctx, lastLineAboveCtx, lastLineAligns, lastLinePadding)
			ctx.logger.Debug("Bottom border: Using Widths=%v", ctx.widths[tw.Row])
			f.Line(t.writer, tw.Formatting{
				Row: tw.RowContext{
					Widths:       ctx.widths[tw.Row],
					Current:      resp.cells,
					Previous:     resp.prevCells,
					Position:     lastLineAboveCtx.position,
					Location:     tw.LocationEnd,
					ColMaxWidths: t.getColMaxWidths(tw.Footer),
				},
				Level:    tw.LevelFooter,
				IsSubRow: false,
			})
		} else {
			ctx.logger.Debug("Footer is empty and no content above or borders disabled, skipping footer render")
		}
		return nil
	}

	ctx.logger.Debug("Rendering footer section (has elements)")
	hasContentAbove := len(ctx.rowLines) > 0 || len(ctx.headerLines) > 0
	colAligns := t.buildAligns(t.config.Footer)
	colPadding := t.buildPadding(t.config.Footer.Padding)
	hctx := &helperContext{position: tw.Footer}
	// Declare paddingLineContentForContext with a default value
	paddingLineContentForContext := make([]string, ctx.numCols)

	if hasContentAbove && cfg.Settings.Lines.ShowFooterLine.Enabled() && !hasTopPadding && len(ctx.footerLines) > 0 {
		ctx.logger.Debug("Rendering footer separator line")
		var lastLineAboveCtx *helperContext
		var lastLineAligns map[int]tw.Align
		var lastLinePadding map[int]tw.Padding
		var lastLinePosition tw.Position

		if len(ctx.rowLines) > 0 {
			lastRowIdx := len(ctx.rowLines) - 1
			lastRowLineIdx := -1
			var lastRowLine []string
			if lastRowIdx >= 0 && len(ctx.rowLines[lastRowIdx]) > 0 {
				lastRowLineIdx = len(ctx.rowLines[lastRowIdx]) - 1
				lastRowLine = padLine(ctx.rowLines[lastRowIdx][lastRowLineIdx], ctx.numCols)
			} else {
				lastRowLine = make([]string, ctx.numCols)
			}
			lastLineAboveCtx = &helperContext{
				position: tw.Row,
				rowIdx:   lastRowIdx,
				lineIdx:  lastRowLineIdx,
				line:     lastRowLine,
				location: tw.LocationMiddle,
			}
			lastLineAligns = t.buildAligns(t.config.Row)
			lastLinePadding = t.buildPadding(t.config.Row.Padding)
			lastLinePosition = tw.Row
		} else {
			lastHeaderLineIdx := -1
			var lastHeaderLine []string
			if len(ctx.headerLines) > 0 {
				lastHeaderLineIdx = len(ctx.headerLines) - 1
				lastHeaderLine = padLine(ctx.headerLines[lastHeaderLineIdx], ctx.numCols)
			} else {
				lastHeaderLine = make([]string, ctx.numCols)
			}
			lastLineAboveCtx = &helperContext{
				position: tw.Header,
				rowIdx:   0,
				lineIdx:  lastHeaderLineIdx,
				line:     lastHeaderLine,
				location: tw.LocationMiddle,
			}
			lastLineAligns = t.buildAligns(t.config.Header)
			lastLinePadding = t.buildPadding(t.config.Header.Padding)
			lastLinePosition = tw.Header
		}

		resp := t.buildCellContexts(ctx, mctx, lastLineAboveCtx, lastLineAligns, lastLinePadding)
		var nextCells map[int]tw.CellContext
		if hasContent {
			nextCells = make(map[int]tw.CellContext)
			for j, cellData := range padLine(ctx.footerLines[0], ctx.numCols) {
				mergeState := tw.MergeState{}
				if mctx.footerMerges != nil {
					mergeState = mctx.footerMerges[j]
				}
				nextCells[j] = tw.CellContext{Data: cellData, Merge: mergeState, Width: ctx.widths[tw.Footer].Get(j)}
			}
		}
		ctx.logger.Debug("Footer separator: Using Widths=%v", ctx.widths[tw.Row])
		f.Line(t.writer, tw.Formatting{
			Row: tw.RowContext{
				Widths:       ctx.widths[tw.Row],
				Current:      resp.cells,
				Previous:     resp.prevCells,
				Next:         nextCells,
				Position:     lastLinePosition,
				Location:     tw.LocationMiddle,
				ColMaxWidths: t.getColMaxWidths(tw.Footer),
			},
			Level:     tw.LevelFooter,
			IsSubRow:  false,
			HasFooter: true,
		})
	}

	if hasTopPadding {
		hctx.rowIdx = 0
		hctx.lineIdx = -1
		if !(hasContentAbove && cfg.Settings.Lines.ShowFooterLine.Enabled()) {
			hctx.location = tw.LocationFirst
		} else {
			hctx.location = tw.LocationMiddle
		}
		hctx.line = t.buildPaddingLineContents(t.config.Footer.Padding.Global.Top, ctx.widths[tw.Footer], ctx.numCols, mctx.footerMerges)
		ctx.logger.Debug("Calling renderPadding for Footer Top Padding line: %v (loc: %v)", hctx.line, hctx.location)
		if err := t.renderPadding(ctx, mctx, hctx, t.config.Footer.Padding.Global.Top); err != nil {
			return err
		}
	}

	lastRenderedLineIdx := -2
	if hasTopPadding {
		lastRenderedLineIdx = -1
	}
	for i, line := range ctx.footerLines {
		hctx.rowIdx = 0
		hctx.lineIdx = i
		hctx.line = padLine(line, ctx.numCols)
		isFirstContentLine := i == 0
		isLastContentLine := i == len(ctx.footerLines)-1
		if isFirstContentLine && !hasTopPadding && !(hasContentAbove && cfg.Settings.Lines.ShowFooterLine.Enabled()) {
			hctx.location = tw.LocationFirst
		} else if isLastContentLine && !hasBottomPaddingConfig {
			hctx.location = tw.LocationEnd
		} else {
			hctx.location = tw.LocationMiddle
		}
		ctx.logger.Debug("Rendering footer content line %d with location %v", i, hctx.location)
		if err := t.renderLine(ctx, mctx, hctx, colAligns, colPadding); err != nil {
			return err
		}
		lastRenderedLineIdx = i
	}

	if hasBottomPaddingConfig {
		paddingLineContentForContext = make([]string, ctx.numCols)
		formattedPaddingCells := make([]string, ctx.numCols)
		var representativePadChar string = " "
		ctx.logger.Debug("Constructing Footer Bottom Padding line content strings")
		for j := 0; j < ctx.numCols; j++ {
			colWd := ctx.widths[tw.Footer].Get(j)
			mergeState := tw.MergeState{}
			if mctx.footerMerges != nil {
				if state, ok := mctx.footerMerges[j]; ok {
					mergeState = state
				}
			}
			if mergeState.Horizontal.Present && !mergeState.Horizontal.Start {
				paddingLineContentForContext[j] = ""
				formattedPaddingCells[j] = ""
				continue
			}
			padChar := " "
			if j < len(t.config.Footer.Padding.PerColumn) && t.config.Footer.Padding.PerColumn[j].Bottom != tw.Empty {
				padChar = t.config.Footer.Padding.PerColumn[j].Bottom
			} else if t.config.Footer.Padding.Global.Bottom != tw.Empty {
				padChar = t.config.Footer.Padding.Global.Bottom
			}
			paddingLineContentForContext[j] = padChar
			if j == 0 || representativePadChar == " " {
				representativePadChar = padChar
			}
			padWidth := tw.DisplayWidth(padChar)
			if padWidth < 1 {
				padWidth = 1
			}
			repeatCount := 0
			if colWd > 0 && padWidth > 0 {
				repeatCount = colWd / padWidth
			}
			if colWd > 0 && repeatCount < 1 && padChar != " " {
				repeatCount = 1
			}
			if colWd == 0 {
				repeatCount = 0
			}
			rawPaddingContent := strings.Repeat(padChar, repeatCount)
			currentWd := tw.DisplayWidth(rawPaddingContent)
			if currentWd < colWd {
				rawPaddingContent += strings.Repeat(" ", colWd-currentWd)
			}
			if currentWd > colWd && colWd > 0 {
				rawPaddingContent = tw.TruncateString(rawPaddingContent, colWd)
			}
			if colWd == 0 {
				rawPaddingContent = ""
			}
			formattedPaddingCells[j] = rawPaddingContent
		}
		ctx.logger.Debug("Manually rendering Footer Bottom Padding line (char like '%s')", representativePadChar)
		var paddingLineOutput strings.Builder
		if cfg.Borders.Left.Enabled() {
			paddingLineOutput.WriteString(cfg.Symbols.Column())
		}
		for colIdx := 0; colIdx < ctx.numCols; {
			if colIdx > 0 && cfg.Settings.Separators.BetweenColumns.Enabled() {
				shouldAddSeparator := true
				if prevMergeState, ok := mctx.footerMerges[colIdx-1]; ok {
					if prevMergeState.Horizontal.Present && !prevMergeState.Horizontal.End {
						shouldAddSeparator = false
					}
				}
				if shouldAddSeparator {
					paddingLineOutput.WriteString(cfg.Symbols.Column())
				}
			}
			if colIdx < len(formattedPaddingCells) {
				paddingLineOutput.WriteString(formattedPaddingCells[colIdx])
			}
			currentMergeState := tw.MergeState{}
			if mctx.footerMerges != nil {
				if state, ok := mctx.footerMerges[colIdx]; ok {
					currentMergeState = state
				}
			}
			if currentMergeState.Horizontal.Present && currentMergeState.Horizontal.Start {
				colIdx += currentMergeState.Horizontal.Span
			} else {
				colIdx++
			}
		}
		if cfg.Borders.Right.Enabled() {
			paddingLineOutput.WriteString(cfg.Symbols.Column())
		}
		paddingLineOutput.WriteString(t.newLine)
		fmt.Fprint(t.writer, paddingLineOutput.String())
		ctx.logger.Debug("Manually rendered Footer Bottom Padding line: %s", strings.TrimSuffix(paddingLineOutput.String(), t.newLine))
		hctx.rowIdx = 0
		hctx.lineIdx = len(ctx.footerLines)
		hctx.line = paddingLineContentForContext
		hctx.location = tw.LocationEnd
		lastRenderedLineIdx = hctx.lineIdx
	}

	if cfg.Borders.Bottom.Enabled() && cfg.Settings.Lines.ShowBottom.Enabled() {
		ctx.logger.Debug("Rendering final table bottom border")
		if lastRenderedLineIdx == len(ctx.footerLines) {
			hctx.rowIdx = 0
			hctx.lineIdx = lastRenderedLineIdx
			hctx.line = paddingLineContentForContext
			hctx.location = tw.LocationEnd
			ctx.logger.Debug("Setting border context based on bottom padding line")
		} else if lastRenderedLineIdx >= 0 {
			hctx.rowIdx = 0
			hctx.lineIdx = lastRenderedLineIdx
			hctx.line = padLine(ctx.footerLines[hctx.lineIdx], ctx.numCols)
			hctx.location = tw.LocationEnd
			ctx.logger.Debug("Setting border context based on last content line idx %d", hctx.lineIdx)
		} else if lastRenderedLineIdx == -1 {
			hctx.rowIdx = 0
			hctx.lineIdx = -1
			hctx.line = paddingLineContentForContext
			hctx.location = tw.LocationEnd
			ctx.logger.Debug("Setting border context based on top padding line")
		} else {
			hctx.rowIdx = 0
			hctx.lineIdx = -2
			hctx.line = make([]string, ctx.numCols)
			hctx.location = tw.LocationEnd
			ctx.logger.Debug("Warning: Cannot determine context for bottom border")
		}
		resp := t.buildCellContexts(ctx, mctx, hctx, colAligns, colPadding)
		ctx.logger.Debug("Bottom border: Using Widths=%v", ctx.widths[tw.Row])
		f.Line(t.writer, tw.Formatting{
			Row: tw.RowContext{
				Widths:       ctx.widths[tw.Row],
				Current:      resp.cells,
				Previous:     resp.prevCells,
				Position:     tw.Footer,
				Location:     tw.LocationEnd,
				ColMaxWidths: t.getColMaxWidths(tw.Footer),
			},
			Level:    tw.LevelFooter,
			IsSubRow: false,
		})
	}

	return nil
}

// renderHeader renders the table's header section with borders and padding.
// Parameters ctx and mctx hold rendering and merge state.
// Returns an error if rendering fails.
func (t *Table) renderHeader(ctx *renderContext, mctx *mergeContext) error {
	if len(ctx.headerLines) == 0 {
		return nil
	}
	ctx.logger.Debug("Rendering header section")

	f := ctx.renderer
	cfg := ctx.cfg
	colAligns := t.buildAligns(t.config.Header)
	colPadding := t.buildPadding(t.config.Header.Padding)
	hctx := &helperContext{position: tw.Header}

	if cfg.Borders.Top.Enabled() && cfg.Settings.Lines.ShowTop.Enabled() {
		ctx.logger.Debug("Rendering table top border")
		nextCells := make(map[int]tw.CellContext)
		if len(ctx.headerLines) > 0 {
			for j, cell := range ctx.headerLines[0] {
				nextCells[j] = tw.CellContext{Data: cell, Merge: mctx.headerMerges[j]}
			}
		}
		f.Line(t.writer, tw.Formatting{
			Row: tw.RowContext{
				Widths:   ctx.widths[tw.Header],
				Next:     nextCells,
				Position: tw.Header,
				Location: tw.LocationFirst,
			},
			Level:    tw.LevelHeader,
			IsSubRow: false,
		})
	}

	if t.config.Header.Padding.Global.Top != tw.Empty {
		hctx.location = tw.LocationFirst
		hctx.line = t.buildPaddingLineContents(t.config.Header.Padding.Global.Top, ctx.widths[tw.Header], ctx.numCols, mctx.headerMerges)
		if err := t.renderPadding(ctx, mctx, hctx, t.config.Header.Padding.Global.Top); err != nil {
			return err
		}
	}

	for i, line := range ctx.headerLines {
		hctx.rowIdx = 0
		hctx.lineIdx = i
		hctx.line = padLine(line, ctx.numCols)
		hctx.location = t.determineLocation(i, len(ctx.headerLines), t.config.Header.Padding.Global.Top, t.config.Header.Padding.Global.Bottom)

		if t.config.Header.Callbacks.Global != nil {
			ctx.logger.Debug("Executing global header callback for line %d", i)
			t.config.Header.Callbacks.Global()
		}
		for colIdx, cb := range t.config.Header.Callbacks.PerColumn {
			if colIdx < ctx.numCols && cb != nil {
				ctx.logger.Debug("Executing per-column header callback for line %d, col %d", i, colIdx)
				cb()
			}
		}

		if err := t.renderLine(ctx, mctx, hctx, colAligns, colPadding); err != nil {
			return err
		}
	}

	if t.config.Header.Padding.Global.Bottom != tw.Empty {
		hctx.location = tw.LocationEnd
		hctx.line = t.buildPaddingLineContents(t.config.Header.Padding.Global.Bottom, ctx.widths[tw.Header], ctx.numCols, mctx.headerMerges)
		if err := t.renderPadding(ctx, mctx, hctx, t.config.Header.Padding.Global.Bottom); err != nil {
			return err
		}
	}

	if cfg.Settings.Lines.ShowHeaderLine.Enabled() && (len(ctx.rowLines) > 0 || len(ctx.footerLines) > 0) {
		ctx.logger.Debug("Rendering header separator line")
		hctx.rowIdx = 0
		hctx.lineIdx = len(ctx.headerLines) - 1
		hctx.line = padLine(ctx.headerLines[hctx.lineIdx], ctx.numCols)
		hctx.location = tw.LocationMiddle
		resp := t.buildCellContexts(ctx, mctx, hctx, colAligns, colPadding)
		f.Line(t.writer, tw.Formatting{
			Row: tw.RowContext{
				Widths:   ctx.widths[tw.Header],
				Current:  resp.cells,
				Previous: resp.prevCells,
				Next:     resp.nextCells,
				Position: tw.Header,
				Location: tw.LocationMiddle,
			},
			Level:    tw.LevelBody,
			IsSubRow: false,
		})
	}
	return nil
}

// renderLine renders a single line with callbacks and normalized widths.
// Parameters include ctx, mctx, hctx, aligns, and padding for rendering.
// Returns an error if rendering fails.
func (t *Table) renderLine(ctx *renderContext, mctx *mergeContext, hctx *helperContext, aligns map[int]tw.Align, padding map[int]tw.Padding) error {
	resp := t.buildCellContexts(ctx, mctx, hctx, aligns, padding)
	f := ctx.renderer

	isPaddingLine := false
	sectionConfig := t.config.Row
	switch hctx.position {
	case tw.Header:
		sectionConfig = t.config.Header
		isPaddingLine = (hctx.lineIdx == -1 && sectionConfig.Padding.Global.Top != tw.Empty) ||
			(hctx.lineIdx == len(ctx.headerLines) && sectionConfig.Padding.Global.Bottom != tw.Empty)
	case tw.Footer:
		sectionConfig = t.config.Footer
		isPaddingLine = (hctx.lineIdx == -1 && sectionConfig.Padding.Global.Top != tw.Empty) ||
			(hctx.lineIdx == len(ctx.footerLines) && (sectionConfig.Padding.Global.Bottom != tw.Empty || t.hasPerColumnBottomPadding()))
	case tw.Row:
		if hctx.rowIdx >= 0 && hctx.rowIdx < len(ctx.rowLines) {
			isPaddingLine = (hctx.lineIdx == -1 && sectionConfig.Padding.Global.Top != tw.Empty) ||
				(hctx.lineIdx == len(ctx.rowLines[hctx.rowIdx]) && sectionConfig.Padding.Global.Bottom != tw.Empty)
		}
	}

	sectionWidths := ctx.widths[hctx.position]
	normalizedWidths := ctx.widths[tw.Row]

	formatting := tw.Formatting{
		Row: tw.RowContext{
			Widths:       sectionWidths,
			ColMaxWidths: t.getColMaxWidths(hctx.position),
			Current:      resp.cells,
			Previous:     resp.prevCells,
			Next:         resp.nextCells,
			Position:     hctx.position,
			Location:     hctx.location,
		},
		Level:            t.getLevel(hctx.position),
		IsSubRow:         hctx.lineIdx > 0 || isPaddingLine,
		NormalizedWidths: normalizedWidths,
	}

	if hctx.position == tw.Row {
		formatting.HasFooter = len(ctx.footerLines) > 0
	}

	switch hctx.position {
	case tw.Header:
		f.Header(t.writer, [][]string{hctx.line}, formatting)
	case tw.Row:
		f.Row(t.writer, hctx.line, formatting)
	case tw.Footer:
		f.Footer(t.writer, [][]string{hctx.line}, formatting)
	}
	return nil
}

// renderPadding renders padding lines for a section.
// Parameters include ctx, mctx, hctx, and padChar for padding content.
// Returns an error if rendering fails.
func (t *Table) renderPadding(ctx *renderContext, mctx *mergeContext, hctx *helperContext, padChar string) error {
	ctx.logger.Debug("Rendering padding line for %s (using char like '%s')", hctx.position, padChar)

	colAligns := t.buildAligns(t.config.Row)
	colPadding := t.buildPadding(t.config.Row.Padding)

	switch hctx.position {
	case tw.Header:
		colAligns = t.buildAligns(t.config.Header)
		colPadding = t.buildPadding(t.config.Header.Padding)
	case tw.Footer:
		colAligns = t.buildAligns(t.config.Footer)
		colPadding = t.buildPadding(t.config.Footer.Padding)
	}

	return t.renderLine(ctx, mctx, hctx, colAligns, colPadding)
}

// renderRow renders the table's row section with borders and padding.
// Parameters ctx and mctx hold rendering and merge state.
// Returns an error if rendering fails.
func (t *Table) renderRow(ctx *renderContext, mctx *mergeContext) error {
	if len(ctx.rowLines) == 0 {
		return nil
	}
	ctx.logger.Debug("Rendering row section (total rows: %d)", len(ctx.rowLines))

	f := ctx.renderer
	cfg := ctx.cfg
	colAligns := t.buildAligns(t.config.Row)
	colPadding := t.buildPadding(t.config.Row.Padding)
	hctx := &helperContext{position: tw.Row}

	footerIsEmptyOrNonExistent := !t.hasFooterElements()
	if len(ctx.headerLines) == 0 && footerIsEmptyOrNonExistent && cfg.Borders.Top.Enabled() && cfg.Settings.Lines.ShowTop.Enabled() {
		ctx.logger.Debug("Rendering table top border (rows only table)")
		nextCells := make(map[int]tw.CellContext)
		if len(ctx.rowLines) > 0 && len(ctx.rowLines[0]) > 0 && len(mctx.rowMerges) > 0 {
			firstLine := ctx.rowLines[0][0]
			firstMerges := mctx.rowMerges[0]
			for j, cell := range padLine(firstLine, ctx.numCols) {
				mergeState := tw.MergeState{}
				if firstMerges != nil {
					mergeState = firstMerges[j]
				}
				nextCells[j] = tw.CellContext{Data: cell, Merge: mergeState, Width: ctx.widths[tw.Row].Get(j)}
			}
		}
		f.Line(t.writer, tw.Formatting{
			Row: tw.RowContext{
				Widths:   ctx.widths[tw.Row],
				Next:     nextCells,
				Position: tw.Row,
				Location: tw.LocationFirst,
			},
			Level:    tw.LevelHeader,
			IsSubRow: false,
		})
	}

	for i, lines := range ctx.rowLines {
		rowHasTopPadding := t.config.Row.Padding.Global.Top != tw.Empty
		if rowHasTopPadding {
			hctx.rowIdx = i
			hctx.lineIdx = -1
			if i == 0 && len(ctx.headerLines) == 0 {
				hctx.location = tw.LocationFirst
			} else {
				hctx.location = tw.LocationMiddle
			}
			hctx.line = t.buildPaddingLineContents(t.config.Row.Padding.Global.Top, ctx.widths[tw.Row], ctx.numCols, mctx.rowMerges[i])
			ctx.logger.Debug("Calling renderPadding for Row Top Padding (row %d): %v (loc: %v)", i, hctx.line, hctx.location)
			if err := t.renderPadding(ctx, mctx, hctx, t.config.Row.Padding.Global.Top); err != nil {
				return err
			}
		}

		footerExists := t.hasFooterElements()
		rowHasBottomPadding := t.config.Row.Padding.Global.Bottom != tw.Empty
		isLastRow := i == len(ctx.rowLines)-1

		for j, line := range lines {
			hctx.rowIdx = i
			hctx.lineIdx = j
			hctx.line = padLine(line, ctx.numCols)

			if j > 0 {
				visualLineHasActualContent := false
				// Check if any cell in this visual line (hctx.line which is lines[j]) has actual content
				for _, cellContentInVisualLineK := range hctx.line {
					if strings.TrimSpace(cellContentInVisualLineK) != "" {
						visualLineHasActualContent = true
						break
					}
				}

				if !visualLineHasActualContent {
					// If the entire visual line `j` (where `j>0`) has no actual content
					// (meaning all its cells are trimmed-empty), then we skip it.
					ctx.logger.Debug("Skipping rendering of visual line %d for logical row %d as it contains no actual content after merges/blanking.", j, i)
					continue
				}
			}

			isFirstRow := i == 0
			isLastLineOfRow := j == len(lines)-1

			if isFirstRow && j == 0 && !rowHasTopPadding && len(ctx.headerLines) == 0 {
				hctx.location = tw.LocationFirst
			} else if isLastRow && isLastLineOfRow && !rowHasBottomPadding && !footerExists {
				hctx.location = tw.LocationEnd
			} else {
				hctx.location = tw.LocationMiddle
			}

			ctx.logger.Debug("Rendering row %d line %d with location %v", i, j, hctx.location)
			if err := t.renderLine(ctx, mctx, hctx, colAligns, colPadding); err != nil {
				return err
			}
		}

		if rowHasBottomPadding {
			hctx.rowIdx = i
			hctx.lineIdx = len(lines)
			if isLastRow && !footerExists {
				hctx.location = tw.LocationEnd
			} else {
				hctx.location = tw.LocationMiddle
			}
			hctx.line = t.buildPaddingLineContents(t.config.Row.Padding.Global.Bottom, ctx.widths[tw.Row], ctx.numCols, mctx.rowMerges[i])
			ctx.logger.Debug("Calling renderPadding for Row Bottom Padding (row %d): %v (loc: %v)", i, hctx.line, hctx.location)
			if err := t.renderPadding(ctx, mctx, hctx, t.config.Row.Padding.Global.Bottom); err != nil {
				return err
			}
		}

		if cfg.Settings.Separators.BetweenRows.Enabled() && !isLastRow {
			ctx.logger.Debug("Rendering between-rows separator after logical row %d", i)
			respCurrent := t.buildCellContexts(ctx, mctx, hctx, colAligns, colPadding)

			var nextCellsForSeparator map[int]tw.CellContext = nil
			nextRowIdx := i + 1
			if nextRowIdx < len(ctx.rowLines) && nextRowIdx < len(mctx.rowMerges) {
				hctxNext := &helperContext{position: tw.Row, rowIdx: nextRowIdx, location: tw.LocationMiddle}
				nextRowActualLines := ctx.rowLines[nextRowIdx]
				nextRowMerges := mctx.rowMerges[nextRowIdx]

				if t.config.Row.Padding.Global.Top != tw.Empty {
					hctxNext.lineIdx = -1
					hctxNext.line = t.buildPaddingLineContents(t.config.Row.Padding.Global.Top, ctx.widths[tw.Row], ctx.numCols, nextRowMerges)
				} else if len(nextRowActualLines) > 0 {
					hctxNext.lineIdx = 0
					hctxNext.line = padLine(nextRowActualLines[0], ctx.numCols)
				} else {
					hctxNext.lineIdx = 0
					hctxNext.line = make([]string, ctx.numCols)
				}
				respNext := t.buildCellContexts(ctx, mctx, hctxNext, colAligns, colPadding)
				nextCellsForSeparator = respNext.cells
			} else {
				ctx.logger.Debug("Separator context: No next logical row for separator after row %d.", i)
			}

			f.Line(t.writer, tw.Formatting{
				Row: tw.RowContext{
					Widths:       ctx.widths[tw.Row],
					Current:      respCurrent.cells,
					Previous:     respCurrent.prevCells,
					Next:         nextCellsForSeparator,
					Position:     tw.Row,
					Location:     tw.LocationMiddle,
					ColMaxWidths: t.getColMaxWidths(tw.Row),
				},
				Level:     tw.LevelBody,
				IsSubRow:  false,
				HasFooter: footerExists,
			})
		}
	} // End LOGICAL ROW LOOP (i)
	return nil
}
