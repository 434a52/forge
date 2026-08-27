package c5n

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Schema is the set of declared types, keyed by type name.
type Schema map[string]*Type

// Type is one schema declaration. External types are hand-written runtime types (c5n
// constructs instances but never emits the body). For table<T> data, Key names the
// identity field — it becomes the constant name and the reference target.
type Type struct {
	Name     string
	Source   string // the schema file that declared it, for collision messages
	Kind     string // "" for a record/external type; KindEnum or KindSeries
	External bool
	Key      string
	Fields   []Field           // in declaration order — this is the ctor arg order
	Lookup   []string          // additional unique fields to emit an index for
	Members  []string          // enum members, in declaration order
	Envelope []Field           // series only: the fields that key an entry, not construct it
	Emit     map[string]string // target -> construction recipe; nil = positional-ctor convention
}

// A kind is how a type says it is something other than a plain record.
const (
	// KindEnum makes c5n emit a type *body* rather than instances of a hand-written one.
	KindEnum = "enum"

	// KindSeries is a collection type wrapping a value type, where each entry carries an
	// envelope alongside the value's own fields. `EffectiveDated` is the first: its
	// envelope is the date a rate takes effect from.
	//
	// The envelope is declared here rather than built into c5n because c5n must not know
	// that a temporal series keys on a field called `from`. A series declares what keys it;
	// an entry missing that field is a validation error rather than a guess, which is the
	// whole of "temporality is declared, never sniffed".
	KindSeries = "series"
)

// IsEnum reports whether this type is a generated enum.
func (t *Type) IsEnum() bool { return t.Kind == KindEnum }

// IsSeries reports whether this type is a collection with a declared envelope.
func (t *Type) IsSeries() bool { return t.Kind == KindSeries }

// DeclaresMember reports whether name is one of this enum's declared members. A linear
// scan: an enum's members are a handful of domain terms, not a table.
func (t *Type) DeclaresMember(name string) bool {
	for _, member := range t.Members {
		if member == name {
			return true
		}
	}
	return false
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
		if err := parseSchemaDoc(&doc, schema, p); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	return schema, nil
}

// parseSchemaDoc walks the top-level mapping (TypeName -> declaration). A yaml MappingNode
// stores entries as a flat slice [key0, val0, key1, val1, …], preserving order.
//
// source names the file being read, so a type declared twice can say where the first one was.
// That matters more than it used to: schema files are a *glob*, and as soon as more than one
// producer emits into it — a code-first model on one side, l10n on the other — two files
// declaring one name becomes reachable. Data rows have been checked for this since 1.4; type
// declarations were not, and last-write-wins silently.
func parseSchemaDoc(doc *yaml.Node, out Schema, source string) error {
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
		if err := checkTypeDecl(t); err != nil {
			return fmt.Errorf("type %s: %w", t.Name, err)
		}
		if prev, clash := out[t.Name]; clash {
			where := "earlier in this file"
			if prev.Source != source {
				where = "in " + prev.Source
			}
			return fmt.Errorf("type %s is already declared %s", t.Name, where)
		}
		t.Source = source
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
		case "lookup":
			lookup, err := parseMembers(val) // same shape: a list of field names
			if err != nil {
				return fmt.Errorf("lookup: %w", err)
			}
			t.Lookup = lookup
		case "fields":
			fields, err := parseFields(val)
			if err != nil {
				return err
			}
			t.Fields = fields
		case "emit":
			recipes, err := parseEmit(val)
			if err != nil {
				return err
			}
			t.Emit = recipes
		case "kind":
			t.Kind = val.Value
		case "members":
			members, err := parseMembers(val)
			if err != nil {
				return err
			}
			t.Members = members
		case "envelope":
			envelope, err := parseFields(val)
			if err != nil {
				return err
			}
			t.Envelope = envelope
		default:
			return fmt.Errorf("unknown declaration key %q", key)
		}
	}
	return nil
}

// parseEmit reads the per-type construction override: target -> recipe. A recipe is the
// construction expression for one row, with {field} placeholders. It exists for the types
// the positional-ctor convention can't build — factories, parse-from-string, singletons —
// and it is per-target because the same value is constructed differently in each language.
func parseEmit(node *yaml.Node) (map[string]string, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("emit must be a mapping of target -> recipe")
	}
	recipes := make(map[string]string, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		target, recipe := node.Content[i], node.Content[i+1]
		if recipe.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("emit.%s must be a string recipe", target.Value)
		}
		recipes[target.Value] = recipe.Value
	}
	return recipes, nil
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

// memberName is the shape an enum member has to have to be a legal identifier in every
// target: a letter or underscore, then letters, digits or underscores.
//
// Shape only — a member that happens to be a *keyword* in one target is left to that
// target's compiler. It is an unlikely collision (members are domain vocabulary), and it
// fails loudly in a gate that already runs, pointing back at the schema. What this does
// catch is the mistake an author actually makes: `zero-rated`, `2ndTier`, `zero rated`.
var memberName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// parseMembers reads an enum's member list, keeping declaration order — which is the
// emitted order, and the order a reader compares against the schema.
func parseMembers(node *yaml.Node) ([]string, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("members must be a list of names")
	}
	members := make([]string, 0, len(node.Content))
	for _, m := range node.Content {
		if m.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("members must be a list of names")
		}
		members = append(members, m.Value)
	}
	return members, nil
}

// checkTypeDecl rejects a declaration that is internally contradictory, at the point of
// declaration rather than at the point some writer trips over it.
//
// An enum is declared, never inferred from data. A value in a data file therefore
// *selects* a member and can never create one — which is the same guarantee a reference
// to a table row gets, and it matters more here: an enum member's name is what crosses
// the wire, so a typo that minted a member would mint a wire token with it.
func checkTypeDecl(t *Type) error {
	switch t.Kind {
	case "", KindEnum, KindSeries:
	default:
		return fmt.Errorf("unknown kind %q (the kinds are %q and %q)", t.Kind, KindEnum, KindSeries)
	}
	if len(t.Members) > 0 && !t.IsEnum() {
		return fmt.Errorf("members are only declared on `kind: %s`", KindEnum)
	}
	if len(t.Envelope) > 0 && !t.IsSeries() {
		return fmt.Errorf("an envelope is only declared on `kind: %s`", KindSeries)
	}
	if err := checkLookup(t); err != nil {
		return err
	}

	if t.IsSeries() {
		switch {
		case !t.External:
			return fmt.Errorf("a series is a hand-written runtime type, so it must be external")
		case len(t.Envelope) == 0:
			return fmt.Errorf("a series must declare an envelope — the field(s) that key an entry")
		case len(t.Fields) > 0:
			return fmt.Errorf("a series has an envelope, not fields — the value's fields come from the type it wraps")
		case t.Key != "":
			return fmt.Errorf("a series has no key — its entries are keyed by the envelope")
		}
		return nil
	}

	if !t.IsEnum() {
		return nil
	}

	switch {
	case t.External:
		return fmt.Errorf("an enum cannot be external — c5n emits the type body")
	case len(t.Fields) > 0:
		return fmt.Errorf("an enum has members, not fields")
	case t.Key != "":
		return fmt.Errorf("an enum has no key — a member's name is its identity")
	case len(t.Emit) > 0:
		return fmt.Errorf("an enum needs no emit: recipe — a member is referenced, never constructed")
	case len(t.Members) == 0:
		return fmt.Errorf("an enum must declare at least one member")
	}

	seen := make(map[string]bool, len(t.Members))
	for _, m := range t.Members {
		if !memberName.MatchString(m) {
			return fmt.Errorf("member %q is not a legal identifier in every target", m)
		}
		if seen[m] {
			return fmt.Errorf("member %q is declared twice", m)
		}
		seen[m] = true
	}
	return nil
}

// checkLookup validates the declared secondary indexes.
//
// `key:` names the identity — the constant's name, the reference target, and the form the
// value takes on the wire. `lookup:` names *additional* fields that are also unique and
// worth finding a row by: a Country is identified by its alpha-3 code, but arrives from
// foreign systems as an alpha-2 or a numeric one just as often.
//
// The two are deliberately different declarations rather than a list of equals. Making them
// interchangeable would leave nothing to say which form is canonical, and a reference type
// with two encodings is the thing f8n's wire format exists to prevent.
func checkLookup(t *Type) error {
	if len(t.Lookup) == 0 {
		return nil
	}
	if t.Key == "" {
		return fmt.Errorf("`lookup:` needs a `key:` — an index is a second way to find a row, not the first")
	}

	declared := make(map[string]bool, len(t.Fields))
	for _, f := range t.Fields {
		declared[f.Name] = true
	}

	fieldType := make(map[string]string, len(t.Fields))
	for _, f := range t.Fields {
		fieldType[f.Name] = f.Type
	}

	seen := make(map[string]bool, len(t.Lookup))
	for _, name := range t.Lookup {
		switch {
		case !declared[name]:
			return fmt.Errorf("lookup field %q is not declared on this type", name)
		case name == t.Key:
			return fmt.Errorf("lookup field %q is already the key, which is always indexed", name)
		case seen[name]:
			return fmt.Errorf("lookup field %q is listed twice", name)
		case !isScalar(fieldType[name]):
			// A reference or a nested value has no obvious key to index by, and inventing
			// one would be guessing at semantics nobody has asked for yet.
			return fmt.Errorf("lookup field %q is a %s; only scalar fields can be indexed", name, fieldType[name])
		}
		seen[name] = true
	}
	return nil
}
