package tablewriter

import (
	"database/sql"
	"fmt"
	"github.com/olekukonko/errors"
	"github.com/olekukonko/tablewriter/tw"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// applyHierarchicalMerges applies hierarchical merges to row content.
// Parameters ctx and mctx hold rendering and merge state.
// No return value.
func (t *Table) applyHierarchicalMerges(ctx *renderContext, mctx *mergeContext) {
	ctx.logger.Debug("Applying hierarchical merges (left-to-right vertical flow - snapshot comparison)")
	if len(ctx.rowLines) <= 1 {
		ctx.logger.Debug("Skipping hierarchical merges - less than 2 rows")
		return
	}
	numCols := ctx.numCols

	originalRowLines := make([][][]string, len(ctx.rowLines))
	for i, row := range ctx.rowLines {
		originalRowLines[i] = make([][]string, len(row))
		for j, line := range row {
			originalRowLines[i][j] = make([]string, len(line))
			copy(originalRowLines[i][j], line)
		}
	}
	ctx.logger.Debug("Created snapshot of original row data for hierarchical merge comparison.")

	hMergeStartRow := make(map[int]int)

	for r := 1; r < len(ctx.rowLines); r++ {
		leftCellContinuedHierarchical := false

		for c := 0; c < numCols; c++ {
			if mctx.rowMerges[r] == nil {
				mctx.rowMerges[r] = make(map[int]tw.MergeState)
			}
			if mctx.rowMerges[r-1] == nil {
				mctx.rowMerges[r-1] = make(map[int]tw.MergeState)
			}

			canCompare := r > 0 &&
				len(originalRowLines[r]) > 0 &&
				len(originalRowLines[r-1]) > 0

			if !canCompare {
				currentState := mctx.rowMerges[r][c]
				currentState.Hierarchical = tw.MergeStateOption{}
				mctx.rowMerges[r][c] = currentState
				ctx.logger.Debug("HCompare Skipped: r=%d, c=%d - Insufficient data in snapshot", r, c)
				leftCellContinuedHierarchical = false
				continue
			}

			// Join all lines of the cell for comparison
			var currentVal, aboveVal string
			for _, line := range originalRowLines[r] {
				if c < len(line) {
					currentVal += line[c]
				}
			}
			for _, line := range originalRowLines[r-1] {
				if c < len(line) {
					aboveVal += line[c]
				}
			}

			if t.config.Behavior.TrimSpace.Enabled() {
				currentVal = strings.TrimSpace(currentVal)
				aboveVal = strings.TrimSpace(aboveVal)
			}
			currentState := mctx.rowMerges[r][c]
			prevStateAbove := mctx.rowMerges[r-1][c]

			valuesMatch := currentVal == aboveVal && currentVal != "" && currentVal != "-"
			hierarchyAllowed := c == 0 || leftCellContinuedHierarchical
			shouldContinue := valuesMatch && hierarchyAllowed

			ctx.logger.Debug("HCompare: r=%d, c=%d; current='%s', above='%s'; match=%v; leftCont=%v; shouldCont=%v",
				r, c, currentVal, aboveVal, valuesMatch, leftCellContinuedHierarchical, shouldContinue)

			if shouldContinue {
				currentState.Hierarchical.Present = true
				currentState.Hierarchical.Start = false

				if prevStateAbove.Hierarchical.Present && !prevStateAbove.Hierarchical.End {
					startRow, ok := hMergeStartRow[c]
					if !ok {
						ctx.logger.Debug("Hierarchical merge WARNING: Recovering lost start row at r=%d, c=%d. Assuming r-1 was start.", r, c)
						startRow = r - 1
						hMergeStartRow[c] = startRow
						startState := mctx.rowMerges[startRow][c]
						startState.Hierarchical.Present = true
						startState.Hierarchical.Start = true
						startState.Hierarchical.End = false
						mctx.rowMerges[startRow][c] = startState
					}
					ctx.logger.Debug("Hierarchical merge CONTINUED row %d, col %d. Block previously started row %d", r, c, startRow)
				} else {
					startRow := r - 1
					hMergeStartRow[c] = startRow
					startState := mctx.rowMerges[startRow][c]
					startState.Hierarchical.Present = true
					startState.Hierarchical.Start = true
					startState.Hierarchical.End = false
					mctx.rowMerges[startRow][c] = startState
					ctx.logger.Debug("Hierarchical merge START detected for block ending at or after row %d, col %d (started at row %d)", r, c, startRow)
				}

				for lineIdx := range ctx.rowLines[r] {
					if c < len(ctx.rowLines[r][lineIdx]) {
						ctx.rowLines[r][lineIdx][c] = tw.Empty
					}
				}

				leftCellContinuedHierarchical = true
			} else {
				currentState.Hierarchical = tw.MergeStateOption{}

				if startRow, ok := hMergeStartRow[c]; ok {
					t.finalizeHierarchicalMergeBlock(ctx, mctx, c, startRow, r-1)
					delete(hMergeStartRow, c)
				}

				leftCellContinuedHierarchical = false
			}

			mctx.rowMerges[r][c] = currentState
		}
	}

	lastRowIdx := len(ctx.rowLines) - 1
	if lastRowIdx >= 0 {
		for c, startRow := range hMergeStartRow {
			t.finalizeHierarchicalMergeBlock(ctx, mctx, c, startRow, lastRowIdx)
		}
	}
	ctx.logger.Debug("Hierarchical merge processing completed")
}

// applyHorizontalMergeWidths adjusts column widths for horizontal merges.
// Parameters include position, ctx for rendering, and mergeStates for merges.
// No return value.
func (t *Table) applyHorizontalMergeWidths(position tw.Position, ctx *renderContext, mergeStates map[int]tw.MergeState) {
	if mergeStates == nil {
		t.logger.Debug("applyHorizontalMergeWidths: Skipping %s - no merge states", position)
		return
	}
	t.logger.Debug("applyHorizontalMergeWidths: Applying HMerge width recalc for %s", position)

	numCols := ctx.numCols
	targetWidthsMap := ctx.widths[position]
	originalNormalizedWidths := tw.NewMapper[int, int]()
	for i := 0; i < numCols; i++ {
		originalNormalizedWidths.Set(i, targetWidthsMap.Get(i))
	}

	separatorWidth := 0
	if t.renderer != nil {
		rendererConfig := t.renderer.Config()
		if rendererConfig.Settings.Separators.BetweenColumns.Enabled() {
			separatorWidth = tw.DisplayWidth(rendererConfig.Symbols.Column())
		}
	}

	processedCols := make(map[int]bool)

	for col := 0; col < numCols; col++ {
		if processedCols[col] {
			continue
		}

		state, exists := mergeStates[col]
		if !exists {
			continue
		}

		if state.Horizontal.Present && state.Horizontal.Start {
			totalWidth := 0
			span := state.Horizontal.Span
			t.logger.Debug("  -> HMerge detected: startCol=%d, span=%d, separatorWidth=%d", col, span, separatorWidth)

			for i := 0; i < span && (col+i) < numCols; i++ {
				currentColIndex := col + i
				normalizedWidth := originalNormalizedWidths.Get(currentColIndex)
				totalWidth += normalizedWidth
				t.logger.Debug("      -> col %d: adding normalized width %d", currentColIndex, normalizedWidth)

				if i > 0 && separatorWidth > 0 {
					totalWidth += separatorWidth
					t.logger.Debug("      -> col %d: adding separator width %d", currentColIndex, separatorWidth)
				}
			}

			targetWidthsMap.Set(col, totalWidth)
			t.logger.Debug("  -> Set %s col %d width to %d (merged)", position, col, totalWidth)
			processedCols[col] = true

			for i := 1; i < span && (col+i) < numCols; i++ {
				targetWidthsMap.Set(col+i, 0)
				t.logger.Debug("  -> Set %s col %d width to 0 (part of merge)", position, col+i)
				processedCols[col+i] = true
			}
		}
	}
	ctx.logger.Debug("applyHorizontalMergeWidths: Final widths for %s: %v", position, targetWidthsMap)
}

// applyVerticalMerges applies vertical merges to row content.
// Parameters ctx and mctx hold rendering and merge state.
// No return value.
func (t *Table) applyVerticalMerges(ctx *renderContext, mctx *mergeContext) {
	ctx.logger.Debug("Applying vertical merges across %d rows", len(ctx.rowLines))
	numCols := ctx.numCols

	mergeStartRow := make(map[int]int)
	mergeStartContent := make(map[int]string)

	for i := 0; i < len(ctx.rowLines); i++ {
		if i >= len(mctx.rowMerges) {
			newRowMerges := make([]map[int]tw.MergeState, i+1)
			copy(newRowMerges, mctx.rowMerges)
			for k := len(mctx.rowMerges); k <= i; k++ {
				newRowMerges[k] = make(map[int]tw.MergeState)
			}
			mctx.rowMerges = newRowMerges
			ctx.logger.Debug("Extended rowMerges to index %d", i)
		} else if mctx.rowMerges[i] == nil {
			mctx.rowMerges[i] = make(map[int]tw.MergeState)
		}

		if len(ctx.rowLines[i]) == 0 {
			continue
		}
		currentLineContent := ctx.rowLines[i]

		for col := 0; col < numCols; col++ {
			// Join all lines of the cell to compare full content
			var currentVal strings.Builder
			for _, line := range currentLineContent {
				if col < len(line) {
					currentVal.WriteString(line[col])
				}
			}
			currentValStr := currentVal.String()
			if t.config.Behavior.TrimSpace.Enabled() {
				currentValStr = strings.TrimSpace(currentValStr)
			}
			startRow, ongoingMerge := mergeStartRow[col]
			startContent := mergeStartContent[col]
			mergeState := mctx.rowMerges[i][col]

			if ongoingMerge && currentValStr == startContent && currentValStr != "" {
				mergeState.Vertical = tw.MergeStateOption{
					Present: true,
					Span:    0,
					Start:   false,
					End:     false,
				}
				mctx.rowMerges[i][col] = mergeState
				for lineIdx := range ctx.rowLines[i] {
					if col < len(ctx.rowLines[i][lineIdx]) {
						ctx.rowLines[i][lineIdx][col] = tw.Empty
					}
				}
				ctx.logger.Debug("Vertical merge continued at row %d, col %d", i, col)
			} else {
				if ongoingMerge {
					endedRow := i - 1
					if endedRow >= 0 && endedRow >= startRow {
						startState := mctx.rowMerges[startRow][col]
						startState.Vertical.Span = (endedRow - startRow) + 1
						startState.Vertical.End = startState.Vertical.Span == 1
						mctx.rowMerges[startRow][col] = startState

						endState := mctx.rowMerges[endedRow][col]
						endState.Vertical.End = true
						endState.Vertical.Span = startState.Vertical.Span
						mctx.rowMerges[endedRow][col] = endState
						ctx.logger.Debug("Vertical merge ended at row %d, col %d, span %d", endedRow, col, startState.Vertical.Span)
					}
					delete(mergeStartRow, col)
					delete(mergeStartContent, col)
				}

				if currentValStr != "" {
					mergeState.Vertical = tw.MergeStateOption{
						Present: true,
						Span:    1,
						Start:   true,
						End:     false,
					}
					mctx.rowMerges[i][col] = mergeState
					mergeStartRow[col] = i
					mergeStartContent[col] = currentValStr
					ctx.logger.Debug("Vertical merge started at row %d, col %d", i, col)
				} else if !mergeState.Horizontal.Present {
					mergeState.Vertical = tw.MergeStateOption{}
					mctx.rowMerges[i][col] = mergeState
				}
			}
		}
	}

	lastRowIdx := len(ctx.rowLines) - 1
	if lastRowIdx >= 0 {
		for col, startRow := range mergeStartRow {
			startState := mctx.rowMerges[startRow][col]
			finalSpan := (lastRowIdx - startRow) + 1
			startState.Vertical.Span = finalSpan
			startState.Vertical.End = finalSpan == 1
			mctx.rowMerges[startRow][col] = startState

			endState := mctx.rowMerges[lastRowIdx][col]
			endState.Vertical.Present = true
			endState.Vertical.End = true
			endState.Vertical.Span = finalSpan
			if startRow != lastRowIdx {
				endState.Vertical.Start = false
			}
			mctx.rowMerges[lastRowIdx][col] = endState
			ctx.logger.Debug("Vertical merge finalized at row %d, col %d, span %d", lastRowIdx, col, finalSpan)
		}
	}
	ctx.logger.Debug("Vertical merges completed")
}

// buildAdjacentCells constructs cell contexts for adjacent lines.
// Parameters include ctx, mctx, hctx, and direction (-1 for prev, +1 for next).
// Returns a map of column indices to CellContext for the adjacent line.
func (t *Table) buildAdjacentCells(ctx *renderContext, mctx *mergeContext, hctx *helperContext, direction int) map[int]tw.CellContext {
	adjCells := make(map[int]tw.CellContext)
	var adjLine []string
	var adjMerges map[int]tw.MergeState
	found := false
	adjPosition := hctx.position // Assume adjacent line is in the same section initially

	switch hctx.position {
	case tw.Header:
		targetLineIdx := hctx.lineIdx + direction
		if direction < 0 { // Previous
			if targetLineIdx >= 0 && targetLineIdx < len(ctx.headerLines) {
				adjLine = ctx.headerLines[targetLineIdx]
				adjMerges = mctx.headerMerges
				found = true
			}
		} else { // Next
			if targetLineIdx < len(ctx.headerLines) {
				adjLine = ctx.headerLines[targetLineIdx]
				adjMerges = mctx.headerMerges
				found = true
			} else if len(ctx.rowLines) > 0 && len(ctx.rowLines[0]) > 0 && len(mctx.rowMerges) > 0 {
				adjLine = ctx.rowLines[0][0]
				adjMerges = mctx.rowMerges[0]
				adjPosition = tw.Row
				found = true
			} else if len(ctx.footerLines) > 0 {
				adjLine = ctx.footerLines[0]
				adjMerges = mctx.footerMerges
				adjPosition = tw.Footer
				found = true
			}
		}
	case tw.Row:
		targetLineIdx := hctx.lineIdx + direction
		if hctx.rowIdx < 0 || hctx.rowIdx >= len(ctx.rowLines) || hctx.rowIdx >= len(mctx.rowMerges) {
			t.logger.Debug("Warning: Invalid row index %d in buildAdjacentCells", hctx.rowIdx)
			return nil
		}
		currentRowLines := ctx.rowLines[hctx.rowIdx]
		currentMerges := mctx.rowMerges[hctx.rowIdx]

		if direction < 0 { // Previous
			if targetLineIdx >= 0 && targetLineIdx < len(currentRowLines) {
				adjLine = currentRowLines[targetLineIdx]
				adjMerges = currentMerges
				found = true
			} else if targetLineIdx < 0 {
				targetRowIdx := hctx.rowIdx - 1
				if targetRowIdx >= 0 && targetRowIdx < len(ctx.rowLines) && targetRowIdx < len(mctx.rowMerges) {
					prevRowLines := ctx.rowLines[targetRowIdx]
					if len(prevRowLines) > 0 {
						adjLine = prevRowLines[len(prevRowLines)-1]
						adjMerges = mctx.rowMerges[targetRowIdx]
						found = true
					}
				} else if len(ctx.headerLines) > 0 {
					adjLine = ctx.headerLines[len(ctx.headerLines)-1]
					adjMerges = mctx.headerMerges
					adjPosition = tw.Header
					found = true
				}
			}
		} else { // Next
			if targetLineIdx >= 0 && targetLineIdx < len(currentRowLines) {
				adjLine = currentRowLines[targetLineIdx]
				adjMerges = currentMerges
				found = true
			} else if targetLineIdx >= len(currentRowLines) {
				targetRowIdx := hctx.rowIdx + 1
				if targetRowIdx < len(ctx.rowLines) && targetRowIdx < len(mctx.rowMerges) && len(ctx.rowLines[targetRowIdx]) > 0 {
					adjLine = ctx.rowLines[targetRowIdx][0]
					adjMerges = mctx.rowMerges[targetRowIdx]
					found = true
				} else if len(ctx.footerLines) > 0 {
					adjLine = ctx.footerLines[0]
					adjMerges = mctx.footerMerges
					adjPosition = tw.Footer
					found = true
				}
			}
		}
	case tw.Footer:
		targetLineIdx := hctx.lineIdx + direction
		if direction < 0 { // Previous
			if targetLineIdx >= 0 && targetLineIdx < len(ctx.footerLines) {
				adjLine = ctx.footerLines[targetLineIdx]
				adjMerges = mctx.footerMerges
				found = true
			} else if targetLineIdx < 0 {
				if len(ctx.rowLines) > 0 {
					lastRowIdx := len(ctx.rowLines) - 1
					if lastRowIdx < len(mctx.rowMerges) && len(ctx.rowLines[lastRowIdx]) > 0 {
						lastRowLines := ctx.rowLines[lastRowIdx]
						adjLine = lastRowLines[len(lastRowLines)-1]
						adjMerges = mctx.rowMerges[lastRowIdx]
						adjPosition = tw.Row
						found = true
					}
				} else if len(ctx.headerLines) > 0 {
					adjLine = ctx.headerLines[len(ctx.headerLines)-1]
					adjMerges = mctx.headerMerges
					adjPosition = tw.Header
					found = true
				}
			}
		} else { // Next
			if targetLineIdx >= 0 && targetLineIdx < len(ctx.footerLines) {
				adjLine = ctx.footerLines[targetLineIdx]
				adjMerges = mctx.footerMerges
				found = true
			}
		}
	}

	if !found {
		return nil
	}

	if adjMerges == nil {
		adjMerges = make(map[int]tw.MergeState)
		t.logger.Debug("Warning: adjMerges was nil in buildAdjacentCells despite found=true")
	}

	paddedAdjLine := padLine(adjLine, ctx.numCols)

	for j := 0; j < ctx.numCols; j++ {
		mergeState := adjMerges[j]
		cellData := paddedAdjLine[j]
		finalAdjColWidth := ctx.widths[adjPosition].Get(j)

		adjCells[j] = tw.CellContext{
			Data:  cellData,
			Merge: mergeState,
			Width: finalAdjColWidth,
		}
	}
	return adjCells
}

// buildCellContexts creates CellContext objects for a given line in batch mode.
// Parameters include ctx, mctx, hctx, aligns, and padding for rendering.
// Returns a renderMergeResponse with current, previous, and next cell contexts.
func (t *Table) buildCellContexts(ctx *renderContext, mctx *mergeContext, hctx *helperContext, aligns map[int]tw.Align, padding map[int]tw.Padding) renderMergeResponse {
	t.logger.Debug("buildCellContexts: Building contexts for position=%s, rowIdx=%d, lineIdx=%d", hctx.position, hctx.rowIdx, hctx.lineIdx)
	var merges map[int]tw.MergeState
	switch hctx.position {
	case tw.Header:
		merges = mctx.headerMerges
	case tw.Row:
		if hctx.rowIdx >= 0 && hctx.rowIdx < len(mctx.rowMerges) && mctx.rowMerges[hctx.rowIdx] != nil {
			merges = mctx.rowMerges[hctx.rowIdx]
		} else {
			merges = make(map[int]tw.MergeState)
			t.logger.Warn("buildCellContexts: Invalid row index %d or nil merges for row", hctx.rowIdx)
		}
	case tw.Footer:
		merges = mctx.footerMerges
	default:
		merges = make(map[int]tw.MergeState)
		t.logger.Warn("buildCellContexts: Invalid position '%s'", hctx.position)
	}

	cells := t.buildCoreCellContexts(hctx.line, merges, ctx.widths[hctx.position], aligns, padding, ctx.numCols)
	return renderMergeResponse{
		cells:     cells,
		prevCells: t.buildAdjacentCells(ctx, mctx, hctx, -1),
		nextCells: t.buildAdjacentCells(ctx, mctx, hctx, +1),
		location:  hctx.location,
	}
}

// buildCoreCellContexts constructs CellContext objects for a single line, shared between batch and streaming modes.
// Parameters:
// - line: The content of the current line (padded to numCols).
// - merges: Merge states for the line's columns (map[int]tw.MergeState).
// - widths: Column widths (tw.Mapper[int, int]).
// - aligns: Column alignments (map[int]tw.Align).
// - padding: Column padding settings (map[int]tw.Padding).
// - numCols: Number of columns to process.
// Returns a map of column indices to CellContext for the current line.
func (t *Table) buildCoreCellContexts(line []string, merges map[int]tw.MergeState, widths tw.Mapper[int, int], aligns map[int]tw.Align, padding map[int]tw.Padding, numCols int) map[int]tw.CellContext {
	cells := make(map[int]tw.CellContext)
	paddedLine := padLine(line, numCols)
	for j := 0; j < numCols; j++ {
		cellData := paddedLine[j]
		mergeState := tw.MergeState{}
		if merges != nil {
			if state, ok := merges[j]; ok {
				mergeState = state
			}
		}
		cells[j] = tw.CellContext{
			Data:    cellData,
			Align:   aligns[j],
			Padding: padding[j],
			Width:   widths.Get(j),
			Merge:   mergeState,
		}
	}
	t.logger.Debug("buildCoreCellContexts: Built cell contexts for %d columns", numCols)
	return cells
}

// buildPaddingLineContents constructs a padding line for a given section, respecting column widths and horizontal merges.
// It generates a []string where each element is the padding content for a column, using the specified padChar.
func (t *Table) buildPaddingLineContents(padChar string, widths tw.Mapper[int, int], numCols int, merges map[int]tw.MergeState) []string {
	line := make([]string, numCols)
	padWidth := tw.DisplayWidth(padChar)
	if padWidth < 1 {
		padWidth = 1
	}
	for j := 0; j < numCols; j++ {
		mergeState := tw.MergeState{}
		if merges != nil {
			if state, ok := merges[j]; ok {
				mergeState = state
			}
		}
		if mergeState.Horizontal.Present && !mergeState.Horizontal.Start {
			line[j] = tw.Empty
			continue
		}
		colWd := widths.Get(j)
		repeatCount := 0
		if colWd > 0 && padWidth > 0 {
			repeatCount = colWd / padWidth
		}
		if colWd > 0 && repeatCount < 1 {
			repeatCount = 1
		}
		content := strings.Repeat(padChar, repeatCount)
		line[j] = content
	}
	if t.logger.Enabled() {
		t.logger.Debug("Built padding line with char '%s' for %d columns", padChar, numCols)
	}
	return line
}

// calculateAndNormalizeWidths computes and normalizes column widths.
// Parameter ctx holds rendering state with width maps.
// Returns an error if width calculation fails.
func (t *Table) calculateAndNormalizeWidths(ctx *renderContext) error {
	ctx.logger.Debug("calculateAndNormalizeWidths: Computing and normalizing widths for %d columns", ctx.numCols)

	// Initialize width maps
	t.headerWidths = tw.NewMapper[int, int]()
	t.rowWidths = tw.NewMapper[int, int]()
	t.footerWidths = tw.NewMapper[int, int]()

	// Compute header widths
	for _, lines := range ctx.headerLines {
		t.updateWidths(lines, t.headerWidths, t.config.Header.Padding)
	}
	ctx.logger.Debug("Initial Header widths: %v", t.headerWidths)

	// Cache row widths to avoid re-iteration
	rowWidthCache := make([]tw.Mapper[int, int], len(ctx.rowLines))
	for i, row := range ctx.rowLines {
		rowWidthCache[i] = tw.NewMapper[int, int]()
		for _, line := range row {
			t.updateWidths(line, rowWidthCache[i], t.config.Row.Padding)
			// Aggregate into t.rowWidths
			for col, width := range rowWidthCache[i] {
				currentMax, _ := t.rowWidths.OK(col)
				if width > currentMax {
					t.rowWidths.Set(col, width)
				}
			}
		}
	}
	ctx.logger.Debug("Initial Row widths: %v", t.rowWidths)

	// Compute footer widths
	for _, lines := range ctx.footerLines {
		t.updateWidths(lines, t.footerWidths, t.config.Footer.Padding)
	}
	ctx.logger.Debug("Initial Footer widths: %v", t.footerWidths)

	// Initialize width maps for normalization
	ctx.widths[tw.Header] = tw.NewMapper[int, int]()
	ctx.widths[tw.Row] = tw.NewMapper[int, int]()
	ctx.widths[tw.Footer] = tw.NewMapper[int, int]()

	// Normalize widths by taking the maximum across sections
	for i := 0; i < ctx.numCols; i++ {
		maxWidth := 0
		for _, w := range []tw.Mapper[int, int]{t.headerWidths, t.rowWidths, t.footerWidths} {
			if wd := w.Get(i); wd > maxWidth {
				maxWidth = wd
			}
		}
		ctx.widths[tw.Header].Set(i, maxWidth)
		ctx.widths[tw.Row].Set(i, maxWidth)
		ctx.widths[tw.Footer].Set(i, maxWidth)
	}
	ctx.logger.Debug("Normalized widths: header=%v, row=%v, footer=%v", ctx.widths[tw.Header], ctx.widths[tw.Row], ctx.widths[tw.Footer])
	return nil
}

// calculateContentMaxWidth computes the maximum content width for a column, accounting for padding and mode-specific constraints.
// Returns the effective content width (after subtracting padding) for the given column index.
func (t *Table) calculateContentMaxWidth(colIdx int, config tw.CellConfig, padLeftWidth, padRightWidth int, isStreaming bool) int {

	var effectiveContentMaxWidth int
	if isStreaming {
		totalColumnWidthFromStream := t.streamWidths.Get(colIdx)
		if totalColumnWidthFromStream < 0 {
			totalColumnWidthFromStream = 0
		}
		effectiveContentMaxWidth = totalColumnWidthFromStream - padLeftWidth - padRightWidth
		if effectiveContentMaxWidth < 1 && totalColumnWidthFromStream > (padLeftWidth+padRightWidth) {
			effectiveContentMaxWidth = 1
		} else if effectiveContentMaxWidth < 0 {
			effectiveContentMaxWidth = 0
		}
		if totalColumnWidthFromStream == 0 {
			effectiveContentMaxWidth = 0
		}
		t.logger.Debug("calculateContentMaxWidth: Streaming col %d, TotalColWd=%d, PadL=%d, PadR=%d -> ContentMaxWd=%d",
			colIdx, totalColumnWidthFromStream, padLeftWidth, padRightWidth, effectiveContentMaxWidth)
	} else {
		hasConstraint := false
		constraintTotalCellWidth := 0
		if config.ColMaxWidths.PerColumn != nil {
			if colMax, ok := config.ColMaxWidths.PerColumn.OK(colIdx); ok && colMax > 0 {
				constraintTotalCellWidth = colMax
				hasConstraint = true
				t.logger.Debug("calculateContentMaxWidth: Batch col %d using config.ColMaxWidths.PerColumn (as total cell width constraint): %d", colIdx, constraintTotalCellWidth)
			}
		}
		if !hasConstraint && config.ColMaxWidths.Global > 0 {
			constraintTotalCellWidth = config.ColMaxWidths.Global
			hasConstraint = true
			t.logger.Debug("calculateContentMaxWidth: Batch col %d using config.Formatting.MaxWidth (as total cell width constraint): %d", colIdx, constraintTotalCellWidth)
		}
		if !hasConstraint && t.config.MaxWidth > 0 && config.Formatting.AutoWrap != tw.WrapNone {
			constraintTotalCellWidth = t.config.MaxWidth
			hasConstraint = true
			t.logger.Debug("calculateContentMaxWidth: Batch col %d using t.config.MaxWidth (as total cell width constraint, due to AutoWrap != WrapNone): %d", colIdx, constraintTotalCellWidth)
		}
		if hasConstraint {
			effectiveContentMaxWidth = constraintTotalCellWidth - padLeftWidth - padRightWidth
			if effectiveContentMaxWidth < 1 && constraintTotalCellWidth > (padLeftWidth+padRightWidth) {
				effectiveContentMaxWidth = 1
			} else if effectiveContentMaxWidth < 0 {
				effectiveContentMaxWidth = 0
			}
			t.logger.Debug("calculateContentMaxWidth: Batch col %d, ConstraintTotalCellWidth=%d, PadL=%d, PadR=%d -> EffectiveContentMaxWidth=%d",
				colIdx, constraintTotalCellWidth, padLeftWidth, padRightWidth, effectiveContentMaxWidth)
		} else {
			effectiveContentMaxWidth = 0
			t.logger.Debug("calculateContentMaxWidth: Batch col %d, No applicable MaxWidth constraint. EffectiveContentMaxWidth set to 0 (unlimited for this stage).", colIdx)
		}
	}
	return effectiveContentMaxWidth
}

// convertToStringer invokes the table's stringer function with optional caching.
func (t *Table) convertToStringer(input interface{}) ([]string, error) {
	// Try type assertion for common case: func(interface{}) []string
	if fn, ok := t.stringer.(func(interface{}) []string); ok {
		t.logger.Debug("convertToStringer: Using type-asserted func(interface{}) []string for input type %T", input)
		return fn(input), nil
	}

	// Fallback to reflection with caching
	inputType := reflect.TypeOf(input)

	if t.stringerCacheEnabled {
		t.stringerCacheMu.RLock()
		if rv, ok := t.stringerCache[inputType]; ok {
			t.stringerCacheMu.RUnlock()
			t.logger.Debug("convertToStringer: Cache hit for type %v", inputType)
			result := rv.Call([]reflect.Value{reflect.ValueOf(input)})
			cells, ok := result[0].Interface().([]string)
			if !ok {
				return nil, errors.Newf("cached stringer for type %T did not return []string", input)
			}
			return cells, nil
		}
		t.stringerCacheMu.RUnlock()
	}

	t.logger.Debug("convertToStringer: Cache miss or caching disabled, using reflection for type %v", inputType)
	rv := reflect.ValueOf(t.stringer)
	stringerType := rv.Type()
	if !(rv.Kind() == reflect.Func && stringerType.NumIn() == 1 && stringerType.NumOut() == 1 &&
		inputType.AssignableTo(stringerType.In(0)) &&
		stringerType.Out(0) == reflect.TypeOf([]string{})) {
		return nil, errors.Newf("stringer must be func(T) []string where T is assignable from %T, got %T", input, t.stringer)
	}

	if t.stringerCacheEnabled {
		t.stringerCacheMu.Lock()
		t.stringerCache[inputType] = rv
		t.stringerCacheMu.Unlock()
	}

	result := rv.Call([]reflect.Value{reflect.ValueOf(input)})
	cells, ok := result[0].Interface().([]string)
	if !ok {
		return nil, errors.Newf("stringer for type %T did not return []string", input)
	}
	return cells, nil
}

// convertToString converts a value to its string representation.
func (t *Table) convertToString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case tw.Formatter:
		return v.Format()
	case io.Reader:
		const maxReadSize = 512
		var buf strings.Builder
		_, err := io.CopyN(&buf, v, maxReadSize)
		if err != nil && err != io.EOF {
			return fmt.Sprintf("[reader error: %v]", err) // Keep fmt.Sprintf for rare error case
		}
		if buf.Len() == maxReadSize {
			buf.WriteString("...")
		}
		return buf.String()
	case sql.NullString:
		if v.Valid {
			return v.String
		}
		return ""
	case sql.NullInt64:
		if v.Valid {
			return strconv.FormatInt(v.Int64, 10)
		}
		return ""
	case sql.NullFloat64:
		if v.Valid {
			return strconv.FormatFloat(v.Float64, 'f', -1, 64)
		}
		return ""
	case sql.NullBool:
		if v.Valid {
			return strconv.FormatBool(v.Bool)
		}
		return ""
	case sql.NullTime:
		if v.Valid {
			return v.Time.String()
		}
		return ""
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		t.logger.Debug("convertToString: Falling back to fmt.Sprintf for type %T", value)
		return fmt.Sprintf("%v", value) // Fallback for rare types
	}
}

// convertCellsToStrings converts a row to its raw string representation using specified cell config for filters.
// 'rowInput' can be []string, []any, or a custom type if t.stringer is set.
func (t *Table) convertCellsToStrings(rowInput interface{}, cellCfg tw.CellConfig) ([]string, error) {
	t.logger.Debug("convertCellsToStrings: Converting row: %v (type: %T)", rowInput, rowInput)

	var cells []string
	var err error

	switch v := rowInput.(type) {
	case []string:
		cells = v

	case []interface{}:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = t.convertToString(val)
		}

	// Integer types
	case []int:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.Itoa(val)
		}

	case []int8:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatInt(int64(val), 10)
		}

	case []int16:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatInt(int64(val), 10)
		}

	case []int32:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatInt(int64(val), 10)
		}

	case []int64:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatInt(val, 10)
		}

	// Unsigned integer types
	case []uint:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatUint(uint64(val), 10)
		}

	case []uint8: // Also handles []byte
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatUint(uint64(val), 10)
		}

	case []uint16:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatUint(uint64(val), 10)
		}

	case []uint32:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatUint(uint64(val), 10)
		}

	case []uint64:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatUint(val, 10)
		}

	// Floating point types
	case []float32:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatFloat(float64(val), 'f', -1, 32)
		}

	case []float64:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatFloat(val, 'f', -1, 64)
		}

	// Boolean type
	case []bool:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = strconv.FormatBool(val)
		}

	// Formatter cases
	case tw.Formatter:
		t.logger.Debug("convertCellsToStrings: Input is tw.Formatter, using Format()")
		cells = []string{v.Format()}

	case []tw.Formatter:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = val.Format()
		}

	// Stringer cases
	case fmt.Stringer:
		t.logger.Debug("convertCellsToStrings: Input is fmt.Stringer, using String()")
		cells = []string{v.String()}

	case []fmt.Stringer:
		cells = make([]string, len(v))
		for i, val := range v {
			cells[i] = val.String()
		}

	default:
		// Fallback to stringer with reflection
		t.logger.Debug("convertCellsToStrings: Attempting conversion using custom stringer for type %T", rowInput)
		cells, err = t.convertToStringer(rowInput)
		if err != nil {
			t.logger.Debug("convertCellsToStrings: Stringer error: %v", err)
			return nil, err
		}
	}

	// Apply filters if any
	if cellCfg.Filter.Global != nil {
		t.logger.Debug("convertCellsToStrings: Applying global filter to cells: %v", cells)
		cells = cellCfg.Filter.Global(cells)
	}

	if len(cellCfg.Filter.PerColumn) > 0 {
		t.logger.Debug("convertCellsToStrings: Applying per-column filters to cells")
		numFilters := len(cellCfg.Filter.PerColumn)
		limit := numFilters
		if len(cells) < limit {
			limit = len(cells)
		}
		for i := 0; i < limit; i++ {
			if t.config.Row.Filter.PerColumn[i] != nil {
				originalCell := cells[i]
				cells[i] = t.config.Row.Filter.PerColumn[i](cells[i])
				if cells[i] != originalCell {
					t.logger.Debug("  convertCellsToStrings: Col %d filter applied: '%s' -> '%s'", i, originalCell, cells[i])
				}
			}
		}
	}

	t.logger.Debug("convertCellsToStrings: Conversion and filtering completed, raw cells: %v", cells)
	return cells, nil
}

// determineLocation determines the boundary location for a line.
// Parameters include lineIdx, totalLines, topPad, and bottomPad.
// Returns a tw.Location indicating First, Middle, or End.
func (t *Table) determineLocation(lineIdx, totalLines int, topPad, bottomPad string) tw.Location {
	if lineIdx == 0 && topPad == tw.Empty {
		return tw.LocationFirst
	}
	if lineIdx == totalLines-1 && bottomPad == tw.Empty {
		return tw.LocationEnd
	}
	return tw.LocationMiddle
}

// ensureStreamWidthsCalculated ensures that stream widths and column count are initialized for streaming mode.
// It uses sampleData and sectionConfig to calculate widths if not already set.
// Returns an error if the column count cannot be determined.
func (t *Table) ensureStreamWidthsCalculated(sampleData []string, sectionConfig tw.CellConfig) error {
	if t.streamWidths != nil && t.streamWidths.Len() > 0 {
		t.logger.Debug("Stream widths already set: %v", t.streamWidths)
		return nil
	}
	t.streamCalculateWidths(sampleData, sectionConfig)
	if t.streamNumCols == 0 {
		t.logger.Warn("Failed to determine column count from sample data")
		return errors.New("failed to determine column count for streaming")
	}
	for i := 0; i < t.streamNumCols; i++ {
		if _, ok := t.streamWidths.OK(i); !ok {
			t.streamWidths.Set(i, 0)
		}
	}
	t.logger.Debug("Initialized stream widths: %v", t.streamWidths)
	return nil
}

// getColMaxWidths retrieves maximum column widths for a section.
// Parameter position specifies the section (Header, Row, Footer).
// Returns a map of column indices to maximum widths.
func (t *Table) getColMaxWidths(position tw.Position) tw.CellWidth {
	switch position {
	case tw.Header:
		return t.config.Header.ColMaxWidths
	case tw.Row:
		return t.config.Row.ColMaxWidths
	case tw.Footer:
		return t.config.Footer.ColMaxWidths
	default:
		return tw.CellWidth{}
	}
}

// getEmptyColumnInfo identifies empty columns in row data.
// Parameter numOriginalCols specifies the total column count.
// Returns a boolean slice (true for empty) and visible column count.
func (t *Table) getEmptyColumnInfo(numOriginalCols int) (isEmpty []bool, visibleColCount int) {
	isEmpty = make([]bool, numOriginalCols)
	for i := range isEmpty {
		isEmpty[i] = true
	}

	if t.config.Behavior.AutoHide.Disabled() {
		t.logger.Debug("getEmptyColumnInfo: AutoHide disabled, marking all %d columns as visible.", numOriginalCols)
		for i := range isEmpty {
			isEmpty[i] = false
		}
		visibleColCount = numOriginalCols
		return isEmpty, visibleColCount
	}

	t.logger.Debug("getEmptyColumnInfo: Checking %d rows for %d columns...", len(t.rows), numOriginalCols)

	for rowIdx, logicalRow := range t.rows {
		for lineIdx, visualLine := range logicalRow {
			for colIdx, cellContent := range visualLine {
				if colIdx >= numOriginalCols {
					continue
				}
				if !isEmpty[colIdx] {
					continue
				}

				if t.config.Behavior.TrimSpace.Enabled() {
					cellContent = strings.TrimSpace(cellContent)
				}

				if cellContent != "" {
					isEmpty[colIdx] = false
					t.logger.Debug("getEmptyColumnInfo: Found content in row %d, line %d, col %d ('%s'). Marked as not empty.", rowIdx, lineIdx, colIdx, cellContent)
				}
			}
		}
	}

	visibleColCount = 0
	for _, empty := range isEmpty {
		if !empty {
			visibleColCount++
		}
	}

	t.logger.Debug("getEmptyColumnInfo: Detection complete. isEmpty: %v, visibleColCount: %d", isEmpty, visibleColCount)
	return isEmpty, visibleColCount
}

// getNumColsToUse determines the number of columns to use for rendering, based on streaming or batch mode.
// Returns the number of columns (streamNumCols for streaming, maxColumns for batch).
func (t *Table) getNumColsToUse() int {
	if t.config.Stream.Enable && t.hasPrinted {
		t.logger.Debug("getNumColsToUse: Using streamNumCols: %d", t.streamNumCols)
		return t.streamNumCols
	}
	numCols := t.maxColumns()
	t.logger.Debug("getNumColsToUse: Using maxColumns: %d", numCols)
	return numCols
}

// prepareTableSection prepares either headers or footers for the table
func (t *Table) prepareTableSection(elements []any, config tw.CellConfig, sectionName string) [][]string {
	actualCellsToProcess := t.processVariadicElements(elements)
	t.logger.Debug("%s(): Effective cells to process: %v", sectionName, actualCellsToProcess)

	stringsResult, err := t.convertCellsToStrings(actualCellsToProcess, config)
	if err != nil {
		t.logger.Error("%s(): Failed to convert elements to strings: %v", sectionName, err)
		stringsResult = []string{}
	}

	prepared := t.prepareContent(stringsResult, config)
	numColsBatch := t.maxColumns()

	if len(prepared) > 0 {
		for i := range prepared {
			if len(prepared[i]) < numColsBatch {
				t.logger.Debug("Padding %s line %d from %d to %d columns", sectionName, i, len(prepared[i]), numColsBatch)
				paddedLine := make([]string, numColsBatch)
				copy(paddedLine, prepared[i])
				for j := len(prepared[i]); j < numColsBatch; j++ {
					paddedLine[j] = tw.Empty
				}
				prepared[i] = paddedLine
			} else if len(prepared[i]) > numColsBatch {
				t.logger.Debug("Truncating %s line %d from %d to %d columns", sectionName, i, len(prepared[i]), numColsBatch)
				prepared[i] = prepared[i][:numColsBatch]
			}
		}
	}

	return prepared
}

// processVariadicElements handles the common logic for processing variadic arguments
// that could be either individual elements or a slice of elements
func (t *Table) processVariadicElements(elements []any) []any {
	// Check if the input looks like a single slice was passed
	if len(elements) == 1 {
		firstArg := elements[0]
		// Try to assert to common slice types users might pass
		switch v := firstArg.(type) {
		case []string:
			t.logger.Debug("Detected single []string argument. Unpacking it.")
			result := make([]any, len(v))
			for i, val := range v {
				result[i] = val
			}
			return result
		case []interface{}:
			t.logger.Debug("Detected single []interface{} argument. Unpacking it.")
			result := make([]any, len(v))
			for i, val := range v {
				result[i] = val
			}
			return result
		}
	}

	// Either multiple arguments were passed, or a single non-slice argument
	t.logger.Debug("Input has multiple elements, is empty, or is a single non-slice element. Using variadic elements as is.")
	return elements
}

// toStringLines converts raw cells to formatted lines for table output
func (t *Table) toStringLines(row interface{}, config tw.CellConfig) ([][]string, error) {
	cells, err := t.convertCellsToStrings(row, config)
	if err != nil {
		return nil, err
	}
	return t.prepareContent(cells, config), nil
}

// updateWidths updates the width map based on cell content and padding.
// Parameters include row content, widths map, and padding configuration.
// No return value.
func (t *Table) updateWidths(row []string, widths tw.Mapper[int, int], padding tw.CellPadding) {
	t.logger.Debug("Updating widths for row: %v", row)
	for i, cell := range row {
		colPad := padding.Global
		if i < len(padding.PerColumn) && padding.PerColumn[i] != (tw.Padding{}) {
			colPad = padding.PerColumn[i]
			t.logger.Debug("  Col %d: Using per-column padding: L:'%s' R:'%s'", i, colPad.Left, colPad.Right)
		} else {
			t.logger.Debug("  Col %d: Using global padding: L:'%s' R:'%s'", i, padding.Global.Left, padding.Global.Right)
		}

		padLeftWidth := tw.DisplayWidth(colPad.Left)
		padRightWidth := tw.DisplayWidth(colPad.Right)

		// Split cell into lines and find maximum content width
		lines := strings.Split(cell, tw.NewLine)
		contentWidth := 0
		for _, line := range lines {
			lineWidth := tw.DisplayWidth(line)
			if t.config.Behavior.TrimSpace.Enabled() {
				lineWidth = tw.DisplayWidth(strings.TrimSpace(line))
			}
			if lineWidth > contentWidth {
				contentWidth = lineWidth
			}
		}

		totalWidth := contentWidth + padLeftWidth + padRightWidth
		minRequiredPaddingWidth := padLeftWidth + padRightWidth

		if contentWidth == 0 && totalWidth < minRequiredPaddingWidth {
			t.logger.Debug("  Col %d: Empty content, ensuring width >= padding width (%d). Setting totalWidth to %d.", i, minRequiredPaddingWidth, minRequiredPaddingWidth)
			totalWidth = minRequiredPaddingWidth
		}

		if totalWidth < 1 {
			t.logger.Debug("  Col %d: Calculated totalWidth is zero, setting minimum width to 1.", i)
			totalWidth = 1
		}

		currentMax, _ := widths.OK(i)
		if totalWidth > currentMax {
			widths.Set(i, totalWidth)
			t.logger.Debug("  Col %d: Updated width from %d to %d (content:%d + padL:%d + padR:%d) for cell '%s'", i, currentMax, totalWidth, contentWidth, padLeftWidth, padRightWidth, cell)
		} else {
			t.logger.Debug("  Col %d: Width %d not greater than current max %d for cell '%s'", i, totalWidth, currentMax, cell)
		}
	}
}
