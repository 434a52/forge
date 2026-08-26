package c5n

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// parseSchema runs schema text through the real reader, so these tests exercise the path
// from authored YAML to Type rather than a hand-built struct.
func parseSchema(t *testing.T, src string) (Schema, error) {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	schema := Schema{}
	return schema, parseSchemaDoc(&doc, schema)
}

// Members are declared, and declaration order is the emitted order — so the schema file is
// the thing a reviewer diffs to see the enum change.
func TestEnumMembersKeepDeclarationOrder(t *testing.T) {
	schema, err := parseSchema(t, "TaxType:\n  kind: enum\n  members: [VAT, GST, SalesTax]\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	typ := schema["TaxType"]
	if !typ.IsEnum() {
		t.Fatalf("want an enum, got kind %q", typ.Kind)
	}
	if got := strings.Join(typ.Members, ","); got != "VAT,GST,SalesTax" {
		t.Errorf("members reordered: %s", got)
	}
}

// A declaration that contradicts itself is rejected where it is written, rather than where
// some writer later trips over it. Each of these is a real way to mis-declare an enum.
func TestBadTypeDeclarationsAreRejected(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"unknown kind", "T:\n  kind: record\n  members: [A]\n", "unknown kind"},
		{"external enum", "T:\n  kind: enum\n  external: true\n  members: [A]\n", "cannot be external"},
		{"enum with fields", "T:\n  kind: enum\n  members: [A]\n  fields: { x: string }\n", "members, not fields"},
		{"enum with key", "T:\n  kind: enum\n  key: x\n  members: [A]\n", "no key"},
		{"enum with emit", "T:\n  kind: enum\n  members: [A]\n  emit: { csharp: \"X\" }\n", "no emit"},
		{"enum with no members", "T:\n  kind: enum\n", "at least one member"},
		{"duplicate member", "T:\n  kind: enum\n  members: [A, B, A]\n", "declared twice"},
		{"member is not an identifier", "T:\n  kind: enum\n  members: [zero-rated]\n", "not a legal identifier"},
		{"members without kind", "T:\n  external: true\n  members: [A]\n", "only declared on"},
		{"unknown kind names both", "T:\n  kind: record\n", "series"},
		{"generated series", "T:\n  kind: series\n  envelope: { from: string }\n", "must be external"},
		{"series with no envelope", "T:\n  kind: series\n  external: true\n", "must declare an envelope"},
		{"series with fields", "T:\n  kind: series\n  external: true\n  envelope: { from: string }\n  fields: { x: string }\n", "envelope, not fields"},
		{"series with key", "T:\n  kind: series\n  external: true\n  key: x\n  envelope: { from: string }\n", "no key"},
		{"envelope without kind", "T:\n  external: true\n  envelope: { from: string }\n", "only declared on"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseSchema(t, c.src); err == nil {
				t.Fatal("want an error, got nil")
			} else if !strings.Contains(err.Error(), c.want) {
				t.Errorf("want an error mentioning %q, got: %v", c.want, err)
			}
		})
	}
}
