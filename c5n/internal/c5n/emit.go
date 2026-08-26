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

// isReference reports whether a field's declared type is one whose values are *named*
// rather than built: a keyed table row (Currency.GBP) or an enum member (TaxType.VAT).
// Either way the emitted expression is a name, not a literal and not a ctor call.
//
// Whether the name actually resolves — a row that exists, a member that is declared — is
// Validate's question, settled before any writer runs, so a writer emits the reference
// without re-checking it.
func isReference(declType string, schema Schema) bool {
	t, ok := schema[declType]
	return ok && (t.Key != "" || t.IsEnum())
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

// valueContext carries what the value-emitter needs that varies by target: how this
// language spells a reference to another table's constant, and a hook to record the
// hand-written types the output ends up referring to (TS needs them as imports; C# does
// not, since a generated partial shares its namespace).
type valueContext struct {
	schema Schema
	target string
	ref    func(refType *Type, value string) string
	use    func(typeName string)
}

// maxNestDepth stops a schema whose type refers to itself from recursing forever. A
// single-field type that contains itself can never be constructed from a finite value, so
// hitting this is a schema bug, not a deep-but-valid case.
const maxNestDepth = 16

// emitValue resolves one authored value to its emitted expression, driven purely by the
// declared type — the three shapes the design names:
//
//   - literal    — a scalar, formatted the way this target spells it
//   - reference  — a keyed table row, emitted as the constant's name
//   - nested ctor — a constructible type, built from the value (recursing)
//
// This is the conformance-critical part: a wrong expression here is wrong data in every
// target at once, which is why it is one shared resolver rather than a branch per writer.
func emitValue(declType, text string, ctx valueContext, depth int) (string, error) {
	switch {
	case isReference(declType, ctx.schema):
		return ctx.ref(ctx.schema[declType], text), nil
	case isScalar(declType):
		return scalarLiteral(declType, text, ctx.target)
	}

	typ, ok := ctx.schema[declType]
	if !ok {
		return "", fmt.Errorf("declared type %s is not in the schema", declType)
	}
	if depth >= maxNestDepth {
		return "", fmt.Errorf("type %s nests more than %d deep — a type that contains itself cannot be built from a value", declType, maxNestDepth)
	}

	// A constructible type with exactly one field may be authored as a plain scalar: the
	// scalar *is* that field's value. It is the wrapper case — Percentage over an exact
	// Rational — and it keeps the data file free of a nested mapping that would carry no
	// information (`rate: 20`, not `rate: { value: 20 }`). Multi-field nesting needs an
	// authored mapping, which the data reader does not accept yet.
	if len(typ.Fields) != 1 {
		return "", fmt.Errorf("%s has %d fields, so it cannot be built from the single value %q — only a one-field type may be authored as a scalar", declType, len(typ.Fields), text)
	}

	inner := typ.Fields[0]
	arg, err := emitValue(inner.Type, text, ctx, depth+1)
	if err != nil {
		return "", fmt.Errorf("%s.%s: %w", typ.Name, inner.Name, err)
	}
	ctx.use(typ.Name)
	return construct(typ, ctx.target, []string{arg}, map[string]string{inner.Name: arg})
}

// rowArgs resolves every declared field of a row to its emitted argument expression,
// returning them both positionally (ctor order) and by name (for recipes).
func rowArgs(row Row, typ *Type, ctx valueContext) ([]string, map[string]string, error) {
	args := make([]string, len(typ.Fields))
	byField := make(map[string]string, len(typ.Fields))
	for i, f := range typ.Fields {
		text, ok := row[f.Name]
		if !ok {
			return nil, nil, fmt.Errorf("field %s: missing from row", f.Name)
		}
		arg, err := emitValue(f.Type, text, ctx, 0)
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
	// C# references a sibling constant by its qualified name: Currency.GBP. Hand-written
	// types need no import — a generated partial shares their namespace.
	ctx := valueContext{
		schema: schema,
		target: "csharp",
		ref:    func(refType *Type, value string) string { return refType.Name + "." + value },
		use:    func(string) {},
	}

	var body strings.Builder
	for _, row := range t.Rows {
		name, err := rowName(row, typ)
		if err != nil {
			return "", fmt.Errorf("%s: %w", typ.Name, err)
		}
		args, byField, err := rowArgs(row, typ, ctx)
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
	b.WriteString(csharpHeader(schemaSrc + " + " + t.Source))
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
	refVals := map[string][]string{} // refType -> identifiers imported from its module
	seen := map[string]bool{}        // "refType.identifier" already imported

	var usedOrder []string    // hand-written types the values construct, first-appearance
	used := map[string]bool{} // already noted

	// importFrom records one identifier to import from a referenced type's generated
	// module, deduped and in first-appearance order so the import block is deterministic.
	importFrom := func(refType, identifier string) {
		key := refType + "." + identifier
		if seen[key] {
			return
		}
		seen[key] = true
		if _, ok := refVals[refType]; !ok {
			refOrder = append(refOrder, refType)
		}
		refVals[refType] = append(refVals[refType], identifier)
	}

	ctx := valueContext{
		schema: schema,
		target: "ts",
		// TS imports exactly what it names, and the two reference shapes name different
		// things. A table row is its own exported constant, so the constant is imported and
		// used bare (GBP). An enum is a single exported const object whose members are
		// properties, so the *type* is imported once however many members are used, and the
		// member is reached through it (TaxType.VAT) — which is also what makes the
		// reference read identically to C#.
		ref: func(refType *Type, value string) string {
			if refType.IsEnum() {
				importFrom(refType.Name, refType.Name)
				return refType.Name + "." + value
			}
			importFrom(refType.Name, value)
			return value
		},
		// TS has no partial, so a constructed hand-written type has to be imported by name.
		use: func(typeName string) {
			if typeName != typ.Name && !used[typeName] {
				used[typeName] = true
				usedOrder = append(usedOrder, typeName)
			}
		},
	}

	var body strings.Builder
	for _, row := range t.Rows {
		name, err := rowName(row, typ)
		if err != nil {
			return "", fmt.Errorf("%s: %w", typ.Name, err)
		}
		args, byField, err := rowArgs(row, typ, ctx)
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
	b.WriteString(tsHeader(schemaSrc + " + " + t.Source))
	// The hand-written types live one dir up from the generated file, by convention. Import
	// specifiers carry the ".js" extension: TypeScript resolves it back to the ".ts" source,
	// and it is what Node's ESM loader requires to run the compiled output directly — without
	// it the package only works inside a bundler.
	fmt.Fprintf(&b, "import { %s } from \"../%s.js\";\n", typ.Name, strings.ToLower(typ.Name))
	for _, u := range usedOrder {
		fmt.Fprintf(&b, "import { %s } from \"../%s.js\";\n", u, strings.ToLower(u))
	}
	for _, rt := range refOrder {
		fmt.Fprintf(&b, "import { %s } from \"./%s.data.js\";\n", strings.Join(refVals[rt], ", "), strings.ToLower(rt))
	}
	b.WriteString("\n")
	b.WriteString(body.String())
	return b.String(), nil
}

// Enums are the first type c5n emits a *body* for rather than instances of a hand-written
// one. They are generated from the schema alone: members are declared, so an enum exists
// whether or not any data row has referenced it yet.
//
// A member's name is emitted verbatim in both targets, and no casing is applied anywhere.
// That is not a style choice — f8n serialises an enum as text, so the member name is the
// token that crosses the wire, and a generator that rewrote `standard` to `Standard` would
// be rewriting a published contract. It is also the only rule under which an acronym
// survives: any automatic PascalCase turns VAT into Vat.

// emitEnumCSharp renders a generated enum as a plain C# enum in the target's namespace.
func emitEnumCSharp(typ *Type, target Target, schemaSrc string) string {
	var b strings.Builder
	b.WriteString(csharpHeader(schemaSrc))
	fmt.Fprintf(&b, "namespace %s;\n\n", target.Namespace)
	fmt.Fprintf(&b, "public enum %s\n{\n", typ.Name)
	for _, member := range typ.Members {
		fmt.Fprintf(&b, "    %s,\n", member)
	}
	b.WriteString("}\n")
	return b.String()
}

// emitEnumTS renders a generated enum as a frozen const object plus a union type of its
// values — deliberately not a TS `enum`.
//
// The const object gives `TaxType.VAT` as a value expression, so a reference reads the same
// in both targets and the shared value-emitter needs no per-target spelling. Each member's
// value is its own name, so the TS runtime value *is* the serialised token that C# writes
// for the same member — the two agree by construction rather than through a converter
// someone has to keep in sync. A TS `enum` would give neither: it is number-backed, and it
// is not erasable syntax, so it is rejected by type-stripping runtimes.
//
// The const and the type share a name legally: TypeScript keeps values and types in
// separate namespaces, so `TaxType` is both the object and the union, which is what makes
// it read like an enum at the use site.
func emitEnumTS(typ *Type, schemaSrc string) string {
	var b strings.Builder
	b.WriteString(tsHeader(schemaSrc))
	b.WriteString("\n")
	fmt.Fprintf(&b, "export const %s = {\n", typ.Name)
	for _, member := range typ.Members {
		fmt.Fprintf(&b, "  %s: %s,\n", member, quote(member))
	}
	b.WriteString("} as const;\n\n")
	fmt.Fprintf(&b, "export type %s = (typeof %s)[keyof typeof %s];\n", typ.Name, typ.Name, typ.Name)
	return b.String()
}

// The headers take one already-joined source list rather than schema + data, because not
// every emitted unit has a data file: an enum's members are declared, so it is generated
// from the schema alone.

func csharpHeader(sources string) string {
	return "// <auto-generated>\n" +
		"//   Generated by c5n from " + sources + " — DO NOT EDIT.\n" +
		"//   Reproducible: re-run `c5n build`. The drift-guard regenerates + diffs against this file.\n" +
		"// </auto-generated>\n"
}

func tsHeader(sources string) string {
	return "// GENERATED by c5n from " + sources + " — DO NOT EDIT.\n" +
		"// Reproducible: re-run `c5n build`. The drift-guard regenerates + diffs against this file.\n"
}
