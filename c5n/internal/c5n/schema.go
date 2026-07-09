package c5n

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Schema is the set of declared types, keyed by type name.
type Schema map[string]*Type

// Type is one schema declaration. External types are hand-written runtime types (c5n
// constructs instances but never emits the body). For table<T> data, Key names the
// identity field — it becomes the constant name and the reference target.
type Type struct {
	Name     string
	External bool
	Key      string
	Fields   []Field // in declaration order — this is the ctor arg order
}

// Field is one declared field: a name and a declared type. The type is a scalar
// ("string", "int", "decimal", "bool") or the name of another declared Type.
type Field struct {
	Name string
	Type string
}

// scalarTypes are the built-in leaf types; any other type name refers to another Type.
var scalarTypes = map[string]bool{
	"string": true, "int": true, "decimal": true, "bool": true,
}

func isScalar(t string) bool { return scalarTypes[t] }

// LoadSchema reads and merges every schema file. It parses via yaml.Node rather than a
// Go map because a map loses key order, and field order is load-bearing (ctor args).
func LoadSchema(root string, paths []string) (Schema, error) {
	schema := Schema{}
	for _, p := range paths {
		raw, err := os.ReadFile(filepath.Join(root, p))
		if err != nil {
			return nil, fmt.Errorf("read schema: %w", err)
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		if err := parseSchemaDoc(&doc, schema); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	return schema, nil
}

// parseSchemaDoc walks the top-level mapping (TypeName -> declaration). A yaml MappingNode
// stores entries as a flat slice [key0, val0, key1, val1, …], preserving order.
func parseSchemaDoc(doc *yaml.Node, out Schema) error {
	if len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping of type declarations")
	}
	for i := 0; i < len(root.Content); i += 2 {
		t := &Type{Name: root.Content[i].Value}
		if err := parseTypeDecl(root.Content[i+1], t); err != nil {
			return fmt.Errorf("type %s: %w", t.Name, err)
		}
		out[t.Name] = t
	}
	return nil
}

func parseTypeDecl(decl *yaml.Node, t *Type) error {
	if decl.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping")
	}
	for i := 0; i < len(decl.Content); i += 2 {
		key, val := decl.Content[i].Value, decl.Content[i+1]
		switch key {
		case "external":
			t.External = val.Value == "true"
		case "key":
			t.Key = val.Value
		case "fields":
			fields, err := parseFields(val)
			if err != nil {
				return err
			}
			t.Fields = fields
		case "kind", "emit":
			// enum kinds and emit: overrides are deferred rooms — parsed-tolerant, not yet used.
		default:
			return fmt.Errorf("unknown declaration key %q", key)
		}
	}
	return nil
}

func parseFields(node *yaml.Node) ([]Field, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("fields must be a mapping")
	}
	fields := make([]Field, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		fields = append(fields, Field{Name: node.Content[i].Value, Type: node.Content[i+1].Value})
	}
	return fields, nil
}
