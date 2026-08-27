using System.Text.Encodings.Web;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace F8n;

/// <summary>
/// The JSON entry point: <see cref="Options"/> carries every f8n converter, so nesting works.
/// </summary>
/// <remarks>
/// <para>
/// C# has no equivalent of JavaScript's <c>toJSON</c> protocol, so the shape here is a
/// converter per type rather than a method per type. It does the same job — a value nested
/// anywhere serialises correctly without its container knowing anything about it — through the
/// extension point the platform actually offers. The asymmetry is in the mechanism, not the
/// contract: both sides produce identical bytes, which is what the vectors pin.
/// </para>
/// <para>
/// Converters also sidestep the immutability problem. f8n's primitives have private,
/// validating constructors and no parameterless one, because a Money with no currency must not
/// be a reachable state — and a deserialiser that constructs-then-populates makes every such
/// state reachable mid-populate. A converter reads the payload and calls the same factory a
/// caller would.
/// </para>
/// </remarks>
public static class F8nJson
{
    /// <summary>Serializer options carrying every f8n converter.</summary>
    public static readonly JsonSerializerOptions Options = Build();

    private static JsonSerializerOptions Build()
    {
        var options = new JsonSerializerOptions
        {
            // Match JavaScript's JSON.stringify, which does not escape non-ASCII. .NET's
            // default encoder escapes it (£ becomes £), which would make the two
            // languages emit different bytes for one value — exactly the divergence the wire
            // format exists to prevent. No f8n wire form carries non-ASCII today, so this is
            // a seam held open rather than a bug fixed; "unsafe" in the name refers to
            // embedding JSON in HTML, which a wire body is not.
            Encoder = JavaScriptEncoder.UnsafeRelaxedJsonEscaping,

            // A container needs no converter of its own — both platforms walk its properties
            // and serialise each value with that value's own rule. But they disagree by
            // default on how a property is *named*: JavaScript emits the field as written
            // (camelCase, in this codebase), and .NET emits the CLR property name
            // (PascalCase). One value, two spellings, and neither language's tests would
            // notice on their own.
            PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        };

        // An enum serialises as TEXT, never as an ordinal — f8n/DESIGN.md -> Enums travel as
        // their member name. .NET's default is the numeric value, which would make the member
        // list's *order* part of the contract; TypeScript's generated enums are string-valued
        // already, so without this the two sides disagree on every enum that crosses.
        options.Converters.Add(new JsonStringEnumConverter());
        options.Converters.Add(new MoneyConverter());
        options.Converters.Add(new PercentageConverter());
        options.Converters.Add(new LocalDateConverter());
        options.Converters.Add(new CurrencyConverter());
        options.Converters.Add(new CountryConverter());
        return options;
    }

    /// <summary>Serialises to the canonical wire form.</summary>
    public static string Serialize<T>(T value) => JsonSerializer.Serialize(value, Options);

    /// <summary>Reads the canonical wire form, or throws.</summary>
    public static T Deserialize<T>(string json)
    {
        return JsonSerializer.Deserialize<T>(json, Options)
            ?? throw new JsonException("null is not a value");
    }
}

/// <summary>
/// <c>{"amount":"12.34","currency":"GBP"}</c> — the one primitive whose wire form is not its
/// field layout, which is exactly when a type needs a converter of its own.
/// </summary>
/// <remarks>
/// Property order is part of the contract, not a detail: the vectors pin exact bytes and
/// JavaScript emits object keys in insertion order, so amount precedes currency on both sides.
/// </remarks>
public sealed class MoneyConverter : JsonConverter<Money>
{
    public override Money Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        if (reader.TokenType != JsonTokenType.StartObject)
        {
            throw new JsonException("a Money is an object with amount and currency");
        }

        string? amount = null;
        string? currency = null;
        while (reader.Read())
        {
            if (reader.TokenType == JsonTokenType.EndObject)
            {
                break;
            }
            if (reader.TokenType != JsonTokenType.PropertyName)
            {
                throw new JsonException("expected a property name");
            }

            var name = reader.GetString();
            reader.Read();
            switch (name)
            {
                case "amount":
                    amount = reader.GetString();
                    break;
                case "currency":
                    currency = reader.GetString();
                    break;
                default:
                    // Rejected rather than ignored, and that is what makes the wire round trip
                    // hold: an ignored field is dropped on the way back out, so one value would
                    // have two encodings and wire equality would stop meaning value equality.
                    throw new JsonException($"a Money has amount and currency only; got {name}");
            }
        }

        if (amount is null)
        {
            throw new JsonException("a Money needs an amount, as a string");
        }
        if (currency is null)
        {
            throw new JsonException("a Money needs a currency code, as a string");
        }
        var found = Currency.ByCode(currency) ?? throw new JsonException($"unknown currency {currency}");
        return Money.FromMajor(amount, found);
    }

    public override void Write(Utf8JsonWriter writer, Money value, JsonSerializerOptions options)
    {
        writer.WriteStartObject();
        writer.WriteString("amount", value.Amount);
        writer.WriteString("currency", value.Currency.Code);
        writer.WriteEndObject();
    }
}

/// <summary>The canonical <c>num/den</c> string. Strict — it shares <c>Parse</c>.</summary>
public sealed class PercentageConverter : JsonConverter<Percentage>
{
    public override Percentage Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var text = reader.TokenType == JsonTokenType.String
            ? reader.GetString()
            : throw new JsonException("a Percentage is the string \"num/den\"");
        return Percentage.Parse(text!);
    }

    public override void Write(Utf8JsonWriter writer, Percentage value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value.ToString());
    }
}

/// <summary>The canonical <c>YYYY-MM-DD</c> string. Strict — it shares <c>Parse</c>.</summary>
public sealed class LocalDateConverter : JsonConverter<LocalDate>
{
    public override LocalDate Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var text = reader.TokenType == JsonTokenType.String
            ? reader.GetString()
            : throw new JsonException("a LocalDate is the string \"YYYY-MM-DD\"");
        return LocalDate.Parse(text!);
    }

    public override void Write(Utf8JsonWriter writer, LocalDate value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value.ToString());
    }
}

/// <summary>
/// A reference type travels as its key, never as an inlined record — and is read back through
/// the generated index, strictly. The alpha code and nothing else; the other code forms are an
/// ingestion concern (<c>Country.Find</c>), and accepting them here would give one value
/// several encodings.
/// </summary>
public sealed class CurrencyConverter : JsonConverter<Currency>
{
    public override Currency Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var code = reader.TokenType == JsonTokenType.String
            ? reader.GetString()
            : throw new JsonException("a Currency is its alpha code, as a string");
        return Currency.ByCode(code!) ?? throw new JsonException($"unknown currency {code}");
    }

    public override void Write(Utf8JsonWriter writer, Currency value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value.Code);
    }
}

/// <summary>The alpha-3 identity and nothing else. See <see cref="CurrencyConverter"/>.</summary>
public sealed class CountryConverter : JsonConverter<Country>
{
    public override Country Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var alpha3 = reader.TokenType == JsonTokenType.String
            ? reader.GetString()
            : throw new JsonException("a Country is its alpha-3 code, as a string");
        return Country.ByAlpha3(alpha3!) ?? throw new JsonException($"unknown country {alpha3}");
    }

    public override void Write(Utf8JsonWriter writer, Country value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value.Alpha3);
    }
}
