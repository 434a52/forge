// run-vector — executes a vector file and reports what this implementation produced.
//
// Deliberately thin: it makes no assertions and uses no test framework. Comparison belongs
// to the driver, so the same driver can be pointed at somebody else's implementation and
// audit it the same way. This one reads the cases and ignores any expected values in the
// file, which is what keeps it from grading its own work.

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
if (subject is not ("f8n.Percentage" or "f8n.LocalDate" or "f8n.EffectiveDated"))
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
        default:
            throw new InvalidOperationException($"unknown op {op} for {subject}");
    }
}

// A series the vectors own, so they pin the lookup's semantics rather than f8n's tax data —
// which will grow, and would take the expected values with it. Deliberately written oldest
// first, since the type sorts its own entries and must not depend on authoring order.
static class Fixture
{
    private static readonly EffectiveDated<string> Series = EffectiveDated.Of(
        (LocalDate.Parse("2010-01-01"), "A"),
        (LocalDate.Parse("2011-01-04"), "B"));

    // A day the series does not cover is a defined outcome, not a rejected input, so it
    // reports a value of its own rather than throwing — an error would put it in the same
    // bucket as a malformed date, which is a different thing entirely.
    public static string AsOf(LocalDate on)
    {
        return Series.TryAsOf(on, out var value) ? value : "(none)";
    }
}
