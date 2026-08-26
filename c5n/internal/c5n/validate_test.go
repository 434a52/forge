package c5n

import (
	"strings"
	"testing"
)

// countrySchema mirrors the shape f8n declares: a keyed table, so error messages can name
// a row by the identity its author recognises rather than by position.
func countrySchema() Schema {
	return Schema{"Country": &Type{
		Name: "Country", External: true, Key: "alpha3",
		Fields: []Field{
			{Name: "alpha2", Type: "string"},
			{Name: "alpha3", Type: "string"},
			{Name: "capitalTz", Type: "string"},
		},
	}}
}

func validateSrc(t *testing.T, src string) error {
	t.Helper()
	tbl := mustTable(t, src)
	tbl.Source = "data/countries.yaml"
	return Validate(countrySchema(), []*Table{tbl})
}

// The failure this check exists for: a misspelled field is read by nothing, so without
// validation the row emits with that value missing — and the generated code compiles.
func TestUndeclaredFieldIsRejected(t *testing.T) {
	err := validateSrc(t, "type: table<Country>\nitems:\n  - { alpha2: GB, alpha3: GBR, captialTz: Europe/London }\n")
	if err == nil {
		t.Fatal("want an error for an undeclared field, got nil")
	}

	// The message has to carry everything needed to fix it without opening the schema.
	for _, want := range []string{
		"data/countries.yaml",       // which file
		"row 1 (GBR)",               // which row, by the identity in it
		`"captialTz"`,               // the offending field, quoted so the typo is visible
		"Country",                   // the type it is not declared on
		"alpha2, alpha3, capitalTz", // what is declared, in declaration order
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

// Every problem is reported in one run: a renamed field breaks many rows at once, and
// fixing them one-per-run wastes the author's time.
func TestAllProblemsReportedTogether(t *testing.T) {
	err := validateSrc(t, `type: table<Country>
items:
  - { alpha2: GB, alpha3: GBR, zzz: 1, aaa: 2 }
  - { alpha2: FR, alpha3: FRA, capitalTz: Europe/Paris, mmm: 3 }
`)
	if err == nil {
		t.Fatal("want errors, got nil")
	}
	msg := err.Error()
	for _, want := range []string{`"aaa"`, `"zzz"`, `"mmm"`, "row 1 (GBR)", "row 2 (FRA)"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error is missing %q:\n%s", want, msg)
		}
	}

	// Rows are maps, so the fields of one row must be sorted before reporting — otherwise
	// the message reorders between runs and stops being diffable.
	if strings.Index(msg, `"aaa"`) > strings.Index(msg, `"zzz"`) {
		t.Errorf("undeclared fields are not reported in sorted order:\n%s", msg)
	}
}

func TestDeclaredFieldsValidate(t *testing.T) {
	err := validateSrc(t, "type: table<Country>\nitems:\n  - { alpha2: GB, alpha3: GBR, capitalTz: Europe/London }\n")
	if err != nil {
		t.Errorf("valid data should validate, got: %v", err)
	}
}

// A row is not required to carry every declared field here — a missing field is the
// writers' error to report, at the point they need it. Validation covers only what the
// schema cannot account for at all.
func TestMissingFieldIsNotAValidationError(t *testing.T) {
	err := validateSrc(t, "type: table<Country>\nitems:\n  - { alpha2: GB, alpha3: GBR }\n")
	if err != nil {
		t.Errorf("a missing field is not this check's job, got: %v", err)
	}
}

func TestUnknownTypeIsRejected(t *testing.T) {
	tbl := mustTable(t, "type: table<Nope>\nitems:\n  - { a: 1 }\n")
	tbl.Source = "data/nope.yaml"
	err := Validate(countrySchema(), []*Table{tbl})
	if err == nil || !strings.Contains(err.Error(), "unknown type Nope") {
		t.Errorf("want an unknown-type error, got: %v", err)
	}
}

// Without a key in the row there is no identity to name, so the label falls back to the
// position — still enough to find the row in the file.
func TestRowLabelFallsBackToPosition(t *testing.T) {
	err := validateSrc(t, "type: table<Country>\nitems:\n  - { alpha2: GB, oops: 1 }\n")
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !strings.Contains(err.Error(), "row 1:") {
		t.Errorf("want a positional row label, got:\n%s", err)
	}
}

// refSchema has a reference field — Country.defaultCurrency points at a Currency row by its
// key — which is the shape the reference and uniqueness checks are about.
func refSchema() Schema {
	return Schema{
		"Currency": &Type{
			Name: "Currency", External: true, Key: "code",
			Fields: []Field{{Name: "code", Type: "string"}, {Name: "numeric", Type: "int"}},
		},
		"Country": &Type{
			Name: "Country", External: true, Key: "alpha3",
			Fields: []Field{
				{Name: "alpha3", Type: "string"},
				{Name: "defaultCurrency", Type: "Currency"},
			},
		},
	}
}

func tableFrom(t *testing.T, source, src string) *Table {
	t.Helper()
	tbl := mustTable(t, src)
	tbl.Source = source
	return tbl
}

// ---- 1.3 references ----

// Left unchecked this reaches the target as `Currency.XXX`: caught only if someone compiles,
// reported against generated code rather than the data file, and reported once per language.
func TestUnresolvedReferenceIsRejected(t *testing.T) {
	currencies := tableFrom(t, "data/currencies.yaml", "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n")
	countries := tableFrom(t, "data/countries.yaml", "type: table<Country>\nitems:\n  - { alpha3: GBR, defaultCurrency: XXX }\n")

	err := Validate(refSchema(), []*Table{currencies, countries})
	if err == nil {
		t.Fatal("want an error for a reference nothing declares, got nil")
	}
	for _, want := range []string{
		"data/countries.yaml", // the file to fix
		"row 1 (GBR)",         // the row
		`"defaultCurrency"`,   // the field
		`Currency "XXX"`,      // what it names
		"Currency rows come from data/currencies.yaml", // where the real ones are
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

func TestResolvedReferencePasses(t *testing.T) {
	currencies := tableFrom(t, "data/currencies.yaml", "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n")
	countries := tableFrom(t, "data/countries.yaml", "type: table<Country>\nitems:\n  - { alpha3: GBR, defaultCurrency: GBP }\n")

	if err := Validate(refSchema(), []*Table{currencies, countries}); err != nil {
		t.Errorf("a reference to a declared row should validate, got: %v", err)
	}
}

// Referencing a type with no data at all is a different mistake from a typo, and saying so
// saves the reader hunting for a file that was never there.
func TestReferenceWithNoDeclaringDataSaysSo(t *testing.T) {
	countries := tableFrom(t, "data/countries.yaml", "type: table<Country>\nitems:\n  - { alpha3: GBR, defaultCurrency: GBP }\n")

	err := Validate(refSchema(), []*Table{countries})
	if err == nil || !strings.Contains(err.Error(), "no data file declares Currency") {
		t.Errorf("want a no-declaring-data message, got: %v", err)
	}
}

// ---- 1.4 key uniqueness ----

func TestDuplicateKeyIsRejected(t *testing.T) {
	currencies := tableFrom(t, "data/currencies.yaml",
		"type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n  - { code: EUR, numeric: 978 }\n  - { code: GBP, numeric: 999 }\n")

	err := Validate(refSchema(), []*Table{currencies})
	if err == nil {
		t.Fatal("want an error for a duplicate key, got nil")
	}
	for _, want := range []string{"row 3 (GBP)", `Currency "GBP"`, "already declared at data/currencies.yaml row 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

// Two files declaring one identity is the same collision as one file doing it twice — and
// the harder one to see, because neither file looks wrong on its own.
func TestDuplicateKeyAcrossFilesIsRejected(t *testing.T) {
	a := tableFrom(t, "data/currencies.yaml", "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n")
	b := tableFrom(t, "data/more-currencies.yaml", "type: table<Currency>\nitems:\n  - { code: GBP, numeric: 826 }\n")

	err := Validate(refSchema(), []*Table{a, b})
	if err == nil || !strings.Contains(err.Error(), "already declared at data/currencies.yaml row 1") {
		t.Errorf("want a cross-file duplicate error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "data/more-currencies.yaml") {
		t.Errorf("error should name the file that clashes:\n%s", err)
	}
}

// A row with no key has no name to emit a constant under, so it cannot be a table row.
func TestRowWithoutItsKeyIsRejected(t *testing.T) {
	currencies := tableFrom(t, "data/currencies.yaml", "type: table<Currency>\nitems:\n  - { numeric: 826 }\n")

	err := Validate(refSchema(), []*Table{currencies})
	if err == nil || !strings.Contains(err.Error(), "no code field, so the row has no identity") {
		t.Errorf("want a missing-identity error, got: %v", err)
	}
}

// levySchema is a table whose row carries an enum-typed field — the reference shape 2.2
// adds alongside a reference to a table row.
func levySchema() Schema {
	return Schema{
		"TaxType": &Type{Name: "TaxType", Kind: KindEnum, Members: []string{"VAT", "GST"}},
		"Levy": &Type{
			Name: "Levy", External: true, Key: "code",
			Fields: []Field{{Name: "code", Type: "string"}, {Name: "taxType", Type: "TaxType"}},
		},
	}
}

// The failure this exists for is the one the alternative design could not have caught:
// with members collected from data, `Vat` would have *created* a member rather than failed
// — and since an enum serialises as text, the typo would have become a wire token.
func TestUndeclaredEnumMemberIsRejected(t *testing.T) {
	tbl := mustTable(t, "type: table<Levy>\nitems:\n  - { code: STD, taxType: Vat }\n")
	tbl.Source = "data/levies.yaml"

	err := Validate(levySchema(), []*Table{tbl})
	if err == nil {
		t.Fatal("want an error for an undeclared enum member, got nil")
	}
	for _, want := range []string{
		"data/levies.yaml", // which file
		"row 1 (STD)",      // which row, by the identity in it
		`"taxType"`,        // which field
		`"Vat"`,            // the value that is wrong, quoted so the casing is visible
		"VAT, GST",         // what is declared — an enum is short enough to list in full
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

// A declared member passes, so the check above is testing the member set and not merely
// erroring on every enum-typed field.
func TestDeclaredEnumMemberIsAccepted(t *testing.T) {
	tbl := mustTable(t, "type: table<Levy>\nitems:\n  - { code: STD, taxType: VAT }\n")
	tbl.Source = "data/levies.yaml"

	if err := Validate(levySchema(), []*Table{tbl}); err != nil {
		t.Fatalf("want a clean run, got: %v", err)
	}
}

// levyTable is a keyed table with a hoistable field, for the `common:` checks below.
func levyTable() Schema {
	return Schema{"Levy": &Type{
		Name: "Levy", External: true, Key: "code",
		Fields: []Field{
			{Name: "code", Type: "string"},
			{Name: "jurisdiction", Type: "string"},
			{Name: "numeric", Type: "int"},
		},
	}}
}

func validateLevies(t *testing.T, src string) error {
	t.Helper()
	tbl := mustTable(t, src)
	tbl.Source = "data/levies.yaml"
	return Validate(levyTable(), []*Table{tbl})
}

// A row setting a field that `common:` also sets is an error, not a cascade. Override is
// the silent option: `common:` reads as authoritative, so a row quietly differing from it
// is invisible in review — and with the cost of expanding the rows now near zero, leniency
// keeps the ambiguity and buys nothing.
func TestRowOverridingCommonIsRejected(t *testing.T) {
	err := validateLevies(t, "type: table<Levy>\ncommon: { jurisdiction: GB }\nitems:\n"+
		"  - { code: STD, jurisdiction: IE, numeric: 20 }\n")
	if err == nil {
		t.Fatal("want an error for a row overriding common, got nil")
	}
	for _, want := range []string{"data/levies.yaml", "row 1 (STD)", `"jurisdiction"`, "common:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

// Hoisting the identity is rejected at the declaration rather than left to the merge, where
// it would surface as every row claiming the same name — several confusing errors standing
// in for one clear one.
func TestHoistingTheIdentityIsRejected(t *testing.T) {
	err := validateLevies(t, "type: table<Levy>\ncommon: { code: STD }\nitems:\n"+
		"  - { jurisdiction: GB, numeric: 20 }\n")
	if err == nil {
		t.Fatal("want an error for hoisting the key field, got nil")
	}
	if !strings.Contains(err.Error(), "identity field") {
		t.Errorf("want the identity named as the reason:\n%s", err)
	}
}

// A hoisted field is still a field: it is checked against the schema and, where it is a
// reference, resolved — once, against `common:` rather than against every row.
func TestUnresolvedReferenceInCommonIsRejected(t *testing.T) {
	schema := levySchema()
	tbl := mustTable(t, "type: table<Levy>\ncommon: { taxType: Vat }\nitems:\n  - { code: STD }\n")
	tbl.Source = "data/levies.yaml"

	err := Validate(schema, []*Table{tbl})
	if err == nil {
		t.Fatal("want an error for the undeclared member in common, got nil")
	}
	if !strings.Contains(err.Error(), "common: field \"taxType\"") {
		t.Errorf("want the problem named against common:\n%s", err)
	}
	if n := strings.Count(err.Error(), `"Vat"`); n != 1 {
		t.Errorf("want it reported once, got %d times:\n%s", n, err)
	}
}
