package c5n

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The value-emitter is the conformance-critical heart: for each field it resolves, driven
// purely by the declared field type, to one of three shapes — literal, reference, or
// nested-ctor. (Nested-ctor is deferred; the current data only needs literal + reference.)
// A wrong expression here is wrong data everywhere it is emitted, so this is what the
// golden vectors pin hardest.
//
// Values arrive as the source text the data file authored (see Row). The declared type
// decides how that text is rendered — never YAML's own inference, which cannot represent
// a decimal without going through float64 and losing digits.

// decimalLit is a plain decimal literal: optional sign, digits, optional fraction.
// Exponent form is rejected rather than translated — the targets spell exponents
// differently and a decimal written in full is unambiguous in both.
var decimalLit = regexp.MustCompile(`^[+-]?(\d+(\.\d+)?|\.\d+)$`)

// placeholder matches a {field} slot in an emit: recipe.
var placeholder = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// scalarLiteral renders a scalar's source text as a per-target literal, validating that
// the text actually is what the schema declared it to be.
func scalarLiteral(declType, text, target string) (string, error) {
	switch declType {
	case "string":
		return quote(text), nil
	case "int":
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			return "", fmt.Errorf("expected int, got %q", text)
		}
		return text, nil
	case "decimal":
		if !decimalLit.MatchString(text) {
			return "", fmt.Errorf("expected a plain decimal, got %q (exponent form is not accepted — write it in full)", text)
		}
		// Emitted verbatim: the authored digits reach the target unchanged. C# takes the
		// decimal suffix; TS has no decimal type, so a bare literal is a float64 at
		// runtime — a type whose value must survive exactly needs an emit: recipe that
		// hands the target a string.
		if target == "csharp" {
			return text + "m", nil
		}
		return text, nil
	case "bool":
		if text != "true" && text != "false" {
			return "", fmt.Errorf("expected bool, got %q", text)
		}
		return text, nil
	}
	return "", fmt.Errorf("unknown scalar type %q", declType)
}

// quote wraps a string in double quotes (valid in both C# and TS), escaping backslash and
// quote. UTF-8 (£, €, …) passes through unescaped.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// isReference reports whether a field's declared type is a keyed table type — i.e. a value
// of that field names a constant in another generated table (Currency.GBP), not a literal.
func isReference(declType string, schema Schema) bool {
	t, ok := schema[declType]
	return ok && t.Key != ""
}

// construct renders one row's construction expression: the type's emit: recipe for this
// target when it declares one, else the positional-ctor convention.
//
// A recipe's {field} placeholders are substituted with the *fully emitted* argument
// expression — quoted strings, suffixed decimals, resolved references — the same values
// the convention would pass positionally. A recipe therefore chooses the shape of the
// call (factory, parse-from-string, argument order); it does not re-spell the literals.
func construct(typ *Type, target string, args []string, byField map[string]string) (string, error) {
	recipe, ok := typ.Emit[target]
	if !ok {
		return "new " + typ.Name + "(" + strings.Join(args, ", ") + ")", nil
	}
	var unknown []string
	out := placeholder.ReplaceAllStringFunc(recipe, func(m string) string {
		name := m[1 : len(m)-1]
		val, ok := byField[name]
		if !ok {
			unknown = append(unknown, name)
			return m
		}
		return val
	})
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return "", fmt.Errorf("emit.%s references undeclared field(s): %s", target, strings.Join(unknown, ", "))
	}
	return out, nil
}

// rowArgs resolves every declared field of a row to its emitted argument expression,
// returning them both positionally (ctor order) and by name (for recipes). resolveRef
// renders a reference the way the target spells it, and lets the caller record imports.
func rowArgs(row Row, typ *Type, schema Schema, target string, resolveRef func(fieldType, value string) string) ([]string, map[string]string, error) {
	args := make([]string, len(typ.Fields))
	byField := make(map[string]string, len(typ.Fields))
	for i, f := range typ.Fields {
		text, ok := row[f.Name]
		if !ok {
			return nil, nil, fmt.Errorf("field %s: missing from row", f.Name)
		}
		var arg string
		var err error
		switch {
		case isReference(f.Type, schema):
			arg = resolveRef(f.Type, text)
		case isScalar(f.Type):
			arg, err = scalarLiteral(f.Type, text, target)
		default:
			err = fmt.Errorf("type %s is neither scalar nor a keyed reference", f.Type)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		args[i] = arg
		byField[f.Name] = arg
	}
	return args, byField, nil
}

// rowName returns the row's identity — the value of the type's key field, which becomes
// the emitted constant's name.
func rowName(row Row, typ *Type) (string, error) {
	name, ok := row[typ.Key]
	if !ok {
		return "", fmt.Errorf("row is missing its key field %q", typ.Key)
	}
	return name, nil
}

// emitCSharp renders one table<T> to a C# partial-class file of static readonly constants.
func emitCSharp(t *Table, typ *Type, schema Schema, target Target, schemaSrc string) (string, error) {
	// C# references a sibling constant by its qualified name: Currency.GBP.
	ref := func(fieldType, value string) string { return fieldType + "." + value }

	var body strings.Builder
	for _, row := range t.Rows {
		name, err := rowName(row, typ)
		if err != nil {
			return "", fmt.Errorf("%s: %w", typ.Name, err)
		}
		args, byField, err := rowArgs(row, typ, schema, "csharp", ref)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
		}
		expr, err := construct(typ, "csharp", args, byField)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
		}
		fmt.Fprintf(&body, "    public static readonly %s %s = %s;\n", typ.Name, name, expr)
	}

	var b strings.Builder
	b.WriteString(csharpHeader(schemaSrc, t.Source))
	fmt.Fprintf(&b, "namespace %s;\n\n", target.Namespace)
	fmt.Fprintf(&b, "public partial class %s\n{\n", typ.Name)
	b.WriteString(body.String())
	b.WriteString("}\n")
	return b.String(), nil
}

// emitTS renders one table<T> to a TypeScript module of exported const instances. Unlike
// C#, TS references are bare imported names (GBP), so the writer also collects the imports.
func emitTS(t *Table, typ *Type, schema Schema, schemaSrc string) (string, error) {
	var refOrder []string            // referenced types, in first-appearance order
	refVals := map[string][]string{} // refType -> referenced values (deduped, ordered)
	seen := map[string]bool{}        // "refType.value" already imported

	ref := func(fieldType, value string) string {
		if key := fieldType + "." + value; !seen[key] {
			seen[key] = true
			if _, ok := refVals[fieldType]; !ok {
				refOrder = append(refOrder, fieldType)
			}
			refVals[fieldType] = append(refVals[fieldType], value)
		}
		return value // bare imported name
	}

	var body strings.Builder
	for _, row := range t.Rows {
		name, err := rowName(row, typ)
		if err != nil {
			return "", fmt.Errorf("%s: %w", typ.Name, err)
		}
		args, byField, err := rowArgs(row, typ, schema, "ts", ref)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
		}
		expr, err := construct(typ, "ts", args, byField)
		if err != nil {
			return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
		}
		fmt.Fprintf(&body, "export const %s = %s;\n", name, expr)
	}

	var b strings.Builder
	b.WriteString(tsHeader(schemaSrc, t.Source))
	// The hand-written type lives one dir up from the generated file, by convention.
	fmt.Fprintf(&b, "import { %s } from \"../%s\";\n", typ.Name, strings.ToLower(typ.Name))
	for _, rt := range refOrder {
		fmt.Fprintf(&b, "import { %s } from \"./%s.data\";\n", strings.Join(refVals[rt], ", "), strings.ToLower(rt))
	}
	b.WriteString("\n")
	b.WriteString(body.String())
	return b.String(), nil
}

func csharpHeader(schemaSrc, dataSrc string) string {
	return "// <auto-generated>\n" +
		"//   Generated by c5n from " + schemaSrc + " + " + dataSrc + " — DO NOT EDIT.\n" +
		"//   Reproducible: re-run `c5n build`. The drift-guard regenerates + diffs against this file.\n" +
		"// </auto-generated>\n"
}

func tsHeader(schemaSrc, dataSrc string) string {
	return "// GENERATED by c5n from " + schemaSrc + " + " + dataSrc + " — DO NOT EDIT.\n" +
		"// Reproducible: re-run `c5n build`. The drift-guard regenerates + diffs against this file.\n"
}
