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
//
// Three checks, in the order their results are useful: fields the schema does not declare,
// identities declared more than once, and references that name a row nothing declares. The
// middle one builds the index the last one reads, so a duplicate is reported where it
// happens rather than as a confusing knock-on further down.
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

	index, duplicates := buildKeyIndex(schema, tables)
	problems = append(problems, duplicates...)
	problems = append(problems, unresolvedReferences(schema, tables, index)...)

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

// rowRef is where an identity was declared: the file, and the row's position in it. Enough
// to send someone straight to the line that clashes.
type rowRef struct {
	Source string
	Row    int // 1-based, matching how rowLabel counts
}

// keyIndex maps a type name to every identity declared for it: type -> key value -> where.
// It is what makes a reference checkable — without it, c5n emits `Currency.GBP` on trust
// and the target compiler is the first thing that knows whether GBP exists.
type keyIndex map[string]map[string]rowRef

// buildKeyIndex collects every declared identity, reporting any declared twice as it goes.
//
// Uniqueness is checked per *type*, not per file: two files both declaring Currency GBP is
// the same collision as one file doing it twice, and it is the one the emitted output hides
// — the second constant silently replaces the first, or the two land in one file that will
// not compile. The identity section of DESIGN.md requires this uniqueness; the map
// authoring form used to give it structurally, and the list form has to check for it.
func buildKeyIndex(schema Schema, tables []*Table) (keyIndex, []string) {
	index := keyIndex{}
	var problems []string

	for _, t := range tables {
		typ, ok := schema[t.ElemType]
		if !ok || typ.Key == "" {
			continue // an unkeyed type has no identities; an unknown one is reported elsewhere
		}
		if _, ok := index[typ.Name]; !ok {
			index[typ.Name] = map[string]rowRef{}
		}

		for i, row := range t.Rows {
			key, ok := row[typ.Key]
			if !ok {
				problems = append(problems, fmt.Sprintf(
					"%s, %s: no %s field, so the row has no identity",
					t.Source, rowLabel(row, typ, i), typ.Key))
				continue
			}
			if first, clash := index[typ.Name][key]; clash {
				problems = append(problems, fmt.Sprintf(
					"%s, %s: %s %q is already declared at %s row %d",
					t.Source, rowLabel(row, typ, i), typ.Name, key, first.Source, first.Row))
				continue
			}
			index[typ.Name][key] = rowRef{Source: t.Source, Row: i + 1}
		}
	}
	return index, problems
}

// unresolvedReferences reports a reference field whose value names no declared row.
//
// c5n holds every table in memory, so it can answer this itself. Left unchecked, a stale or
// mistyped reference is emitted verbatim — `Currency.XXX` — and the first thing to notice is
// the target's compiler, which means it is caught only if someone compiles, reported against
// generated code rather than the data file that is wrong, and reported once per language.
func unresolvedReferences(schema Schema, tables []*Table, index keyIndex) []string {
	var problems []string
	for _, t := range tables {
		typ, ok := schema[t.ElemType]
		if !ok {
			continue
		}
		for i, row := range t.Rows {
			// Declaration order, so a row with several bad references reads top to bottom.
			for _, f := range typ.Fields {
				if !isReference(f.Type, schema) {
					continue
				}
				value, ok := row[f.Name]
				if !ok {
					continue // a missing field is the writers' error, at the point they need it
				}
				if _, known := index[f.Type][value]; known {
					continue
				}
				problems = append(problems, fmt.Sprintf(
					"%s, %s: field %q references %s %q, which no row declares (%s)",
					t.Source, rowLabel(row, typ, i), f.Name, f.Type, value, declaringSources(index, f.Type)))
			}
		}
	}
	return problems
}

// declaringSources says where the identities of a type come from, so the reader knows which
// file to open. Naming the files beats listing the keys: a table can hold hundreds of rows,
// and a wall of them buries the message that matters.
func declaringSources(index keyIndex, typeName string) string {
	if len(index[typeName]) == 0 {
		return "no data file declares " + typeName
	}
	seen := map[string]bool{}
	var sources []string
	for _, ref := range index[typeName] {
		if !seen[ref.Source] {
			seen[ref.Source] = true
			sources = append(sources, ref.Source)
		}
	}
	sort.Strings(sources) // the index is a map; the message must not reorder between runs
	return typeName + " rows come from " + strings.Join(sources, ", ")
}
