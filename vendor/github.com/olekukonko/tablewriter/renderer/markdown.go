package renderer

import (
	"io"
	"strings"

	"github.com/olekukonko/ll"
	"github.com/olekukonko/tablewriter/pkg/twwidth"

	"github.com/olekukonko/tablewriter/tw"
)

// Markdown renders tables in Markdown format with customizable settings.
type Markdown struct {
	config          tw.Rendition // Rendering configuration
	logger          *ll.Logger   // Debug trace messages
	headerAlignment tw.Alignment // Cached header alignments
	bodyAlignment   tw.Alignment // Cached body alignments
	bodyResolved    bool         // Whether body alignment has been resolved from actual row data
	w               io.Writer
	// Deferred separator rendering
	pendingSeparator bool
	pendingSepCtx    tw.Formatting
}

// NewMarkdown initializes a Markdown renderer with defaults tailored for Markdown.
func NewMarkdown(configs ...tw.Rendition) *Markdown {
	cfg := defaultBlueprint()
	cfg.Symbols = tw.NewSymbols(tw.StyleMarkdown)
	cfg.Borders = tw.Border{Left: tw.On, Right: tw.On, Top: tw.Off, Bottom: tw.Off}
	cfg.Settings.Separators.BetweenColumns = tw.On
	cfg.Settings.Separators.BetweenRows = tw.Off
	cfg.Settings.Lines.ShowHeaderLine = tw.On
	cfg.Settings.Lines.ShowTop = tw.Off
	cfg.Settings.Lines.ShowBottom = tw.Off
	cfg.Settings.Lines.ShowFooterLine = tw.Off

	if len(configs) > 0 {
		cfg = mergeMarkdownConfig(cfg, configs[0])
	}
	return &Markdown{config: cfg, logger: ll.New("markdown").Disable()}
}

// mergeMarkdownConfig combines user-provided config with Markdown defaults, enforcing Markdown-specific settings.
func mergeMarkdownConfig(defaults, overrides tw.Rendition) tw.Rendition {
	if overrides.Borders.Left != 0 {
		defaults.Borders.Left = overrides.Borders.Left
	}
	if overrides.Borders.Right != 0 {
		defaults.Borders.Right = overrides.Borders.Right
	}
	if overrides.Symbols != nil {
		defaults.Symbols = overrides.Symbols
	}
	defaults.Settings = mergeSettings(defaults.Settings, overrides.Settings)
	// Enforce Markdown requirements
	defaults.Settings.Lines.ShowHeaderLine = tw.On
	defaults.Settings.Separators.BetweenColumns = tw.On
	return defaults
}

func (m *Markdown) Logger(logger *ll.Logger) {
	m.logger = logger.Namespace("markdown")
}

// Config returns the renderer's current configuration.
func (m *Markdown) Config() tw.Rendition {
	return m.config
}

func (m *Markdown) Rendition(config tw.Rendition) {
	m.config = mergeRendition(m.config, config)
	m.Reset()
}

// Header renders the Markdown table header. Separator is deferred until body alignment is known.
func (m *Markdown) Header(headers [][]string, ctx tw.Formatting) {
	m.resolveHeaderAlignment(ctx)
	if len(headers) == 0 || len(headers[0]) == 0 {
		m.logger.Debug("Header: No headers to render")
		return
	}
	m.logger.Debugf("Rendering header with %d lines, widths=%v, current=%v, next=%v",
		len(headers), ctx.Row.Widths, ctx.Row.Current, ctx.Row.Next)

	// Render header content immediately
	m.renderMarkdownLine(headers[0], ctx, false)

	// Defer separator rendering until we know the body alignment
	if m.config.Settings.Lines.ShowHeaderLine.Enabled() {
		m.pendingSeparator = true
		m.pendingSepCtx = ctx
		m.logger.Debug("Header: Deferred separator rendering until body alignment is known")
	}
}

// Row renders a Markdown table data row. First row triggers deferred separator if pending.
func (m *Markdown) Row(row []string, ctx tw.Formatting) {
	m.resolveBodyAlignment(ctx)

	// Render deferred separator if pending and body alignment is now known
	if m.pendingSeparator && m.bodyResolved {
		m.renderDeferredSeparator()
	}

	m.logger.Debugf("Rendering row with data=%v, widths=%v, previous=%v, current=%v, next=%v",
		row, ctx.Row.Widths, ctx.Row.Previous, ctx.Row.Current, ctx.Row.Next)
	m.renderMarkdownLine(row, ctx, false)
}

// Footer renders the Markdown table footer.
func (m *Markdown) Footer(footers [][]string, ctx tw.Formatting) {
	m.resolveBodyAlignment(ctx)

	// Render deferred separator if still pending (no rows to trigger it)
	if m.pendingSeparator && m.bodyResolved {
		m.renderDeferredSeparator()
	}

	if len(footers) == 0 || len(footers[0]) == 0 {
		m.logger.Debug("Footer: No footers to render")
		return
	}
	m.logger.Debugf("Rendering footer with %d lines, widths=%v, previous=%v, current=%v, next=%v",
		len(footers), ctx.Row.Widths, ctx.Row.Previous, ctx.Row.Current, ctx.Row.Next)
	m.renderMarkdownLine(footers[0], ctx, false)
}

// renderDeferredSeparator renders the separator line now that body alignment is known.
func (m *Markdown) renderDeferredSeparator() {
	m.logger.Debug("Rendering deferred separator with known body alignment: %s", m.bodyAlignment)

	sepCtx := m.pendingSepCtx
	sepCtx.Row.Widths = m.pendingSepCtx.Row.Widths
	sepCtx.Row.Previous = m.pendingSepCtx.Row.Current
	sepCtx.IsSubRow = true
	sepCtx.Row.Current = m.pendingSepCtx.Row.Current // Use header context for separator rendering

	m.renderMarkdownLine(nil, sepCtx, true)
	m.pendingSeparator = false
}

func (m *Markdown) Line(ctx tw.Formatting) {
	m.logger.Debugf("Line: Generic Line call received (pos: %s, loc: %s). Markdown ignores these.",
		ctx.Row.Position, ctx.Row.Location)
}

// Reset clears the renderer's internal state, including debug traces.
func (m *Markdown) Reset() {
	m.headerAlignment = nil
	m.bodyAlignment = nil
	m.bodyResolved = false
	m.pendingSeparator = false
	m.logger.Info("Reset: Cleared alignment caches")
}

func (m *Markdown) Start(w io.Writer) error {
	m.w = w
	m.logger.Warn("Markdown.Start() called (no-op).")
	return nil
}

func (m *Markdown) Close() error {
	if !m.pendingSeparator {
		return nil
	}

	// If we have a deferred separator but body alignment was never resolved,
	// fall back to header alignment so the separator still renders.
	if !m.bodyResolved && len(m.headerAlignment) > 0 {
		m.bodyResolved = true
		m.bodyAlignment = make(tw.Alignment, len(m.headerAlignment))
		copy(m.bodyAlignment, m.headerAlignment)
	}

	if m.bodyResolved {
		m.renderDeferredSeparator()
	}

	return nil
}

func (m *Markdown) resolveHeaderAlignment(ctx tw.Formatting) tw.Alignment {
	if len(m.headerAlignment) != 0 {
		return m.headerAlignment
	}
	total := len(ctx.Row.Current)
	for i := 0; i < total; i++ {
		m.headerAlignment = append(m.headerAlignment, tw.AlignNone)
	}
	for i := 0; i < total; i++ {
		m.headerAlignment[i] = ctx.Row.Current[i].Align
	}
	m.logger.Debugf(" → Header Align Resolved %s", m.headerAlignment)
	return m.headerAlignment
}

func (m *Markdown) resolveBodyAlignment(ctx tw.Formatting) tw.Alignment {
	// Only resolve from actual row data once
	if m.bodyResolved {
		return m.bodyAlignment
	}

	total := len(ctx.Row.Current)
	if total == 0 {
		return m.bodyAlignment
	}

	// Initialize if needed
	if len(m.bodyAlignment) == 0 {
		for i := 0; i < total; i++ {
			m.bodyAlignment = append(m.bodyAlignment, tw.AlignNone)
		}
	}

	// Only update from row context if position is Row or Footer (not Header/separator)
	if ctx.Row.Position == tw.Row || ctx.Row.Position == tw.Footer {
		for i := 0; i < total && i < len(m.bodyAlignment); i++ {
			m.bodyAlignment[i] = ctx.Row.Current[i].Align
		}
		m.bodyResolved = true
		m.logger.Debugf(" → Body Align Resolved from %s: %s", ctx.Row.Position, m.bodyAlignment)
	}

	return m.bodyAlignment
}

// resolveAlignmentRule applies the Deliberate Rules:
// Rule 1: No explicit alignment → Center (backward compatible)
// Rule 2 & 3: Body has explicit alignment → Body wins
// Rule 4: Only header has explicit alignment → Header wins
func (m *Markdown) resolveAlignmentRule(headerAlign, bodyAlign tw.Align) tw.Align {
	headerExplicit := headerAlign != tw.AlignNone && headerAlign != tw.Empty && headerAlign != tw.Skip
	bodyExplicit := bodyAlign != tw.AlignNone && bodyAlign != tw.Empty && bodyAlign != tw.Skip

	if bodyExplicit {
		return bodyAlign
	} else if headerExplicit {
		return headerAlign
	}
	return tw.AlignCenter
}

// formatCell formats a Markdown cell's content with padding and alignment, ensuring at least 3 characters wide.
func (m *Markdown) formatCell(content string, width int, align tw.Align, padding tw.Padding) string {
	contentVisualWidth := twwidth.Width(content)
	padLeftChar := padding.Left
	if padLeftChar == tw.Empty {
		padLeftChar = tw.Space
	}
	padRightChar := padding.Right
	if padRightChar == tw.Empty {
		padRightChar = tw.Space
	}
	padLeftCharWidth := twwidth.Width(padLeftChar)
	padRightCharWidth := twwidth.Width(padRightChar)
	minWidth := tw.Max(3, contentVisualWidth+padLeftCharWidth+padRightCharWidth)
	targetWidth := tw.Max(width, minWidth)

	totalPaddingNeeded := max(targetWidth-contentVisualWidth, 0)

	var leftPadStr, rightPadStr string
	switch align {
	case tw.AlignRight:
		leftPadCount := tw.Max(0, totalPaddingNeeded-padRightCharWidth)
		rightPadCount := totalPaddingNeeded - leftPadCount
		leftPadStr = strings.Repeat(padLeftChar, leftPadCount)
		rightPadStr = strings.Repeat(padRightChar, rightPadCount)
	case tw.AlignCenter:
		leftPadCount := totalPaddingNeeded / 2
		rightPadCount := totalPaddingNeeded - leftPadCount
		if leftPadCount < padLeftCharWidth && totalPaddingNeeded >= padLeftCharWidth+padRightCharWidth {
			leftPadCount = padLeftCharWidth
			rightPadCount = totalPaddingNeeded - leftPadCount
		}
		if rightPadCount < padRightCharWidth && totalPaddingNeeded >= padLeftCharWidth+padRightCharWidth {
			rightPadCount = padRightCharWidth
			leftPadCount = totalPaddingNeeded - rightPadCount
		}
		leftPadStr = strings.Repeat(padLeftChar, leftPadCount)
		rightPadStr = strings.Repeat(padRightChar, rightPadCount)
	default: // AlignLeft
		rightPadCount := tw.Max(0, totalPaddingNeeded-padLeftCharWidth)
		leftPadCount := totalPaddingNeeded - rightPadCount
		leftPadStr = strings.Repeat(padLeftChar, leftPadCount)
		rightPadStr = strings.Repeat(padRightChar, rightPadCount)
	}

	result := leftPadStr + content + rightPadStr
	finalWidth := twwidth.Width(result)
	if finalWidth != targetWidth {
		m.logger.Debugf("Markdown formatCell MISMATCH: content='%s', target_w=%d, paddingL='%s', paddingR='%s', align=%s -> result='%s', result_w=%d",
			content, targetWidth, padding.Left, padding.Right, align, result, finalWidth)
		adjNeeded := targetWidth - finalWidth
		if adjNeeded > 0 {
			adjStr := strings.Repeat(tw.Space, adjNeeded)
			switch align {
			case tw.AlignRight:
				result = adjStr + result
			case tw.AlignCenter:
				leftAdj := adjNeeded / 2
				rightAdj := adjNeeded - leftAdj
				result = strings.Repeat(tw.Space, leftAdj) + result + strings.Repeat(tw.Space, rightAdj)
			default:
				result += adjStr
			}
		} else {
			result = twwidth.Truncate(result, targetWidth)
		}
		m.logger.Debugf("Markdown formatCell Corrected: target_w=%d, result='%s', new_w=%d", targetWidth, result, twwidth.Width(result))
	}

	m.logger.Debugf("Markdown formatCell: content='%s', width=%d, align=%s, paddingL='%s', paddingR='%s' -> '%s' (target %d)",
		content, width, align, padding.Left, padding.Right, result, targetWidth)
	return result
}

func (m *Markdown) formatSeparator(width int, align tw.Align) string {
	targetWidth := tw.Max(3, width)
	var sb strings.Builder

	switch align {
	case tw.AlignLeft:
		sb.WriteRune(':')
		sb.WriteString(strings.Repeat("-", targetWidth-1))
	case tw.AlignRight:
		sb.WriteString(strings.Repeat("-", targetWidth-1))
		sb.WriteRune(':')
	case tw.AlignCenter:
		sb.WriteRune(':')
		sb.WriteString(strings.Repeat("-", targetWidth-2))
		sb.WriteRune(':')
	default:
		sb.WriteRune(':')
		sb.WriteString(strings.Repeat("-", targetWidth-2))
		sb.WriteRune(':')
	}

	result := sb.String()
	currentLen := twwidth.Width(result)
	if currentLen < targetWidth {
		result += strings.Repeat("-", targetWidth-currentLen)
	} else if currentLen > targetWidth {
		result = twwidth.Truncate(result, targetWidth)
	}

	m.logger.Debugf("Markdown formatSeparator: width=%d, align=%s -> '%s'", width, align, result)
	return result
}

// renderMarkdownLine renders a single Markdown line (header, row, footer, or separator) with pipes and alignment.
func (m *Markdown) renderMarkdownLine(line []string, ctx tw.Formatting, isHeaderSep bool) {
	numCols := 0
	if len(ctx.Row.Widths) > 0 {
		maxKey := -1
		for k := range ctx.Row.Widths {
			if k > maxKey {
				maxKey = k
			}
		}
		numCols = maxKey + 1
	} else if len(ctx.Row.Current) > 0 {
		maxKey := -1
		for k := range ctx.Row.Current {
			if k > maxKey {
				maxKey = k
			}
		}
		numCols = maxKey + 1
	} else if len(line) > 0 && !isHeaderSep {
		numCols = len(line)
	}

	if numCols == 0 && !isHeaderSep {
		m.logger.Debug("renderMarkdownLine: Skipping line with zero columns.")
		return
	}

	var output strings.Builder
	prefix := m.config.Symbols.Column()
	if m.config.Borders.Left == tw.Off {
		prefix = tw.Empty
	}
	suffix := m.config.Symbols.Column()
	if m.config.Borders.Right == tw.Off {
		suffix = tw.Empty
	}
	separator := m.config.Symbols.Column()
	output.WriteString(prefix)

	colIndex := 0
	separatorWidth := twwidth.Width(separator)

	for colIndex < numCols {
		cellCtx, ok := ctx.Row.Current[colIndex]
		defaultPadding := tw.Padding{Left: tw.Space, Right: tw.Space}
		if !ok {
			cellCtx = tw.CellContext{
				Data: tw.Empty, Align: tw.AlignNone, Padding: defaultPadding,
				Width: ctx.Row.Widths.Get(colIndex), Merge: tw.MergeState{},
			}
		} else if !cellCtx.Padding.Paddable() {
			cellCtx.Padding = defaultPadding
		}

		isContinuation := ok && cellCtx.Merge.Horizontal.Present && !cellCtx.Merge.Horizontal.Start
		if colIndex > 0 && !isContinuation {
			output.WriteString(separator)
			m.logger.Debugf("renderMarkdownLine: Added separator '%s' before col %d", separator, colIndex)
		}

		span := 1
		visualWidth := 0
		isHMergeStart := ok && cellCtx.Merge.Horizontal.Present && cellCtx.Merge.Horizontal.Start
		if isHMergeStart {
			span = cellCtx.Merge.Horizontal.Span
			totalWidth := 0
			for k := 0; k < span && colIndex+k < numCols; k++ {
				colWidth := max(ctx.NormalizedWidths.Get(colIndex+k), 0)
				totalWidth += colWidth
				if k > 0 && separatorWidth > 0 {
					totalWidth += separatorWidth
				}
			}
			visualWidth = totalWidth
			m.logger.Debugf("renderMarkdownLine: HMerge col %d, span %d, visualWidth %d", colIndex, span, visualWidth)
		} else {
			visualWidth = ctx.Row.Widths.Get(colIndex)
			m.logger.Debugf("renderMarkdownLine: Regular col %d, visualWidth %d", colIndex, visualWidth)
		}
		if visualWidth < 0 {
			visualWidth = 0
		}

		if isContinuation {
			m.logger.Debugf("renderMarkdownLine: Skipping col %d (HMerge continuation)", colIndex)
			colIndex += span
			continue
		}

		var formattedSegment string
		if isHeaderSep {
			// Separator: use body alignment (if resolved) else header alignment
			headerAlign := tw.AlignNone
			if colIndex < len(m.headerAlignment) {
				headerAlign = m.headerAlignment[colIndex]
			}
			bodyAlign := tw.AlignNone
			if m.bodyResolved && colIndex < len(m.bodyAlignment) {
				bodyAlign = m.bodyAlignment[colIndex]
			}
			sepAlign := m.resolveAlignmentRule(headerAlign, bodyAlign)
			formattedSegment = m.formatSeparator(visualWidth, sepAlign)
			m.logger.Debugf("renderMarkdownLine: Separator col %d - headerAlign=%s, bodyAlign=%s, final=%s",
				colIndex, headerAlign, bodyAlign, sepAlign)
		} else {
			content := ""
			if colIndex < len(line) {
				content = line[colIndex]
			}
			if ctx.Row.Position == tw.Header {
				// Header content uses its own alignment
				headerAlign := tw.AlignNone
				if colIndex < len(m.headerAlignment) {
					headerAlign = m.headerAlignment[colIndex]
				}
				if headerAlign == tw.AlignNone || headerAlign == tw.Empty || headerAlign == tw.Skip {
					headerAlign = tw.AlignCenter
				}
				formattedSegment = m.formatCell(content, visualWidth, headerAlign, cellCtx.Padding)
				m.logger.Debugf("renderMarkdownLine: Header col %d - align=%s", colIndex, headerAlign)
			} else {
				// Body/footer: apply rules
				headerAlign := tw.AlignNone
				if colIndex < len(m.headerAlignment) {
					headerAlign = m.headerAlignment[colIndex]
				}
				bodyAlign := tw.AlignNone
				if m.bodyResolved && colIndex < len(m.bodyAlignment) {
					bodyAlign = m.bodyAlignment[colIndex]
				}
				rowAlign := m.resolveAlignmentRule(headerAlign, bodyAlign)
				formattedSegment = m.formatCell(content, visualWidth, rowAlign, cellCtx.Padding)
				m.logger.Debugf("renderMarkdownLine: Row col %d - headerAlign=%s, bodyAlign=%s, final=%s",
					colIndex, headerAlign, bodyAlign, rowAlign)
			}
		}
		output.WriteString(formattedSegment)
		m.logger.Debugf("renderMarkdownLine: Wrote col %d (span %d, width %d): '%s'",
			colIndex, span, visualWidth, formattedSegment)
		colIndex += span
	}

	output.WriteString(suffix)
	output.WriteString(tw.NewLine)
	m.w.Write([]byte(output.String()))
	m.logger.Debugf("renderMarkdownLine: Final line: %s", strings.TrimSuffix(output.String(), tw.NewLine))
}

var _ tw.Renditioning = (*Markdown)(nil)
