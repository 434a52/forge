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

	var files []GenFile
	for _, t := range tables {
		// Validate has already rejected an unknown type; the check keeps the lookup total.
		typ, ok := schema[t.ElemType]
		if !ok {
			return nil, nil, fmt.Errorf("%s: unknown type %s", t.Source, t.ElemType)
		}
		if t.Kind != "table" {
			return nil, nil, fmt.Errorf("%s: collection kind %q not yet supported", t.Source, t.Kind)
		}
		for _, name := range targetNames {
			target := m.Targets[name]
			var code, outPath string
			switch name {
			case "csharp":
				code, err = emitCSharp(t, typ, schema, target, schemaSrc)
				outPath = filepath.Join(dir, target.Out, typ.Name+".g.cs")
			case "ts":
				code, err = emitTS(t, typ, schema, schemaSrc)
				outPath = filepath.Join(dir, target.Out, strings.ToLower(typ.Name)+".data.ts")
			default:
				return nil, nil, fmt.Errorf("unknown target %q in manifest", name)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("emit %s/%s: %w", name, typ.Name, err)
			}
			rel, _ := filepath.Rel(dir, outPath)
			files = append(files, GenFile{Path: outPath, Rel: rel, Content: code, Target: name})
		}
	}
	return files, m, nil
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
