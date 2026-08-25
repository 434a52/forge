package c5n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture writes a small but complete project — manifest, schema, data — and returns its
// root. These tests go through generate rather than the pieces, because what they pin is
// behaviour that depends on the order the pieces run in, which a unit test cannot see.
func fixture(t *testing.T, dataFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"c5n.yaml": "targets:\n" +
			"  csharp: { out: gen/cs/, namespace: X }\n" +
			"  ts:     { out: gen/ts/ }\n" +
			"sources:\n" +
			"  schema: schema/*.yaml\n" +
			"  data:   data/**/*.yaml\n",
		"schema/types.yaml": "Currency:\n" +
			"  external: true\n" +
			"  key: code\n" +
			"  fields: { code: string, numeric: int }\n",
	}
	for name, content := range dataFiles {
		files["data/"+name] = content
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Validation has to run before the writers, and the ordering is load-bearing rather than
// tidy: both halves see this row, and they say different things about it.
//
// A misspelled key leaves the declared field absent, so the writers report the absence —
// "field numeric: missing from row" — naming the field that is spelled correctly and
// staying silent about the one that is wrong. Validation names the misspelling itself.
// Reorder the two and the diagnosis silently degrades without any test failing, which is
// exactly what this one is here to stop.
func TestValidationRunsBeforeTheWriters(t *testing.T) {
	dir := fixture(t, map[string]string{
		"currencies.yaml": "type: table<Currency>\nitems:\n  - { code: GBP, numric: 826 }\n",
	})

	_, _, err := generate(dir)
	if err == nil {
		t.Fatal("want an error for the misspelled field, got nil")
	}
	if !strings.Contains(err.Error(), `"numric"`) {
		t.Errorf("want the misspelling named, got:\n%s", err)
	}
	if strings.Contains(err.Error(), "missing from row") {
		t.Errorf("the writers reported first — validation must run ahead of them:\n%s", err)
	}
}

// The fixture is only trustworthy as a negative test if it generates cleanly when the data
// is right, so this pins that too: one file per target, from one table.
func TestGenerateEmitsEveryConfiguredTarget(t *testing.T) {
	dir := fixture(t, map[string]string{
		"currencies.yaml": "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n",
	})

	files, _, err := generate(dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("want one file per target, got %d", len(files))
	}

	byTarget := map[string]GenFile{}
	for _, f := range files {
		byTarget[f.Target] = f
	}
	if !strings.Contains(byTarget["csharp"].Content, `new Currency("GBP", 826)`) {
		t.Errorf("C# output:\n%s", byTarget["csharp"].Content)
	}
	if !strings.Contains(byTarget["ts"].Content, `new Currency("GBP", 826)`) {
		t.Errorf("TS output:\n%s", byTarget["ts"].Content)
	}
}

// Splitting reference data across files is an authoring convenience — per region, per
// source — and the output must not inherit that shape. Both files declare Currency, so both
// feed one Currency unit per target. Before this, the second file's rows silently replaced
// the first's, and `c5n check` then failed straight after a clean build.
func TestSameTypeAcrossFilesEmitsOneUnit(t *testing.T) {
	dir := fixture(t, map[string]string{
		"a-currencies.yaml": "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n",
		"b-currencies.yaml": "type: table<Currency>\nitems:\n  - { code: EUR, numeric: 978 }\n",
	})

	files, _, err := generate(dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("want one file per target (2), got %d: %v", len(files), paths(files))
	}

	for _, f := range files {
		if !strings.Contains(f.Content, "GBP") || !strings.Contains(f.Content, "EUR") {
			t.Errorf("%s lost rows from one of its sources:\n%s", f.Rel, f.Content)
		}
		// Sources are globbed in sorted order, so rows stay in a deterministic, reviewable
		// order and each file's rows remain contiguous.
		if strings.Index(f.Content, "GBP") > strings.Index(f.Content, "EUR") {
			t.Errorf("%s: rows are not in source order:\n%s", f.Rel, f.Content)
		}
		// The header has to name every file that fed the unit, or it sends a reader to the
		// wrong half of the input.
		for _, want := range []string{"a-currencies.yaml", "b-currencies.yaml"} {
			if !strings.Contains(f.Content, want) {
				t.Errorf("%s: header omits source %s:\n%s", f.Rel, want, f.Content)
			}
		}
	}
}

func paths(files []GenFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Rel
	}
	return out
}
