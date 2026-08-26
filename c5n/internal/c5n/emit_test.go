package c5n

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// mustTable parses data-file text the way LoadData does, so tests exercise the real path
// from authored YAML to Row rather than hand-built rows.
func mustTable(t *testing.T, src string) *Table {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	parsed, err := parseDataDoc(&doc)
	if err != nil {
		t.Fatalf("parseDataDoc: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want one collection, got %d", len(parsed))
	}
	return parsed[0]
}

func rateSchema(emit map[string]string) Schema {
	return Schema{"Rate": &Type{
		Name: "Rate", External: true, Key: "code", Emit: emit,
		Fields: []Field{{Name: "code", Type: "string"}, {Name: "value", Type: "decimal"}},
	}}
}

// A decimal must reach the target with the digits the data file authored. Decoding YAML
// through `any` yields float64, which cannot hold what a C# decimal can — the value would
// be altered before emission, and the generated code would compile clean while being wrong.
func TestDecimalPrecisionSurvives(t *testing.T) {
	tbl := mustTable(t, "type: table<Rate>\nitems:\n  - code: HIGH\n    value: 123456789.123456789\n")
	schema := rateSchema(nil)

	cs, err := emitCSharp(tbl, schema["Rate"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	if want := `new Rate("HIGH", 123456789.123456789m)`; !strings.Contains(cs, want) {
		t.Errorf("C#: want %s\ngot:\n%s", want, cs)
	}

	ts, err := emitTS(tbl, schema["Rate"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}
	if want := `new Rate("HIGH", 123456789.123456789)`; !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}
}

func TestScalarLiteral(t *testing.T) {
	cases := []struct {
		name, declType, text, target, want string
		wantErr                            bool
	}{
		{name: "string quoted", declType: "string", text: `a"b`, target: "csharp", want: `"a\"b"`},
		{name: "utf8 passes through", declType: "string", text: "£", target: "ts", want: `"£"`},
		{name: "int", declType: "int", text: "826", target: "csharp", want: "826"},
		{name: "int rejects text", declType: "int", text: "eight", target: "csharp", wantErr: true},
		{name: "decimal csharp suffix", declType: "decimal", text: "1.005", target: "csharp", want: "1.005m"},
		{name: "decimal ts bare", declType: "decimal", text: "1.005", target: "ts", want: "1.005"},
		{name: "decimal negative", declType: "decimal", text: "-0.5", target: "ts", want: "-0.5"},
		{name: "decimal rejects exponent", declType: "decimal", text: "1.2e+08", target: "csharp", wantErr: true},
		{name: "decimal rejects text", declType: "decimal", text: "lots", target: "csharp", wantErr: true},
		{name: "bool", declType: "bool", text: "true", target: "ts", want: "true"},
		{name: "bool rejects yes", declType: "bool", text: "yes", target: "ts", wantErr: true},
		{name: "unknown scalar", declType: "money", text: "1", target: "ts", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := scalarLiteral(c.declType, c.text, c.target)
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("want %q, got %q", c.want, got)
			}
		})
	}
}

// An emit: recipe chooses the shape of the construction call, per target — the seam that
// lets a value be constructed exactly in each language (a decimal literal in C#, a parsed
// string in TS) from one declaration.
func TestEmitRecipePerTarget(t *testing.T) {
	tbl := mustTable(t, "type: table<Rate>\nitems:\n  - code: VAT\n    value: 0.20\n")
	schema := rateSchema(map[string]string{
		"csharp": "Rate.Parse({code}, {value})",
		"ts":     "Rate.parse({code}, String({value}))",
	})

	cs, err := emitCSharp(tbl, schema["Rate"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	if want := `Rate.Parse("VAT", 0.20m)`; !strings.Contains(cs, want) {
		t.Errorf("C#: want %s\ngot:\n%s", want, cs)
	}

	ts, err := emitTS(tbl, schema["Rate"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}
	if want := `Rate.parse("VAT", String(0.20))`; !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}
}

// A type with a recipe for one target only keeps the ctor convention for the others.
func TestEmitRecipeFallsBackToConvention(t *testing.T) {
	tbl := mustTable(t, "type: table<Rate>\nitems:\n  - code: VAT\n    value: 0.20\n")
	schema := rateSchema(map[string]string{"csharp": "Rate.Parse({code})"})

	ts, err := emitTS(tbl, schema["Rate"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}
	if want := `new Rate("VAT", 0.20)`; !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}
}

func TestEmitRecipeUnknownField(t *testing.T) {
	tbl := mustTable(t, "type: table<Rate>\nitems:\n  - code: VAT\n    value: 0.20\n")
	schema := rateSchema(map[string]string{"csharp": "Rate.Parse({code}, {rate})"})

	_, err := emitCSharp(tbl, schema["Rate"], schema, Target{Namespace: "X"}, "s.yaml")
	if err == nil {
		t.Fatal("want an error naming the undeclared field")
	}
	if !strings.Contains(err.Error(), "rate") {
		t.Errorf("error should name the undeclared field, got: %v", err)
	}
}

// A row missing a declared field must fail by name, not emit a silently empty value.
func TestMissingFieldIsAnError(t *testing.T) {
	tbl := mustTable(t, "type: table<Rate>\nitems:\n  - code: VAT\n")
	schema := rateSchema(nil)

	_, err := emitCSharp(tbl, schema["Rate"], schema, Target{Namespace: "X"}, "s.yaml")
	if err == nil {
		t.Fatal("want an error for the missing field")
	}
	if !strings.Contains(err.Error(), "value") {
		t.Errorf("error should name the missing field, got: %v", err)
	}
}

func TestNullIsRejected(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("type: table<Rate>\nitems:\n  - code: VAT\n    value: ~\n"), &doc); err != nil {
		t.Fatal(err)
	}
	if _, err := parseDataDoc(&doc); err == nil {
		t.Fatal("want an error for a null field value")
	}
}

// References resolve to a sibling constant: qualified in C#, a bare imported name in TS.
func TestReferenceResolution(t *testing.T) {
	schema := Schema{
		"Currency": &Type{Name: "Currency", External: true, Key: "code",
			Fields: []Field{{Name: "code", Type: "string"}}},
		"Country": &Type{Name: "Country", External: true, Key: "alpha3",
			Fields: []Field{{Name: "alpha3", Type: "string"}, {Name: "ccy", Type: "Currency"}}},
	}
	tbl := mustTable(t, "type: table<Country>\nitems:\n  - alpha3: GBR\n    ccy: GBP\n")

	cs, err := emitCSharp(tbl, schema["Country"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	if want := `new Country("GBR", Currency.GBP)`; !strings.Contains(cs, want) {
		t.Errorf("C#: want %s\ngot:\n%s", want, cs)
	}

	ts, err := emitTS(tbl, schema["Country"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}
	if want := `new Country("GBR", GBP)`; !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}
	if want := `import { GBP } from "./currency.data.js";`; !strings.Contains(ts, want) {
		t.Errorf("TS: want import %s\ngot:\n%s", want, ts)
	}
}

// wrapperSchema is the nested-ctor shape: a table row whose field is typed as a
// constructible one-field type — Percentage over an exact Rational, as f8n declares it.
func wrapperSchema(emit map[string]string) Schema {
	return Schema{
		"Band": &Type{
			Name: "Band", External: true, Key: "code",
			Fields: []Field{{Name: "code", Type: "string"}, {Name: "rate", Type: "Percentage"}},
		},
		"Percentage": &Type{
			Name: "Percentage", External: true, Emit: emit,
			Fields: []Field{{Name: "value", Type: "string"}},
		},
	}
}

// The third value shape. A one-field type may be authored as a plain scalar — `rate: 17.5`,
// not `rate: { value: 17.5 }` — since the mapping would carry no information the schema does
// not already have. The recipe is what names the unit, in one reviewed place.
func TestNestedCtorUsesTheRecipe(t *testing.T) {
	tbl := mustTable(t, "type: table<Band>\nitems:\n  - { code: STANDARD, rate: 17.5 }\n")
	schema := wrapperSchema(map[string]string{
		"csharp": "Percentage.FromPercent({value})",
		"ts":     "Percentage.fromPercent({value})",
	})

	cs, err := emitCSharp(tbl, schema["Band"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	if want := `new Band("STANDARD", Percentage.FromPercent("17.5"))`; !strings.Contains(cs, want) {
		t.Errorf("C#: want %s\ngot:\n%s", want, cs)
	}

	ts, err := emitTS(tbl, schema["Band"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}
	if want := `new Band("STANDARD", Percentage.fromPercent("17.5"))`; !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}
	// TS has no partial, so a constructed hand-written type must be imported by name or the
	// generated module does not compile.
	if want := `import { Percentage } from "../percentage.js";`; !strings.Contains(ts, want) {
		t.Errorf("TS: missing import for the nested type\ngot:\n%s", ts)
	}
}

// Without a recipe the positional convention applies, exactly as for a top-level type.
func TestNestedCtorFallsBackToConvention(t *testing.T) {
	tbl := mustTable(t, "type: table<Band>\nitems:\n  - { code: STANDARD, rate: 17.5 }\n")
	schema := wrapperSchema(nil)

	cs, err := emitCSharp(tbl, schema["Band"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	if want := `new Band("STANDARD", new Percentage("17.5"))`; !strings.Contains(cs, want) {
		t.Errorf("want %s\ngot:\n%s", want, cs)
	}
}

// The scalar shorthand only makes sense for one field. With more, which field the value
// belongs to is a guess — so it is an error naming the type and what it was handed.
func TestMultiFieldNestedTypeIsRejected(t *testing.T) {
	schema := wrapperSchema(nil)
	schema["Percentage"].Fields = append(schema["Percentage"].Fields, Field{Name: "basis", Type: "string"})
	tbl := mustTable(t, "type: table<Band>\nitems:\n  - { code: STANDARD, rate: 17.5 }\n")

	_, err := emitCSharp(tbl, schema["Band"], schema, Target{Namespace: "X"}, "s.yaml")
	if err == nil {
		t.Fatal("want an error for a multi-field nested type, got nil")
	}
	for _, want := range []string{"Percentage", "2 fields", `"17.5"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q: %v", want, err)
		}
	}
}

// A one-field type whose field is itself cannot be built from any finite value; without a
// depth guard the emitter would recurse until the stack gave out.
func TestSelfNestingTypeIsRejected(t *testing.T) {
	schema := wrapperSchema(nil)
	schema["Percentage"].Fields = []Field{{Name: "value", Type: "Percentage"}}
	tbl := mustTable(t, "type: table<Band>\nitems:\n  - { code: STANDARD, rate: 17.5 }\n")

	_, err := emitCSharp(tbl, schema["Band"], schema, Target{Namespace: "X"}, "s.yaml")
	if err == nil {
		t.Fatal("want an error for a self-nesting type, got nil")
	}
	if !strings.Contains(err.Error(), "contains itself") {
		t.Errorf("want a self-nesting message, got: %v", err)
	}
}

// taxSchema is an enum plus a table type with a field declared as that enum — the two
// halves of 2.2: emitting a type body, and referencing one of its members from data.
func taxSchema() Schema {
	return Schema{
		"TaxType": &Type{Name: "TaxType", Kind: KindEnum, Members: []string{"VAT", "GST"}},
		"Levy": &Type{
			Name: "Levy", External: true, Key: "code",
			Fields: []Field{{Name: "code", Type: "string"}, {Name: "taxType", Type: "TaxType"}},
		},
	}
}

// Member names are emitted exactly as declared, in both targets and in every position.
// f8n serialises an enum as text, so the member name is the token that crosses the wire —
// a generator that applied casing would be rewriting a published contract, and it would
// turn VAT into Vat on the way.
func TestEnumMembersAreEmittedVerbatim(t *testing.T) {
	typ := taxSchema()["TaxType"]

	cs := emitEnumCSharp(typ, Target{Namespace: "X"}, "s.yaml")
	for _, want := range []string{"public enum TaxType", "    VAT,", "    GST,"} {
		if !strings.Contains(cs, want) {
			t.Errorf("C#: want %q\ngot:\n%s", want, cs)
		}
	}

	ts := emitEnumTS(typ, "s.yaml")
	for _, want := range []string{
		"export const TaxType = {",
		`  VAT: "VAT",`,
		`  GST: "GST",`,
		"} as const;",
		"export type TaxType = (typeof TaxType)[keyof typeof TaxType];",
	} {
		if !strings.Contains(ts, want) {
			t.Errorf("TS: want %q\ngot:\n%s", want, ts)
		}
	}

	// Not a TS `enum`: it is number-backed, so the runtime value would no longer be the
	// serialised token, and it is not erasable syntax.
	if strings.Contains(ts, "enum ") {
		t.Errorf("TS emitted an enum declaration:\n%s", ts)
	}
}

// The reference reads *identically* in both targets — the reason for the const-object
// spelling over a bare string union. One shared resolver produces one expression, so there
// is no per-target reference spelling that could drift.
func TestEnumReferenceIsIdenticalInBothTargets(t *testing.T) {
	tbl := mustTable(t, "type: table<Levy>\nitems:\n  - { code: STD, taxType: VAT }\n")
	schema := taxSchema()

	cs, err := emitCSharp(tbl, schema["Levy"], schema, Target{Namespace: "X"}, "s.yaml")
	if err != nil {
		t.Fatalf("emitCSharp: %v", err)
	}
	ts, err := emitTS(tbl, schema["Levy"], schema, "s.yaml")
	if err != nil {
		t.Fatalf("emitTS: %v", err)
	}

	const want = `new Levy("STD", TaxType.VAT)`
	if !strings.Contains(cs, want) {
		t.Errorf("C#: want %s\ngot:\n%s", want, cs)
	}
	if !strings.Contains(ts, want) {
		t.Errorf("TS: want %s\ngot:\n%s", want, ts)
	}

	// TS imports what it names, and for an enum that is the const object once — not one
	// import per member, which is what a table constant gets.
	if w := `import { TaxType } from "./taxtype.data.js";`; !strings.Contains(ts, w) {
		t.Errorf("TS: want %s\ngot:\n%s", w, ts)
	}
}

// seriesOnly is a series type declared with the recipes named by the caller, for pinning
// what happens when a recipe is absent or malformed.
func seriesOnly(emit map[string]string) Schema {
	return Schema{
		"Stamp": &Type{Name: "Stamp", External: true, Fields: []Field{{Name: "value", Type: "string"}}},
		"Reading": &Type{Name: "Reading", External: true,
			Fields: []Field{{Name: "code", Type: "string"}}},
		"Series": &Type{Name: "Series", External: true, Kind: KindSeries,
			Envelope: []Field{{Name: "at", Type: "Stamp"}}, Emit: emit},
	}
}

// A series has no positional-ctor convention to fall back on — a collection is built by a
// factory taking a list, and what that factory is called is per-language. Both ways of
// getting the recipe wrong fail with a message naming the type and the target.
func TestSeriesRecipeIsRequiredAndMustTakeTheEntries(t *testing.T) {
	tbl := mustTable(t, "type: Series<Reading>\nitems:\n  - { at: noon, code: A }\n")

	cases := []struct {
		name string
		emit map[string]string
		want string
	}{
		{"no recipe", map[string]string{"ts": "x({entries})"}, "no emit.csharp recipe"},
		{"recipe ignores the entries", map[string]string{"csharp": "Series.Empty()"}, "{entries}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			schema := seriesOnly(c.emit)
			_, err := emitSeriesCSharp(tbl, schema["Series"], schema["Reading"], schema, Target{Namespace: "X"}, "s.yaml")
			if err == nil {
				t.Fatal("want an error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("want an error mentioning %q, got: %v", c.want, err)
			}
		})
	}
}
