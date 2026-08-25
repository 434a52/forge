package c5n

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Validation runs once over the loaded schema and data, before any writer sees them.
//
// Ahead of the writers, and not inside them, for two reasons: a data problem is the same
// problem in every target, so validating per-target would report it once per language; and
// the author's fix is in the data file, not in the code c5n would have produced from it.

// Validate checks the loaded data against the schema, reporting every problem it finds
// rather than stopping at the first. A renamed or misspelled field usually breaks several
// rows at once, and fixing them one error-per-run is a poor trade for whoever is editing
// the data.
func Validate(schema Schema, tables []*Table) error {
	var problems []string
	for _, t := range tables {
		typ, ok := schema[t.ElemType]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: unknown type %s", t.Source, t.ElemType))
			continue
		}
		problems = append(problems, undeclaredFields(t, typ)...)
	}

	if len(problems) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("data does not match the schema:\n")
	for _, p := range problems {
		fmt.Fprintf(&b, "  - %s\n", p)
	}
	return errors.New(strings.TrimRight(b.String(), "\n"))
}

// undeclaredFields reports any field in a row that the schema does not declare on that
// row's type. Two different mistakes arrive here, and they used to fail differently:
//
//   - A field the schema has no place for — one the author added expecting it to appear in
//     the output, or one left behind by a schema rename. This was dropped in complete
//     silence: the writers walk the *declared* fields and never look at the rest, so the
//     output regenerated, compiled, and simply did not contain what the author wrote.
//
//   - A misspelled key, which also leaves a declared field absent. The writers did catch
//     this, but by reporting the absence — "field capitalTz: missing from row" — which
//     names the field spelled correctly and says nothing about the one that is wrong.
//     Naming the undeclared key instead points at the actual mistake.
//
// The first is the silent one and the reason this check exists; the second is why it runs
// *before* the writers rather than alongside them.
//
// When `EffectiveDated<T>` lands, a series row will also carry the envelope key the
// *collection* declares (`from`), which is not a field of `T`. That exemption belongs
// here, driven by the declared collection kind — not by a general escape hatch, which
// would give back the silence this check removes.
func undeclaredFields(t *Table, typ *Type) []string {
	declared := make(map[string]bool, len(typ.Fields))
	for _, f := range typ.Fields {
		declared[f.Name] = true
	}

	var problems []string
	for i, row := range t.Rows {
		var undeclared []string
		for name := range row {
			if !declared[name] {
				undeclared = append(undeclared, name)
			}
		}
		// A Row is a map, so sort before reporting: Go randomises map iteration, and error
		// output that reorders between runs is output nobody can diff.
		sort.Strings(undeclared)

		for _, name := range undeclared {
			problems = append(problems, fmt.Sprintf(
				"%s, %s: field %q is not declared on %s (declared: %s)",
				t.Source, rowLabel(row, typ, i), name, typ.Name, declaredList(typ)))
		}
	}
	return problems
}

// rowLabel identifies a row in an error message: its key value where the type declares a
// key and the row carries one — the name the author recognises — and otherwise its
// position in the file.
func rowLabel(row Row, typ *Type, index int) string {
	if typ.Key != "" {
		if key, ok := row[typ.Key]; ok {
			return fmt.Sprintf("row %d (%s)", index+1, key)
		}
	}
	return fmt.Sprintf("row %d", index+1)
}

// declaredList names the type's fields in declaration order — the list the author needs in
// front of them to spot which one they meant.
func declaredList(typ *Type) string {
	names := make([]string, len(typ.Fields))
	for i, f := range typ.Fields {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}
