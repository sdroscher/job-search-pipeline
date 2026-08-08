// This test reflects over unexported request structs, so it lives in package
// api rather than api_test.
//
//nolint:testpackage
package api

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commandDoc is the slash command that drives this API. Its "API contract"
// section is the only schema the agent running the command ever sees, so a
// field added here and not there is a field the agent will never send.
const commandDoc = "../../.claude/commands/job-search.md"

// jsonFields returns the sorted JSON field names of a struct.
func jsonFields(v any) []string {
	typ := reflect.TypeOf(v)

	var names []string

	for field := range typ.Fields() {
		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if tag != "" && tag != "-" {
			names = append(names, tag)
		}
	}

	slices.Sort(names)

	return names
}

// isTableRow reports whether line is a markdown table row naming a field, and
// returns that field name. Header and |---| separator rows return false.
func isTableRow(line string) (string, bool) {
	if !strings.HasPrefix(line, "|") {
		return "", false
	}

	cell := strings.TrimSpace(strings.Split(strings.Trim(line, "|"), "|")[0])
	if !strings.HasPrefix(cell, "`") {
		return "", false
	}

	return strings.Trim(cell, "`"), true
}

// docFields extracts the sorted field names from the markdown table following
// the `<!-- schema:name -->` marker.
func docFields(t *testing.T, doc, name string) []string {
	t.Helper()

	_, after, found := strings.Cut(doc, "<!-- schema:"+name+" -->")
	require.True(t, found, "no <!-- schema:%s --> marker in %s", name, commandDoc)

	var names []string

	for line := range strings.SplitSeq(after, "\n") {
		field, ok := isTableRow(strings.TrimSpace(line))
		if ok {
			names = append(names, field)

			continue
		}

		if names != nil {
			break // the table ended
		}
	}

	require.NotEmpty(t, names, "schema table for %s is empty", name)
	slices.Sort(names)

	return names
}

// TestCommandDocMatchesSchema fails when a request struct and its table in the
// command doc drift apart. If this fails, update the matching table in
// .claude/commands/job-search.md.
func TestCommandDocMatchesSchema(t *testing.T) {
	// Maps each `<!-- schema:NAME -->` marker to the struct it documents.
	schemas := map[string]any{
		"createJobRequest":      createJobRequest{},
		"UpdateJobParams":       db.UpdateJobParams{},
		"profileRequest":        profileRequest{},
		"createActivityRequest": createActivityRequest{},
		"createArtifactRequest": createArtifactRequest{},
		"parseRequest":          parseRequest{},
	}

	// Fields deliberately left out of a doc table. UpdateJobParams.ID is set
	// from the URL path by handleUpdateJob, so the doc tells the agent to put
	// the job id in the path instead of the body. The three profile fields are
	// server-owned: accepted so a GET/edit/PUT round trip works, but there is
	// nothing for the agent to decide about them.
	omits := map[string][]string{
		"UpdateJobParams": {"id"},
		"profileRequest":  {"id", "profile_hash", "updated_at"},
	}

	raw, err := os.ReadFile(filepath.Clean(commandDoc))
	require.NoError(t, err)

	doc := string(raw)

	for name, req := range schemas {
		t.Run(name, func(t *testing.T) {
			want := slices.DeleteFunc(jsonFields(req), func(field string) bool {
				return slices.Contains(omits[name], field)
			})

			assert.Equal(t, want, docFields(t, doc, name),
				"the %s table in %s is out of sync with the Go struct", name, commandDoc)
		})
	}
}

// TestParseResponseFieldsDocumented checks that every field /api/parse returns
// appears in the parse-to-create translation table, so a new parser field can't
// silently go unmapped when creating a job.
func TestParseResponseFieldsDocumented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(commandDoc))
	require.NoError(t, err)

	translationTable, _, _ := strings.Cut(string(raw), "## POST /api/jobs")

	for _, field := range jsonFields(parser.ParsedJob{}) {
		assert.Contains(t, translationTable, "| `"+field+"` |",
			"/api/parse returns %q but the translation table in %s doesn't map it", field, commandDoc)
	}
}
