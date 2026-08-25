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

var results = new JsonArray();

foreach (JsonNode? group in document["groups"]!.AsArray())
{
    foreach (JsonNode? testCase in group!["cases"]!.AsArray())
    {
        string id = testCase!["id"]!.GetValue<string>();
        string op = testCase["op"]!.GetValue<string>();

        var inputs = new List<string>();
        foreach (JsonNode? input in testCase["in"]!.AsArray())
        {
            inputs.Add(input!.GetValue<string>());
        }

        // An unknown op is a fault in the harness, not a failing case — reporting it as a
        // case error would let a reject case "pass" for entirely the wrong reason.
        if (!IsKnownOperation(op))
        {
            Console.Error.WriteLine($"run-vector: {id}: unknown op \"{op}\"");
            return 2;
        }

        var result = new JsonObject { ["id"] = id };
        try
        {
            result["out"] = Execute(op, inputs);
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

static bool IsKnownOperation(string op)
{
    return op is "fromPercent" or "fromProportion" or "parse";
}

static string Execute(string op, List<string> inputs)
{
    switch (op)
    {
        case "fromPercent":
            return Percentage.FromPercent(inputs[0]).ToString();
        case "fromProportion":
            return Percentage.FromProportion(inputs[0]).ToString();
        case "parse":
            return Percentage.Parse(inputs[0]).ToString();
        default:
            throw new InvalidOperationException($"unknown op {op}");
    }
}
