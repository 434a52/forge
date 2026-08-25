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
