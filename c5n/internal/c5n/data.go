package c5n

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Table is one data collection bound to a schema type: `type: table<Currency>` + rows.
type Table struct {
	Kind     string           // "table", "list", "tree", "EffectiveDated"
	ElemType string           // e.g. "Currency"
	Rows     []map[string]any // each row: field name -> raw value (string / int / bool)
	Source   string           // the data file it came from (relative to root; for headers)
}

// dataFile is the on-disk shape of a data file.
type dataFile struct {
	Type  string           `yaml:"type"`
	Items []map[string]any `yaml:"items"`
}

// typeExpr matches a collection type like "table<Currency>" → kind, element.
var typeExpr = regexp.MustCompile(`^(\w+)<(\w+)>$`)

// LoadData reads every data file into a Table. Row values keep their YAML type (int vs
// string), which the emitter needs to choose a literal vs a reference.
func LoadData(root string, paths []string) ([]*Table, error) {
	var tables []*Table
	for _, p := range paths {
		raw, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		}
		var df dataFile
		if err := yaml.Unmarshal(raw, &df); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		m := typeExpr.FindStringSubmatch(df.Type)
		if m == nil {
			return nil, fmt.Errorf("%s: bad type %q (want kind<Elem>)", p, df.Type)
		}
		tables = append(tables, &Table{Kind: m[1], ElemType: m[2], Rows: df.Items, Source: p})
	}
	return tables, nil
}
