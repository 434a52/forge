package c5n

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenFile is one file c5n would emit: its content plus where it goes.
type GenFile struct {
	Path    string // absolute output path
	Rel     string // path relative to dir (for messages)
	Content string
	Target  string // "csharp" / "ts" — the emitting target
	Source  string // the data file(s) it was emitted from
}

// targetExt maps a target to the filename suffix it owns, so the drift-guard can spot
// orphaned generated files (matching output with no source) in an out dir.
var targetExt = map[string]string{"csharp": ".g.cs", "ts": ".data.ts"}

// generate loads the manifest, schema and data and returns every file c5n would emit,
// without touching disk. Build writes the result; Check compares against it.
func generate(dir string) ([]GenFile, *Manifest, error) {
	m, err := LoadManifest(dir)
	if err != nil {
		return nil, nil, err
	}

	schemaPaths, err := expandGlob(dir, m.Sources.Schema)
	if err != nil {
		return nil, nil, fmt.Errorf("schema glob %q: %w", m.Sources.Schema, err)
	}
	dataPaths, err := expandGlob(dir, m.Sources.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("data glob %q: %w", m.Sources.Data, err)
	}

	schema, err := LoadSchema(dir, schemaPaths)
	if err != nil {
		return nil, nil, err
	}
	tables, err := LoadData(dir, dataPaths)
	if err != nil {
		return nil, nil, err
	}
	if err := Validate(schema, tables); err != nil {
		return nil, nil, err
	}
	schemaSrc := strings.Join(schemaPaths, " + ")

	// Deterministic target order so output (and logs) don't depend on map iteration.
	targetNames := make([]string, 0, len(m.Targets))
	for name := range m.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	for _, t := range tables {
		if t.Kind != "table" {
			return nil, nil, fmt.Errorf("%s: collection kind %q not yet supported", t.Source, t.Kind)
		}
	}

	var files []GenFile
	emitters := map[string]string{} // output path -> the sources that produced it

	// addFile records one emitted file. Two units resolving to one path means one silently
	// overwrites the other, so it is an error rather than a race between writes — the state
	// it used to produce was a `c5n check` failure immediately after a clean build,
	// advising a rebuild that could not fix it.
	addFile := func(outPath, code, targetName, source string) error {
		rel, _ := filepath.Rel(dir, outPath)
		if prev, clash := emitters[outPath]; clash {
			return fmt.Errorf("%s would be written twice — from %s and from %s", rel, prev, source)
		}
		emitters[outPath] = source
		files = append(files, GenFile{Path: outPath, Rel: rel, Content: code, Target: targetName, Source: source})
		return nil
	}

	// Enums are emitted from the schema, not from data. Their members are declared, so an
	// enum exists — and is part of the published API — whether or not any data row has
	// referenced it yet. Every other unit so far is derived from a data file, which is why
	// this is its own pass rather than another case in the loop below.
	for _, typ := range sortedEnums(schema) {
		for _, name := range targetNames {
			target := m.Targets[name]
			outPath, err := outputPath(dir, target, name, typ.Name)
			if err != nil {
				return nil, nil, err
			}
			var code string
			switch name {
			case "csharp":
				code = emitEnumCSharp(typ, target, schemaSrc)
			case "ts":
				code = emitEnumTS(typ, schemaSrc)
			default:
				return nil, nil, fmt.Errorf("unknown target %q in manifest", name)
			}
			if err := addFile(outPath, code, name, schemaSrc); err != nil {
				return nil, nil, err
			}
		}
	}

	for _, t := range mergeByEmittedType(tables) {
		// Validate has already rejected an unknown type; the check keeps the lookup total.
		typ, ok := schema[t.ElemType]
		if !ok {
			return nil, nil, fmt.Errorf("%s: unknown type %s", t.Source, t.ElemType)
		}
		for _, name := range targetNames {
			target := m.Targets[name]
			outPath, err := outputPath(dir, target, name, typ.Name)
			if err != nil {
				return nil, nil, err
			}
			var code string
			switch name {
			case "csharp":
				code, err = emitCSharp(t, typ, schema, target, schemaSrc)
			case "ts":
				code, err = emitTS(t, typ, schema, schemaSrc)
			default:
				return nil, nil, fmt.Errorf("unknown target %q in manifest", name)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("emit %s/%s: %w", name, typ.Name, err)
			}
			if err := addFile(outPath, code, name, t.Source); err != nil {
				return nil, nil, err
			}
		}
	}
	return files, m, nil
}

// outputPath is where the unit declaring typeName goes in one target.
//
// Output is named for what it *declares* (DESIGN.md → output paths), so this depends on the
// type and the target and on nothing else — not on the data file a table happened to be
// authored in, and not on whether the unit came from data at all. One home for the rule, so
// the two emission passes above cannot drift apart on it.
func outputPath(dir string, target Target, targetName, typeName string) (string, error) {
	switch targetName {
	case "csharp":
		return filepath.Join(dir, target.Out, typeName+".g.cs"), nil
	case "ts":
		return filepath.Join(dir, target.Out, strings.ToLower(typeName)+".data.ts"), nil
	}
	return "", fmt.Errorf("unknown target %q in manifest", targetName)
}

// sortedEnums returns the schema's enums in name order. Schema is a map and Go randomises
// map iteration, so without this the *set* of emitted files would be stable but their order
// would not — and an unstable build order is one the drift-guard cannot report cleanly.
func sortedEnums(schema Schema) []*Type {
	var names []string
	for name, typ := range schema {
		if typ.IsEnum() {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	enums := make([]*Type, 0, len(names))
	for _, name := range names {
		enums = append(enums, schema[name])
	}
	return enums
}

// mergeByEmittedType groups the loaded tables by what they are *emitted as*, because that
// is what names the output file — not the file they happened to be authored in.
//
// A `table<T>` emits one unit per type: however many data files declare Currency rows, they
// become one Currency.g.cs. Splitting reference data across files is an authoring
// convenience — per region, per source, per reviewer — and the output should not inherit
// that shape. Before this, two files declaring one type resolved to one path and the second
// silently overwrote the first; `c5n check` then failed immediately after a clean build,
// advising a rebuild that could not fix it.
//
// Rows keep source order: tables arrive sorted by path and rows in file order, so the merged
// output is deterministic and each data file's rows stay contiguous in the diff.
//
// The unit is per *kind*, so this grows as kinds do — `EffectiveDated` will emit one unit
// per named series rather than one per type, and will group accordingly.
func mergeByEmittedType(tables []*Table) []*Table {
	var order []string // element types in first-appearance order
	merged := map[string]*Table{}
	sources := map[string][]string{}

	for _, t := range tables {
		unit, seen := merged[t.ElemType]
		if !seen {
			clone := *t
			clone.Rows = append([]Row(nil), t.Rows...)
			merged[t.ElemType] = &clone
			sources[t.ElemType] = []string{t.Source}
			order = append(order, t.ElemType)
			continue
		}
		unit.Rows = append(unit.Rows, t.Rows...)
		sources[t.ElemType] = append(sources[t.ElemType], t.Source)
	}

	units := make([]*Table, 0, len(order))
	for _, name := range order {
		unit := merged[name]
		// Provenance is every file that fed the unit, so the generated header still says
		// where to go and looking at one source no longer tells half the story.
		unit.Source = strings.Join(sources[name], " + ")
		units = append(units, unit)
	}
	return units
}

// Build runs the full pipeline rooted at dir and writes the generated files.
func Build(dir string) error {
	files, _, err := generate(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := writeFile(f.Path, f.Content); err != nil {
			return err
		}
		fmt.Printf("  %s\n", f.Rel)
	}
	fmt.Printf("c5n: wrote %d file(s)\n", len(files))
	return nil
}

// Check is the drift-guard: it regenerates in memory and compares against the committed
// files, reporting anything out of date, missing, or orphaned. It writes nothing. A clean
// result means the committed output is exactly what the current sources produce.
func Check(dir string) error {
	files, m, err := generate(dir)
	if err != nil {
		return err
	}

	var drift []string
	expected := make(map[string]bool, len(files))
	for _, f := range files {
		expected[f.Path] = true
		existing, err := os.ReadFile(f.Path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			drift = append(drift, f.Rel+" (missing — never generated)")
		case err != nil:
			return err
		case string(existing) != f.Content:
			drift = append(drift, f.Rel+" (out of date)")
		}
	}
	orphans, err := findOrphans(dir, m, expected)
	if err != nil {
		return err
	}
	drift = append(drift, orphans...)
	sort.Strings(drift)

	if len(drift) > 0 {
		var b strings.Builder
		b.WriteString("generated files are out of sync with their sources:\n")
		for _, d := range drift {
			fmt.Fprintf(&b, "  - %s\n", d)
		}
		b.WriteString("run `c5n build` and commit the result")
		return errors.New(b.String())
	}
	fmt.Printf("c5n: %d generated file(s) in sync\n", len(files))
	return nil
}

// findOrphans reports generated-looking files in the targets' out dirs that c5n would not
// produce — e.g. left behind when a source type is renamed or removed.
func findOrphans(dir string, m *Manifest, expected map[string]bool) ([]string, error) {
	var orphans []string
	for name, target := range m.Targets {
		ext, ok := targetExt[name]
		if !ok {
			continue
		}
		outDir := filepath.Join(dir, target.Out)
		entries, err := os.ReadDir(outDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
				continue
			}
			p := filepath.Join(outDir, e.Name())
			if !expected[p] {
				rel, _ := filepath.Rel(dir, p)
				orphans = append(orphans, rel+" (orphan — no source)")
			}
		}
	}
	return orphans, nil
}

// writeFile creates parent dirs and writes content.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// expandGlob resolves a source pattern to file paths relative to dir. It supports the two
// shapes the manifest uses: a plain glob ("schema/*.yaml") and a recursive one
// ("data/**/*.yaml"), which stdlib filepath.Glob can't do — so "**" triggers a walk.
func expandGlob(dir, pattern string) ([]string, error) {
	var out []string
	if strings.Contains(pattern, "**") {
		base := strings.TrimSuffix(pattern[:strings.Index(pattern, "**")], "/")
		namePat := filepath.Base(pattern)
		err := filepath.WalkDir(filepath.Join(dir, base), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if ok, _ := filepath.Match(namePat, d.Name()); ok {
				rel, _ := filepath.Rel(dir, p)
				out = append(out, rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		for _, p := range matches {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, rel)
		}
	}
	sort.Strings(out) // stable order → stable provenance + deterministic output
	return out, nil
}
