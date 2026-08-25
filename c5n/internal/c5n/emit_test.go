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
	tbl, err := parseDataDoc(&doc)
	if err != nil {
		t.Fatalf("parseDataDoc: %v", err)
	}
	return tbl
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
	if want := `import { GBP } from "./currency.data";`; !strings.Contains(ts, want) {
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
	if want := `import { Percentage } from "../percentage";`; !strings.Contains(ts, want) {
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
