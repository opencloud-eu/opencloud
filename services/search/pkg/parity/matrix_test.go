package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	sprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	searchMessage "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/search"
)

const (
	matrixFile  = "README.md"
	matrixEntry = "engine parity"
)

// matrixRow is one line of the README. It travels in a report entry, which
// crosses processes as JSON under ginkgo -p, hence the exported fields and
// the scope already rendered to a string.
type matrixRow struct {
	Section        string
	Group          string
	ID             string
	Query          string
	Scope          string
	Limit          int32
	Reads          string
	Context        string
	Want           []string
	WantCount      *int
	WantBadRequest bool
	// Overrides is what an engine is held to instead, rendered like expected
	Overrides map[string]string

	GroupAt, CaseAt, QueryAt int
}

func (r matrixRow) key() string {
	return fmt.Sprintf("%s/%s/%s/%d", r.Group, r.ID, r.Query, r.QueryAt)
}

// matrixAnswer is what a spec attaches to its report: one engine's answer to
// one row. A skipped engine leaves the README alone.
type matrixAnswer struct {
	Row     matrixRow
	Engine  string
	Answer  []string
	Skipped bool
}

func recordAnswer(row matrixRow, engine string, answer []string) {
	AddReportEntry(matrixEntry, matrixAnswer{Row: row, Engine: engine, Answer: answer}, ReportEntryVisibilityNever)
}

func recordSkip(row matrixRow, engine string) {
	AddReportEntry(matrixEntry, matrixAnswer{Row: row, Engine: engine, Skipped: true}, ReportEntryVisibilityNever)
}

// matrixPlanned is every row the spec tree announced while it was built, so
// the README is only written when each of them got an answer from every engine.
var matrixPlanned []matrixRow

func planRow(rows ...matrixRow) {
	matrixPlanned = append(matrixPlanned, rows...)
}

type matrixResult struct {
	matrixRow
	answered map[string][]string
	skipped  map[string]bool
}

// matrixAnswerOf reads an entry back: in-process it still carries the value,
// from another process only its JSON.
func matrixAnswerOf(entry types.ReportEntry) (matrixAnswer, bool) {
	if answer, ok := entry.GetRawValue().(matrixAnswer); ok {
		return answer, true
	}

	var answer matrixAnswer
	if err := json.Unmarshal([]byte(entry.Value.AsJSON), &answer); err != nil {
		return matrixAnswer{}, false
	}

	return answer, true
}

func collectMatrix(report types.Report) []*matrixResult {
	results := map[string]*matrixResult{}
	for _, spec := range report.SpecReports {
		for _, entry := range spec.ReportEntries {
			if entry.Name != matrixEntry {
				continue
			}

			answer, ok := matrixAnswerOf(entry)
			if !ok {
				continue
			}

			key := answer.Row.key()
			result, known := results[key]
			if !known {
				result = &matrixResult{matrixRow: answer.Row, answered: map[string][]string{}, skipped: map[string]bool{}}
				results[key] = result
			}

			if answer.Skipped {
				result.skipped[answer.Engine] = true
				continue
			}

			result.answered[answer.Engine] = answer.Answer
		}
	}

	rows := make([]*matrixResult, 0, len(results))
	for _, result := range results {
		rows = append(rows, result)
	}

	return rows
}

func writeMatrix(report types.Report) {
	rows := collectMatrix(report)

	answered := map[string]*matrixResult{}
	for _, row := range rows {
		answered[row.key()] = row
	}

	var missing []string
	for _, planned := range matrixPlanned {
		result, ok := answered[planned.key()]
		switch {
		case !ok:
			missing = append(missing, planned.ID+" "+planned.Query)
		case len(result.skipped) > 0:
			fmt.Fprintf(os.Stderr, "%s left alone, an engine was not reachable\n", matrixFile)
			return
		case len(result.answered) != len(engineNames):
			missing = append(missing, planned.ID+" "+planned.Query)
		}
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "%s left alone, no answer from every engine for: %s\n", matrixFile, strings.Join(missing, "; "))
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.GroupAt != b.GroupAt:
			return a.GroupAt < b.GroupAt
		case a.CaseAt != b.CaseAt:
			return a.CaseAt < b.CaseAt
		default:
			return a.QueryAt < b.QueryAt
		}
	})

	out := &strings.Builder{}
	out.WriteString("# Engine parity\n\n")
	out.WriteString("Written by the parity suite (`UPDATE_SEARCH_PARITY_MATRIX=true go test ./services/search/pkg/parity/`), do not edit.\n")
	out.WriteString("Every case runs against bleve and OpenSearch. `same?` is ✅ when both answer as\n")
	out.WriteString("expected, `❌ known` when an engine's divergence is documented in the case\n")
	out.WriteString("(`engineOverrides`), `❌` when it is not. `✅ stale` when every engine\n")
	out.WriteString("answers the expected value although the case still documents a\n")
	out.WriteString("divergence, that override can come out.\n")

	group, section := "", ""
	for _, row := range rows {
		if row.Section != section {
			section = row.Section
			if out.Len() > 0 {
				out.WriteString("\n")
			}

			fmt.Fprintf(out, "## %s\n", section)
		}

		if row.Group != group {
			group = row.Group
			fmt.Fprintf(out, "\n### %s\n\n%s\n\n", group, matrixFixtures(group))
			out.WriteString("| Case | Query | expected | bleve | OpenSearch | same? |\n")
			out.WriteString("|---|---|---|---|---|---|\n")
		}

		query := "`" + shortenQuery(row.Query) + "`"
		if row.Scope != "" {
			query += " " + row.Scope
		}

		if row.Limit != 0 {
			query = fmt.Sprintf("%s with a limit of %d", query, row.Limit)
		}

		if row.Reads != "" {
			query += " reads `" + row.Reads + "`"
		}

		if row.Context != "" {
			query = row.Context + ", then " + query
		}

		expected := matrixNames(row.Want)
		switch {
		case row.WantBadRequest:
			expected = "bad request"
		case row.WantCount != nil:
			expected = fmt.Sprintf("%d items", *row.WantCount)
		}

		fmt.Fprintf(out, "| %s | %s | %s | %s | %s | %s |\n",
			row.ID, query, expected,
			matrixNames(row.answered["bleve"]), matrixNames(row.answered["opensearch"]), matrixVerdict(row))
	}

	if os.Getenv("UPDATE_SEARCH_PARITY_MATRIX") != "" {
		if err := os.WriteFile(matrixFile, []byte(out.String()), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", matrixFile, err)
		}
		return
	}

	// the committed matrix must match what the suite answers
	previous, _ := os.ReadFile(matrixFile)
	if string(previous) != out.String() {
		Fail(matrixFile + " is out of date: regenerate it with UPDATE_SEARCH_PARITY_MATRIX=true and commit the change")
	}
}

func matrixFixtures(group string) string {
	var (
		fixtures []search.Resource
		withIDs  bool
	)

	for _, g := range queryGroups() {
		if g.name != group {
			continue
		}

		fixtures = g.fixtures
		for _, c := range g.cases {
			withIDs = withIDs || strings.Contains(c.query, "id:")
		}
	}

	for _, g := range lifecycleGroups() {
		if g.name != group {
			continue
		}

		withIDs = true
		fixtures = g.fixtures
		for _, c := range g.cases {
			fixtures = append(fixtures, c.fixtures...)
		}
	}

	for _, g := range responseGroups() {
		if g.name != group {
			continue
		}

		withIDs = true
		fixtures = g.fixtures
	}

	if group == "aggregations" {
		fixtures = aggregationFixtures()
	}

	if len(fixtures) == 0 {
		return "Fixtures: none"
	}

	seen := map[string]bool{}
	lines := []string{"Fixtures:", ""}
	for _, f := range fixtures {
		line := "- `" + shorten(f.Name) + "`"
		if fields := fixtureFields(f, withIDs); fields != "" {
			line += ", " + fields
		}

		if seen[line] {
			continue
		}

		seen[line] = true
		lines = append(lines, line)
	}

	if listed := len(lines) - 2; listed > 12 {
		lines = append(lines[:12], fmt.Sprintf("- ... and %d more of the same", listed-10))
	}

	return strings.Join(lines, "\n")
}

func fixtureFields(f search.Resource, withID bool) string {
	var fields []string

	add := func(format string, args ...any) {
		fields = append(fields, fmt.Sprintf(format, args...))
	}

	if withID {
		add("ID = %s", f.ID)
	}

	if f.Type == uint64(sprovider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		add("folder")
	} else if f.MimeType != "text/plain" {
		add("MimeType = %s", f.MimeType)
	}

	if f.Path != "./"+f.Name {
		add("Path = %s", f.Path)
	}

	if f.Title != "" {
		add("Title = %q", f.Title)
	}

	if f.Content != "" {
		add("Content = %q", f.Content)
	}

	if len(f.Tags) > 0 {
		add("Tags = %s", strings.Join(f.Tags, ", "))
	}

	if len(f.Favorites) > 0 {
		add("Favorites = %s", strings.Join(f.Favorites, ", "))
	}

	if f.Size != 1000 {
		add("Size = %d", f.Size)
	}

	if f.Mtime == nil || f.Mtime.Format("2006-01-02") != fixtureNow.Format("2006-01-02") {
		add("Mtime = %s", f.Mtime.Format(time.RFC3339))
	}

	if f.Hidden {
		add("hidden")
	}

	if f.Deleted {
		add("deleted")
	}

	return strings.Join(fields, ", ")
}

func matrixScope(ref *searchMessage.Reference) string {
	if ref == nil {
		return ""
	}

	space := storagespace.FormatResourceID(&sprovider.ResourceId{
		StorageId: ref.GetResourceId().GetStorageId(),
		SpaceId:   ref.GetResourceId().GetSpaceId(),
		OpaqueId:  ref.GetResourceId().GetOpaqueId(),
	})

	if path := ref.GetPath(); path != "" {
		return "in " + space + " under " + path
	}

	return "in " + space
}

func matrixNames(names []string) string {
	if len(names) == 0 {
		return "no match"
	}

	shortened := make([]string, 0, len(names))
	for _, name := range names {
		shortened = append(shortened, shorten(name))
	}
	sort.Strings(shortened)

	return strings.Join(shortened, ", ")
}

func shorten(name string) string {
	if len(name) <= 24 {
		return name
	}

	return name[:10] + "..." + name[len(name)-8:]
}

func shortenQuery(q string) string {
	if len(q) <= 64 {
		return q
	}

	return q[:32] + "..." + q[len(q)-24:]
}

func matrixVerdict(row *matrixResult) string {
	var off []string
	known := true
	for _, engine := range engineNames {
		answer := matrixNames(row.answered[engine])

		agrees := answer == matrixNames(row.Want)
		switch {
		case row.WantBadRequest:
			agrees = answer == "bad request"
		case row.WantCount != nil:
			agrees = len(row.answered[engine]) == *row.WantCount
		}

		if agrees {
			continue
		}

		off = append(off, engine)
		if row.WantCount != nil {
			answer = fmt.Sprintf("%d items", len(row.answered[engine]))
		}

		if expected, ok := row.Overrides[engine]; !ok || expected != answer {
			known = false
		}
	}

	switch {
	case len(off) == 0 && len(row.Overrides) > 0:
		return "✅ stale"
	case len(off) == 0:
		return "✅"
	case known:
		return "❌ known"
	default:
		return "❌"
	}
}

func (c lifecycleCase) matrixRows(group string, groupAt, caseAt int) []matrixRow {
	rows := make([]matrixRow, 0, len(c.expect)+1)
	for i, expect := range c.expect {
		rows = append(rows, matrixRow{
			Section: "Operations", Group: group, ID: c.label(group), Query: expect.query, Context: c.title, Want: expect.want, Overrides: renderOverrides(c.overridesFor(expect.query)),
			GroupAt: groupAt, CaseAt: caseAt, QueryAt: i,
		})
	}

	if c.wantDocCount != nil {
		rows = append(rows, matrixRow{
			Section: "Operations", Group: group, ID: c.label(group), Query: "DocCount()", Context: c.title,
			Want:      []string{fmt.Sprint(*c.wantDocCount)},
			Overrides: renderOverrides(c.docCountOverrides()),
			GroupAt:   groupAt, CaseAt: caseAt, QueryAt: len(c.expect),
		})
	}

	return rows
}
