package c5n

import (
	"fmt"
	"strconv"
	"strings"
)

// The value-emitter is the conformance-critical heart: for each field it resolves, driven
// purely by the declared field type, to one of three shapes — literal, reference, or
// nested-ctor. (Nested-ctor is deferred; the current data only needs literal + reference.)
// A wrong expression here is wrong data everywhere it is emitted, so this is what the
// golden vectors pin hardest.

// scalarLiteral renders a scalar value as a per-target literal.
func scalarLiteral(declType string, raw any, target string) (string, error) {
	switch declType {
	case "string":
		s, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("expected string, got %T", raw)
		}
		return quote(s), nil
	case "int":
		switch n := raw.(type) {
		case int:
			return strconv.Itoa(n), nil
		case int64:
			return strconv.FormatInt(n, 10), nil
		default:
			return "", fmt.Errorf("expected int, got %T", raw)
		}
	case "decimal":
		lit := fmt.Sprintf("%v", raw)
		if target == "csharp" {
			lit += "m" // C# decimal literal suffix
		}
		return lit, nil
	case "bool":
		return fmt.Sprintf("%v", raw), nil
	}
	return "", fmt.Errorf("unknown scalar type %q", declType)
}

// quote wraps a string in double quotes (valid in both C# and TS), escaping backslash and
// quote. UTF-8 (£, €, …) passes through unescaped.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// asString returns the value as a plain string (used for keys and reference values, which
// are always string-typed).
func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// isReference reports whether a field's declared type is a keyed table type — i.e. a value
// of that field names a constant in another generated table (Currency.GBP), not a literal.
func isReference(declType string, schema Schema) bool {
	t, ok := schema[declType]
	return ok && t.Key != ""
}

// emitCSharp renders one table<T> to a C# partial-class file of static readonly constants.
func emitCSharp(t *Table, typ *Type, schema Schema, target Target, schemaSrc string) (string, error) {
	var body strings.Builder
	for _, row := range t.Rows {
		name := asString(row[typ.Key])
		args := make([]string, len(typ.Fields))
		for i, f := range typ.Fields {
			raw := row[f.Name]
			var arg string
			var err error
			if isReference(f.Type, schema) {
				// C# references a sibling constant by its qualified name: Currency.GBP.
				arg = f.Type + "." + asString(raw)
			} else if isScalar(f.Type) {
				arg, err = scalarLiteral(f.Type, raw, "csharp")
			} else {
				err = fmt.Errorf("field %s: type %s is neither scalar nor a keyed reference", f.Name, f.Type)
			}
			if err != nil {
				return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
			}
			args[i] = arg
		}
		fmt.Fprintf(&body, "    public static readonly %s %s = new %s(%s);\n",
			typ.Name, name, typ.Name, strings.Join(args, ", "))
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
	var refOrder []string             // referenced types, in first-appearance order
	refVals := map[string][]string{}  // refType -> referenced values (deduped, ordered)
	seen := map[string]bool{}         // "refType.value" already imported

	var body strings.Builder
	for _, row := range t.Rows {
		name := asString(row[typ.Key])
		args := make([]string, len(typ.Fields))
		for i, f := range typ.Fields {
			raw := row[f.Name]
			if isReference(f.Type, schema) {
				val := asString(raw)
				args[i] = val // bare imported name
				if key := f.Type + "." + val; !seen[key] {
					seen[key] = true
					if _, ok := refVals[f.Type]; !ok {
						refOrder = append(refOrder, f.Type)
					}
					refVals[f.Type] = append(refVals[f.Type], val)
				}
			} else if isScalar(f.Type) {
				lit, err := scalarLiteral(f.Type, raw, "ts")
				if err != nil {
					return "", fmt.Errorf("%s.%s: %w", typ.Name, name, err)
				}
				args[i] = lit
			} else {
				return "", fmt.Errorf("%s.%s: field %s: type %s is neither scalar nor a keyed reference",
					typ.Name, name, f.Name, f.Type)
			}
		}
		fmt.Fprintf(&body, "export const %s = new %s(%s);\n", name, typ.Name, strings.Join(args, ", "))
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
