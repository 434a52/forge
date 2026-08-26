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
	return fixtureWithSchema(t, currencySchemaSrc, dataFiles)
}

const currencySchemaSrc = "Currency:\n" +
	"  external: true\n" +
	"  key: code\n" +
	"  fields: { code: string, numeric: int }\n"

// fixtureWithSchema is the same project, with the schema written by the caller — for the
// cases where what is under test is a declaration rather than a data row.
func fixtureWithSchema(t *testing.T, schemaSrc string, dataFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"c5n.yaml": "targets:\n" +
			"  csharp: { out: gen/cs/, namespace: X }\n" +
			"  ts:     { out: gen/ts/ }\n" +
			"sources:\n" +
			"  schema: schema/*.yaml\n" +
			"  data:   data/**/*.yaml\n",
		"schema/types.yaml": schemaSrc,
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

// An enum is emitted from the schema alone. Its members are declared, so it is part of the
// published API whether or not a data row has referenced it yet — which is the structural
// point of 2.2: every other unit so far is derived from a data file, and this one is not.
//
// It is also what the alternative design would have made impossible. Had members been
// collected from data, an enum nothing referenced could not be emitted at all, so the
// public API would have depended on data coverage.
func TestEnumEmitsWithoutAnyData(t *testing.T) {
	dir := fixtureWithSchema(t,
		currencySchemaSrc+"TaxType:\n  kind: enum\n  members: [VAT, GST]\n",
		map[string]string{
			"currencies.yaml": "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n",
		})

	files, _, err := generate(dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	byRel := map[string]GenFile{}
	for _, f := range files {
		byRel[f.Rel] = f
	}
	// Named for what it declares, like every other unit.
	for _, rel := range []string{
		filepath.Join("gen", "cs", "TaxType.g.cs"),
		filepath.Join("gen", "ts", "taxtype.data.ts"),
	} {
		if _, ok := byRel[rel]; !ok {
			t.Fatalf("want %s, got %v", rel, paths(files))
		}
	}

	// The header names the schema it came from and no data file, because there is none.
	cs := byRel[filepath.Join("gen", "cs", "TaxType.g.cs")].Content
	if !strings.Contains(cs, "schema/types.yaml") || strings.Contains(cs, "data/") {
		t.Errorf("header should name the schema and no data file:\n%s", cs)
	}
}

const levySchemaSrc = "Levy:\n" +
	"  external: true\n" +
	"  key: code\n" +
	"  fields: { code: string, jurisdiction: string, numeric: int }\n"

// The whole claim of `common:`-hoisting is that it changes the data file and nothing else.
// So the test is not "does it merge" but "is the output the same as writing every field
// out" — byte-for-byte, both targets. Anything less and hoisting would be a decision an
// author has to weigh rather than a free convenience.
func TestCommonHoistingEmitsIdenticalOutput(t *testing.T) {
	hoisted := fixtureWithSchema(t, levySchemaSrc, map[string]string{
		"levies.yaml": "type: table<Levy>\ncommon: { jurisdiction: GB }\nitems:\n" +
			"  - { code: STD, numeric: 20 }\n  - { code: RED, numeric: 5 }\n",
	})
	written := fixtureWithSchema(t, levySchemaSrc, map[string]string{
		"levies.yaml": "type: table<Levy>\nitems:\n" +
			"  - { code: STD, jurisdiction: GB, numeric: 20 }\n" +
			"  - { code: RED, jurisdiction: GB, numeric: 5 }\n",
	})

	got, _, err := generate(hoisted)
	if err != nil {
		t.Fatalf("generate (hoisted): %v", err)
	}
	want, _, err := generate(written)
	if err != nil {
		t.Fatalf("generate (written out): %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("different file counts: %v vs %v", paths(got), paths(want))
	}
	byRel := map[string]string{}
	for _, f := range want {
		byRel[f.Rel] = f.Content
	}
	for _, f := range got {
		if byRel[f.Rel] != f.Content {
			t.Errorf("%s differs when hoisted:\n--- hoisted ---\n%s\n--- written out ---\n%s",
				f.Rel, f.Content, byRel[f.Rel])
		}
	}
}

// Merging has to happen after validation, not in the reader. A misspelled field in
// `common:` is copied into every row, so merging first would report one mistake once per
// row and name `common:` in none of them — the 1.2 failure again, one layer up.
func TestCommonMistakeIsReportedOnceAgainstCommon(t *testing.T) {
	dir := fixtureWithSchema(t, levySchemaSrc, map[string]string{
		"levies.yaml": "type: table<Levy>\ncommon: { jurisdicton: GB }\nitems:\n" +
			"  - { code: STD, jurisdiction: GB, numeric: 20 }\n" +
			"  - { code: RED, jurisdiction: GB, numeric: 5 }\n",
	})

	_, _, err := generate(dir)
	if err == nil {
		t.Fatal("want an error for the misspelled common field, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "common: field \"jurisdicton\"") {
		t.Errorf("the mistake must be named against common:\n%s", msg)
	}
	if n := strings.Count(msg, "jurisdicton"); n != 1 {
		t.Errorf("want the mistake reported once, got %d times:\n%s", n, msg)
	}
}
