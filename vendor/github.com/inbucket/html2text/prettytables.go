package html2text

import (
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrettyTablesOptions overrides tablewriter behaviors
type PrettyTablesOptions struct {
	AutoFormatHeader bool
	AutoWrapText     bool
	// Deprecated. Tablewriter always assumes this to be `true`
	ReflowDuringAutoWrap bool
	ColWidth             int
	ColumnSeparator      string
	RowSeparator         string
	CenterSeparator      string
	HeaderAlignment      tw.Align
	FooterAlignment      tw.Align
	Alignment            tw.Align
	ColumnAlignment      tw.Alignment
	// Deprecated. Tablewriter always assumes this to be `\n`
	NewLine        string
	HeaderLine     bool
	RowLine        bool
	AutoMergeCells bool
	Borders        Border
	// Configuration allows to directly manipulate the `Table` with all what [tablewriter] offers.
	// Setting this ignores all the rest of the settings of this struct.
	Configuration func(table *tablewriter.Table)
}

// NewPrettyTablesOptions creates PrettyTablesOptions with default settings
func NewPrettyTablesOptions() *PrettyTablesOptions {
	return &PrettyTablesOptions{
		AutoFormatHeader: true,
		AutoWrapText:     true,
		ColWidth:         32, // old tablewriter.MAX_ROW_WIDTH + borders now count into width
		ColumnSeparator:  defaultBorderStyle.ColumnSeparator,
		RowSeparator:     defaultBorderStyle.RowSeparator,
		CenterSeparator:  defaultBorderStyle.CenterSeparator,
		HeaderAlignment:  tw.AlignCenter,
		FooterAlignment:  tw.AlignCenter,
		Alignment:        tw.AlignDefault,
		ColumnAlignment:  make(tw.Alignment, 0),
		HeaderLine:       true,
		RowLine:          false,
		AutoMergeCells:   false,
		Borders:          Border{Left: true, Right: true, Bottom: true, Top: true},
	}
}

func (p *PrettyTablesOptions) configureTable(table *tablewriter.Table) {
	if p.Configuration != nil {
		p.Configuration(table)
		return
	}

	cfg := tablewriter.NewConfigBuilder()

	cfg.WithHeaderAutoFormat(asState(p.AutoFormatHeader)).WithFooterAutoFormat(asState(p.AutoFormatHeader)).
		WithRowAutoWrap(p.wrapMode()).WithHeaderAutoWrap(p.wrapMode()).WithFooterAutoWrap(p.wrapMode()).
		WithRowMaxWidth(p.ColWidth).
		WithHeaderAlignment(p.HeaderAlignment).
		WithFooterAlignment(p.FooterAlignment).
		WithRowAlignment(p.Alignment).
		WithRowMergeMode(p.mergeMode())

	if len(p.ColumnAlignment) > 0 {
		cfg.Row().Alignment().WithPerColumn(p.ColumnAlignment)
	}

	rendition := tw.Rendition{
		Borders:  p.Borders.withStates(),
		Symbols:  p.borderStyle(),
		Settings: p.renderSettings(),
	}

	table.Options(
		tablewriter.WithConfig(cfg.Build()),
		tablewriter.WithRendition(rendition))
}

func (p *PrettyTablesOptions) wrapMode() int {
	if p.AutoWrapText {
		return tw.WrapNormal
	} else {
		return tw.WrapNone
	}
}

func (p *PrettyTablesOptions) mergeMode() int {
	if p.AutoMergeCells {
		return tw.MergeVertical
	} else {
		return tw.MergeNone
	}
}

func (p *PrettyTablesOptions) renderSettings() tw.Settings {
	return tw.Settings{
		Lines: tw.Lines{
			ShowHeaderLine: asState(p.HeaderLine),
		},
		Separators: tw.Separators{
			BetweenRows: asState(p.RowLine),
		},
	}
}

// Border controls tablewriter borders. It uses simple bools instead of tablewriters `State`
type Border struct {
	Left, Right, Bottom, Top bool
}

func (b Border) withStates() tw.Border {
	return tw.Border{
		Left:   asState(b.Left),
		Right:  asState(b.Right),
		Bottom: asState(b.Bottom),
		Top:    asState(b.Top),
	}
}

type BorderStyle struct {
	ColumnSeparator string
	RowSeparator    string
	CenterSeparator string
}

func (b BorderStyle) Name() string {
	return "html2text"
}

func (b BorderStyle) Center() string {
	return b.CenterSeparator
}

func (b BorderStyle) Row() string {
	return b.RowSeparator
}

func (b BorderStyle) Column() string {
	return b.ColumnSeparator
}

func (b BorderStyle) TopLeft() string {
	return b.CenterSeparator
}

func (b BorderStyle) TopMid() string {
	return b.CenterSeparator
}

func (b BorderStyle) TopRight() string {
	return b.CenterSeparator
}

func (b BorderStyle) MidLeft() string {
	return b.CenterSeparator
}

func (b BorderStyle) MidRight() string {
	return b.CenterSeparator
}

func (b BorderStyle) BottomLeft() string {
	return b.CenterSeparator
}

func (b BorderStyle) BottomMid() string {
	return b.CenterSeparator
}

func (b BorderStyle) BottomRight() string {
	return b.CenterSeparator
}

func (b BorderStyle) HeaderLeft() string {
	return b.CenterSeparator
}

func (b BorderStyle) HeaderMid() string {
	return b.CenterSeparator
}

func (b BorderStyle) HeaderRight() string {
	return b.CenterSeparator
}

var defaultBorderStyle = BorderStyle{
	ColumnSeparator: "|",
	RowSeparator:    "-",
	CenterSeparator: "+",
}

func (p *PrettyTablesOptions) borderStyle() BorderStyle {
	return BorderStyle{
		ColumnSeparator: p.ColumnSeparator,
		RowSeparator:    p.RowSeparator,
		CenterSeparator: p.CenterSeparator,
	}
}

func asState(b bool) tw.State {
	// TableWriter does not provide this by default :(
	if b {
		return tw.On
	} else {
		return tw.Off
	}
}
