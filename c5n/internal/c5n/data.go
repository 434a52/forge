package c5n

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Table is one data collection bound to a schema type: `type: table<Currency>` + rows.
//
// Common holds the fields hoisted out of every row (`common:`). It is kept separate from
// the rows here and merged in later, after validation, so that a mistake in it is reported
// once against `common:` rather than once per row it was copied into.
type Table struct {
	Kind     string // "table", "list", "tree", "EffectiveDated"
	ElemType string // e.g. "Currency"
	Common   Row    // fields constant across every row; nil when the file declares none
	Rows     []Row
	Source   string // the data file it came from (relative to root; for headers)
}

// Row is one data row: field name -> the scalar's source text, exactly as authored.
//
// Text, not a decoded Go value. Decoding through `any` yields float64 for anything
// fractional, and float64 cannot hold what a C# decimal can — so a value round-tripped
// that way is silently altered before it is ever emitted. Keeping the authored text means
// the emitter reproduces what the data file says; the declared field type (not YAML's
// inference) decides how it is rendered.
type Row map[string]string

// typeExpr matches a collection type like "table<Currency>" → kind, element.
var typeExpr = regexp.MustCompile(`^(\w+)<(\w+)>$`)

// LoadData reads every data file into a Table.
func LoadData(root string, paths []string) ([]*Table, error) {
	var tables []*Table
	for _, p := range paths {
		raw, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		t, err := parseDataDoc(&doc)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		t.Source = p
		tables = append(tables, t)
	}
	return tables, nil
}

// parseDataDoc walks the top-level mapping: `type:` plus `items:`.
func parseDataDoc(doc *yaml.Node) (*Table, error) {
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a mapping with `type` and `items`")
	}

	t := &Table{}
	var items *yaml.Node
	for i := 0; i < len(root.Content); i += 2 {
		key, val := root.Content[i].Value, root.Content[i+1]
		switch key {
		case "type":
			m := typeExpr.FindStringSubmatch(val.Value)
			if m == nil {
				return nil, fmt.Errorf("bad type %q (want kind<Elem>)", val.Value)
			}
			t.Kind, t.ElemType = m[1], m[2]
		case "common":
			common, err := parseRow(val)
			if err != nil {
				return nil, fmt.Errorf("common: %w", err)
			}
			t.Common = common
		case "items":
			items = val
		default:
			return nil, fmt.Errorf("unknown key %q (want `type`, `common` or `items`)", key)
		}
	}
	if t.Kind == "" {
		return nil, fmt.Errorf("missing `type`")
	}
	if items == nil {
		return nil, fmt.Errorf("missing `items`")
	}
	if items.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("`items` must be a list of rows")
	}

	for i, item := range items.Content {
		row, err := parseRow(item)
		if err != nil {
			return nil, fmt.Errorf("items[%d]: %w", i, err)
		}
		t.Rows = append(t.Rows, row)
	}
	return t, nil
}

// parseRow reads one row mapping, keeping each scalar's source text verbatim.
func parseRow(node *yaml.Node) (Row, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("row must be a mapping of field -> value")
	}
	row := make(Row, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		key, val := node.Content[i], node.Content[i+1]
		if val.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("field %s: expected a scalar value", key.Value)
		}
		if val.Tag == "!!null" {
			return nil, fmt.Errorf("field %s: null is not a supported value", key.Value)
		}
		row[key.Value] = val.Value
	}
	return row, nil
}

// MergeCommon copies each table's hoisted fields into every row, so that everything
// downstream sees complete rows and the emitted output is identical to a file that wrote
// every field out. Purely an authoring affordance: nothing past this point knows it happened.
//
// It runs *after* validation, which is what lets a mistake in `common:` be reported against
// `common:` — once — instead of appearing in every row it was copied into and being reported
// against each of them. Validation has already rejected an overlap between the two, so this
// never overwrites a value a row set for itself.
func MergeCommon(tables []*Table) {
	for _, t := range tables {
		if len(t.Common) == 0 {
			continue
		}
		for _, row := range t.Rows {
			for field, value := range t.Common {
				row[field] = value
			}
		}
	}
}
