/**
 * run-vector — executes a vector file and reports what this implementation produced.
 *
 * Deliberately thin: it makes no assertions and uses no test framework. Comparison belongs
 * to the driver, so the same driver can be pointed at somebody else's implementation and
 * audit it the same way. This one reads the cases and ignores any expected values in the
 * file, which is what keeps it from grading its own work.
 */

import { readFileSync } from "node:fs";
import { EffectiveDated } from "./effectivedated.js";
import { LocalDate } from "./localdate.js";
import { Percentage } from "./percentage.js";

interface VectorCase {
  id: string;
  op: string;
  in: string[];
}

interface VectorGroup {
  cases: VectorCase[];
}

interface VectorFile {
  subject: string;
  groups: VectorGroup[];
}

interface CaseResult {
  id: string;
  out?: string;
  error?: string;
}

// The op alone is not enough to dispatch: "parse" means one thing for a Percentage and
// another for a LocalDate. The subject names which type's vectors these are, so one runner
// per language covers every f8n subject rather than one binary per type.
const knownOperations = new Map<string, Set<string>>([
  ["f8n.Percentage", new Set(["fromPercent", "fromProportion", "parse"])],
  ["f8n.LocalDate", new Set(["parse"])],
  ["f8n.EffectiveDated", new Set(["asOf"])],
]);

// A series the vectors own, so they pin the lookup's semantics rather than f8n's tax data —
// which will grow, and would take the expected values with it. Deliberately written oldest
// first, since the type sorts its own entries and must not depend on authoring order.
const fixture = EffectiveDated.of<string>([
  [LocalDate.parse("2010-01-01"), "A"],
  [LocalDate.parse("2011-01-04"), "B"],
]);

function execute(subject: string, op: string, inputs: string[]): string {
  switch (`${subject}.${op}`) {
    case "f8n.Percentage.fromPercent":
      return Percentage.fromPercent(inputs[0]).toString();
    case "f8n.Percentage.fromProportion":
      return Percentage.fromProportion(inputs[0]).toString();
    case "f8n.Percentage.parse":
      return Percentage.parse(inputs[0]).toString();
    case "f8n.LocalDate.parse":
      return LocalDate.parse(inputs[0]).toString();
    // A day the series does not cover is a defined outcome, not a rejected input, so it
    // reports a value of its own rather than throwing — an error would put it in the same
    // bucket as a malformed date, which is a different thing entirely.
    case "f8n.EffectiveDated.asOf":
      return fixture.asOf(LocalDate.parse(inputs[0])) ?? "(none)";
    default:
      throw new Error(`unknown op ${op} for ${subject}`);
  }
}

function main(): number {
  const path = process.argv[2];
  if (path === undefined) {
    process.stderr.write("usage: run-vector <vectors.json>\n");
    return 1;
  }

  let document: VectorFile;
  try {
    document = JSON.parse(readFileSync(path, "utf8")) as VectorFile;
  } catch (cause) {
    process.stderr.write(`run-vector: cannot read ${path}: ${String(cause)}\n`);
    return 1;
  }

  const operations = knownOperations.get(document.subject);
  if (operations === undefined) {
    process.stderr.write(`run-vector: unknown subject "${document.subject}"\n`);
    return 2;
  }

  const results: CaseResult[] = [];
  for (const group of document.groups) {
    for (const testCase of group.cases) {
      // An unknown op is a fault in the harness, not a failing case — reporting it as a
      // case error would let a reject case "pass" for entirely the wrong reason.
      if (!operations.has(testCase.op)) {
        process.stderr.write(`run-vector: ${testCase.id}: unknown op "${testCase.op}"\n`);
        return 2;
      }
      try {
        results.push({ id: testCase.id, out: execute(document.subject, testCase.op, testCase.in) });
      } catch (cause) {
        results.push({ id: testCase.id, error: cause instanceof Error ? cause.message : String(cause) });
      }
    }
  }

  process.stdout.write(JSON.stringify(results) + "\n");
  return 0;
}

process.exitCode = main();
