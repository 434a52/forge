// run-vector — executes a vector file and reports what this implementation produced.
//
// Deliberately thin: it makes no assertions and uses no test framework. Comparison belongs
// to the driver, so the same driver can be pointed at somebody else's implementation and
// audit it the same way. This one reads the cases and ignores any expected values in the
// file, which is what keeps it from grading its own work.

using System.Globalization;
using System.Numerics;
using System.Text.Json;
using System.Text.Json.Nodes;
using F8n;

if (args.Length < 1)
{
    Console.Error.WriteLine("usage: run-vector <vectors.json>");
    return 1;
}

JsonNode document;
try
{
    document = JsonNode.Parse(File.ReadAllText(args[0]))
        ?? throw new InvalidOperationException("empty vector file");
}
catch (Exception ex)
{
    Console.Error.WriteLine($"run-vector: cannot read {args[0]}: {ex.Message}");
    return 1;
}

// The op alone is not enough to dispatch: "parse" means one thing for a Percentage and
// another for a LocalDate. The subject names which type's vectors these are, so one runner
// per language covers every f8n subject rather than one binary per type.
var subject = document["subject"]?.GetValue<string>() ?? "";
if (subject is not ("f8n.Percentage" or "f8n.LocalDate" or "f8n.EffectiveDated" or "f8n.Rounding" or "f8n.Money" or "f8n.Country" or "f8n.CultureCanary"))
{
    Console.Error.WriteLine($"run-vector: unknown subject \"{subject}\"");
    return 2;
}

var results = new JsonArray();

foreach (var group in document["groups"]!.AsArray())
{
    foreach (var testCase in group!["cases"]!.AsArray())
    {
        var id = testCase!["id"]!.GetValue<string>();
        var op = testCase["op"]!.GetValue<string>();

        var inputs = new List<string>();
        foreach (var input in testCase["in"]!.AsArray())
        {
            inputs.Add(input!.GetValue<string>());
        }

        // An unknown op is a fault in the harness, not a failing case — reporting it as a
        // case error would let a reject case "pass" for entirely the wrong reason.
        if (!IsKnownOperation(subject, op))
        {
            Console.Error.WriteLine($"run-vector: {id}: unknown op \"{op}\"");
            return 2;
        }

        var result = new JsonObject { ["id"] = id };
        try
        {
            result["out"] = Execute(subject, op, inputs);
        }
        catch (Exception ex)
        {
            result["error"] = ex.Message;
        }
        results.Add(result);
    }
}

Console.WriteLine(results.ToJsonString());
return 0;

static bool IsKnownOperation(string subject, string op)
{
    return (subject, op) switch
    {
        ("f8n.Percentage", "fromPercent") => true,
        ("f8n.Percentage", "fromProportion") => true,
        ("f8n.Percentage", "parse") => true,
        ("f8n.LocalDate", "parse") => true,
        ("f8n.EffectiveDated", "asOf") => true,
        ("f8n.Percentage", "property.percentMatchesProportion") => true,
        ("f8n.Percentage", "property.roundTrip") => true,
        ("f8n.LocalDate", "property.roundTrip") => true,
        ("f8n.LocalDate", "property.orderIsTotal") => true,
        ("f8n.EffectiveDated", "property.asOfAtEachBoundary") => true,
        ("f8n.EffectiveDated", "property.orderIndependence") => true,
        ("f8n.Rounding", "divide") => true,
        ("f8n.Rounding", "property.signSymmetric") => true,
        ("f8n.Money", "fromMajor") => true,
        ("f8n.Money", "fromMinor") => true,
        ("f8n.Money", "multiplyByRate") => true,
        ("f8n.Money", "divideBy") => true,
        ("f8n.Money", "property.minorMatchesMajor") => true,
        ("f8n.Money", "property.addSubtractIsExact") => true,
        ("f8n.Country", "find") => true,
        ("f8n.Country", "property.formsAgree") => true,
        ("f8n.CultureCanary", "upperCaseI") => true,
        _ => false,
    };
}

static string Execute(string subject, string op, List<string> inputs)
{
    switch (subject, op)
    {
        case ("f8n.Percentage", "fromPercent"):
            return Percentage.FromPercent(inputs[0]).ToString();
        case ("f8n.Percentage", "fromProportion"):
            return Percentage.FromProportion(inputs[0]).ToString();
        case ("f8n.Percentage", "parse"):
            return Percentage.Parse(inputs[0]).ToString();
        case ("f8n.LocalDate", "parse"):
            return LocalDate.Parse(inputs[0]).ToString();
        case ("f8n.EffectiveDated", "asOf"):
            return Fixture.AsOf(LocalDate.Parse(inputs[0]));
        case ("f8n.Percentage", "property.percentMatchesProportion"):
            return Property.PercentMatchesProportion(inputs[0], inputs[1]);
        case ("f8n.Percentage", "property.roundTrip"):
            return Property.PercentageRoundTrip(inputs[0]);
        case ("f8n.LocalDate", "property.roundTrip"):
            return Property.DateRoundTrip(inputs[0]);
        case ("f8n.LocalDate", "property.orderIsTotal"):
            return Property.OrderIsTotal(inputs);
        case ("f8n.EffectiveDated", "property.asOfAtEachBoundary"):
            return Fixture.AsOfAtEachBoundary();
        case ("f8n.EffectiveDated", "property.orderIndependence"):
            return Fixture.OrderIndependence();
        case ("f8n.Rounding", "divide"):
            return Rounding.DivideWithRounding(
                BigInteger.Parse(inputs[0], CultureInfo.InvariantCulture),
                BigInteger.Parse(inputs[1], CultureInfo.InvariantCulture),
                Modes.Parse(inputs[2])).ToString(CultureInfo.InvariantCulture);
        case ("f8n.Rounding", "property.signSymmetric"):
            return Property.SignSymmetric(inputs[0], inputs[1], inputs[2]);
        case ("f8n.Money", "fromMajor"):
            return Money.FromMajor(inputs[0], Currencies.Lookup(inputs[1])).Amount;
        case ("f8n.Money", "fromMinor"):
            return Money.FromMinor(BigInteger.Parse(inputs[0], CultureInfo.InvariantCulture), Currencies.Lookup(inputs[1])).Amount;
        case ("f8n.Money", "multiplyByRate"):
            return Money.FromMajor(inputs[0], Currencies.Lookup(inputs[1]))
                .Multiply(Percentage.FromPercent(inputs[2]), Modes.Parse(inputs[3])).Amount;
        case ("f8n.Money", "divideBy"):
            return Money.FromMajor(inputs[0], Currencies.Lookup(inputs[1]))
                .Divide(BigInteger.Parse(inputs[2], CultureInfo.InvariantCulture), Modes.Parse(inputs[3])).Amount;
        case ("f8n.Money", "property.minorMatchesMajor"):
            return Property.MinorMatchesMajor(inputs[0], inputs[1], inputs[2]);
        case ("f8n.Money", "property.addSubtractIsExact"):
            return Property.AddSubtractIsExact(inputs[0], inputs[1], inputs[2]);
        case ("f8n.Country", "find"):
            return Country.Find(inputs[0])?.Alpha3 ?? "(none)";
        case ("f8n.Country", "property.formsAgree"):
            return Property.FormsAgree(inputs[0], inputs[1], inputs[2]);
        // Deliberately culture-SENSITIVE, and the only thing in f8n that is. It exists to
        // prove the hostile-locale CI step is actually hostile; everything else uses the
        // invariant form precisely so it is not.
        case ("f8n.CultureCanary", "upperCaseI"):
            return "i".ToUpper();
        default:
            throw new InvalidOperationException($"unknown op {op} for {subject}");
    }
}

// A series the vectors own, so they pin the lookup's semantics rather than f8n's tax data —
// which will grow, and would take the expected values with it. Deliberately written oldest
// first, since the type sorts its own entries and must not depend on authoring order.
static class Fixture
{
    private static readonly (LocalDate From, string Value)[] Entries =
    {
        (LocalDate.Parse("2010-01-01"), "A"),
        (LocalDate.Parse("2011-01-04"), "B"),
    };

    private static readonly EffectiveDated<string> Series = EffectiveDated.Of(Entries);

    // Probe dates spanning the series: before it, on each boundary, between, and after.
    private static readonly string[] Probes =
    {
        "2009-12-31", "2010-01-01", "2010-06-15", "2011-01-03", "2011-01-04", "2011-01-05", "9999-12-31",
    };

    // A day the series does not cover is a defined outcome, not a rejected input, so it
    // reports a value of its own rather than throwing — an error would put it in the same
    // bucket as a malformed date, which is a different thing entirely.
    public static string AsOf(LocalDate on)
    {
        return Series.TryAsOf(on, out var value) ? value : "(none)";
    }

    // Every entry is in effect on the day it takes effect. Trivially true to state and the
    // exact thing an off-by-one in the comparison breaks, in one language only.
    public static string AsOfAtEachBoundary()
    {
        foreach (var (from, expected) in Entries)
        {
            if (!Series.TryAsOf(from, out var actual))
            {
                return $"no value at {from}, which is an entry's own effective date";
            }
            if (actual != expected)
            {
                return $"at {from} the series gave \"{actual}\", not the entry's own \"{expected}\"";
            }
        }
        return "true";
    }

    // The type sorts its own entries, so the order a data file happened to be written in
    // cannot change an answer. Asserted rather than assumed: nothing else tests it, and the
    // failure it prevents is a data file rearranged for readability changing a rate.
    public static string OrderIndependence()
    {
        var reversed = Entries.Reverse().ToArray();
        var other = EffectiveDated.Of(reversed);
        foreach (var probe in Probes)
        {
            var date = LocalDate.Parse(probe);
            var first = Series.TryAsOf(date, out var a) ? a : "(none)";
            var second = other.TryAsOf(date, out var b) ? b : "(none)";
            if (first != second)
            {
                return $"at {probe} the two orderings gave \"{first}\" and \"{second}\"";
            }
        }
        return "true";
    }
}

// Properties: invariants that hold for every input, so the expected value is "true" and
// comes from the rule rather than from any implementation. That is what makes them the
// independent derivation path — a value captured from one language and blessed by another
// written against the capture would still satisfy the dataset, and would not satisfy these.
static class Property
{
    // 17.5% and the proportion 0.175 are the same value. The pairs are derived by dividing
    // by a hundred, not read off an implementation.
    public static string PercentMatchesProportion(string percent, string proportion)
    {
        var fromPercent = Percentage.FromPercent(percent);
        var fromProportion = Percentage.FromProportion(proportion);
        if (!fromPercent.Equals(fromProportion))
        {
            return $"{percent}% is {fromPercent}, but the proportion {proportion} is {fromProportion}";
        }
        return "true";
    }

    // parse(canonical(x)) == x. The canonical form is unique per value, so a value that
    // survives a round trip has exactly one encoding — which is what makes wire equality
    // mean value equality.
    public static string PercentageRoundTrip(string percent)
    {
        var canonical = Percentage.FromPercent(percent).ToString();
        var reparsed = Percentage.Parse(canonical).ToString();
        if (reparsed != canonical)
        {
            return $"{percent}% canonicalises to {canonical} but reparses to {reparsed}";
        }
        return "true";
    }

    public static string DateRoundTrip(string text)
    {
        var canonical = LocalDate.Parse(text).ToString();
        var reparsed = LocalDate.Parse(canonical).ToString();
        if (reparsed != canonical)
        {
            return $"{text} canonicalises to {canonical} but reparses to {reparsed}";
        }
        return "true";
    }

    // A comparison is a total order: irreflexive-on-equal, antisymmetric, and transitive.
    // The dates arrive in ascending order and every pair and triple is checked.
    // divide(-n, d) == -divide(n, d). The property the HalfUp decision was made to preserve:
    // away-from-zero keeps it, toward-positive-infinity does not.
    public static string SignSymmetric(string numerator, string denominator, string mode)
    {
        var n = BigInteger.Parse(numerator, CultureInfo.InvariantCulture);
        var d = BigInteger.Parse(denominator, CultureInfo.InvariantCulture);
        var m = Modes.Parse(mode);

        var positive = Rounding.DivideWithRounding(n, d, m);
        var negated = Rounding.DivideWithRounding(-n, d, m);
        if (negated != -positive)
        {
            return $"{numerator}/{denominator} gave {positive} but -{numerator}/{denominator} gave {negated}";
        }
        return "true";
    }

    // The two constructors name different units for the same value, so they must agree.
    public static string MinorMatchesMajor(string minor, string major, string code)
    {
        var currency = Currencies.Lookup(code);
        var fromMinor = Money.FromMinor(BigInteger.Parse(minor, CultureInfo.InvariantCulture), currency);
        var fromMajor = Money.FromMajor(major, currency);
        if (!fromMinor.Equals(fromMajor))
        {
            return $"{minor} minor units is {fromMinor.Amount}, but the major form {major} is {fromMajor.Amount}";
        }
        return "true";
    }

    // (a + b) - b == a. Addition is integer work on minor units, so it is exact by
    // construction — and this is what says so rather than assuming it.
    public static string AddSubtractIsExact(string first, string second, string code)
    {
        var currency = Currencies.Lookup(code);
        var a = Money.FromMajor(first, currency);
        var b = Money.FromMajor(second, currency);
        var roundTripped = a.Add(b).Subtract(b);
        if (!roundTripped.Equals(a))
        {
            return $"({first} + {second}) - {second} gave {roundTripped.Amount}, not {first}";
        }
        return "true";
    }

    // Every form of one country finds the same row. This is what makes accepting three forms
    // a convenience rather than three different answers.
    public static string FormsAgree(string alpha2, string alpha3, string numeric)
    {
        var byTwo = Country.Find(alpha2);
        var byThree = Country.Find(alpha3);
        var byNumber = Country.Find(numeric);
        if (byTwo is null || byThree is null || byNumber is null)
        {
            return $"one of {alpha2}/{alpha3}/{numeric} found nothing";
        }
        if (byTwo.Alpha3 != byThree.Alpha3 || byTwo.Alpha3 != byNumber.Alpha3)
        {
            return $"{alpha2} found {byTwo.Alpha3}, {alpha3} found {byThree.Alpha3}, {numeric} found {byNumber.Alpha3}";
        }
        return "true";
    }

    public static string OrderIsTotal(List<string> ascending)
    {
        var dates = ascending.Select(LocalDate.Parse).ToArray();
        for (var i = 0; i < dates.Length; i++)
        {
            if (dates[i].CompareTo(dates[i]) != 0)
            {
                return $"{dates[i]} does not compare equal to itself";
            }
            for (var j = i + 1; j < dates.Length; j++)
            {
                if (dates[i].CompareTo(dates[j]) >= 0)
                {
                    return $"{dates[i]} does not sort before {dates[j]}";
                }
                if (dates[j].CompareTo(dates[i]) <= 0)
                {
                    return $"{dates[j]} does not sort after {dates[i]}";
                }
            }
        }
        return "true";
    }
}

// The vectors name a currency by its code, and the currencies are generated — so this is
// also the first place a vector run reaches c5n's output rather than only hand-written code.
static class Currencies
{
    public static Currency Lookup(string code)
    {
        return code switch
        {
            "GBP" => Currency.GBP,
            "EUR" => Currency.EUR,
            "USD" => Currency.USD,
            "JPY" => Currency.JPY,
            "BHD" => Currency.BHD,
            _ => throw new InvalidOperationException($"unknown currency {code}"),
        };
    }
}

static class Modes
{
    public static RoundingMode Parse(string name)
    {
        return name switch
        {
            "HalfEven" => RoundingMode.HalfEven,
            "HalfUp" => RoundingMode.HalfUp,
            _ => throw new InvalidOperationException($"unknown rounding mode {name}"),
        };
    }
}
