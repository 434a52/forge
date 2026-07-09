package c5n

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Build runs the full pipeline rooted at dir (the folder holding c5n.yaml): load the
// manifest, schema and data, then emit every configured target for every table.
func Build(dir string) error {
	m, err := LoadManifest(dir)
	if err != nil {
		return err
	}

	schemaPaths, err := expandGlob(dir, m.Sources.Schema)
	if err != nil {
		return fmt.Errorf("schema glob %q: %w", m.Sources.Schema, err)
	}
	dataPaths, err := expandGlob(dir, m.Sources.Data)
	if err != nil {
		return fmt.Errorf("data glob %q: %w", m.Sources.Data, err)
	}

	schema, err := LoadSchema(dir, schemaPaths)
	if err != nil {
		return err
	}
	tables, err := LoadData(dir, dataPaths)
	if err != nil {
		return err
	}
	schemaSrc := strings.Join(schemaPaths, " + ")

	// Deterministic target order so output (and logs) don't depend on map iteration.
	targetNames := make([]string, 0, len(m.Targets))
	for name := range m.Targets {
		targetNames = append(targetNames, name)
	}
	sort.Strings(targetNames)

	written := 0
	for _, t := range tables {
		typ, ok := schema[t.ElemType]
		if !ok {
			return fmt.Errorf("%s: unknown type %s", t.Source, t.ElemType)
		}
		if t.Kind != "table" {
			return fmt.Errorf("%s: collection kind %q not yet supported", t.Source, t.Kind)
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
				return fmt.Errorf("unknown target %q in manifest", name)
			}
			if err != nil {
				return fmt.Errorf("emit %s/%s: %w", name, typ.Name, err)
			}
			if err := writeFile(outPath, code); err != nil {
				return err
			}
			rel, _ := filepath.Rel(dir, outPath)
			fmt.Printf("  %s\n", rel)
			written++
		}
	}
	fmt.Printf("c5n: wrote %d file(s)\n", written)
	return nil
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
