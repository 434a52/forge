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
  ["f8n.Percentage", new Set([
    "fromPercent", "fromProportion", "parse",
    "property.percentMatchesProportion", "property.roundTrip",
  ])],
  ["f8n.LocalDate", new Set(["parse", "property.roundTrip", "property.orderIsTotal"])],
  ["f8n.EffectiveDated", new Set([
    "asOf", "property.asOfAtEachBoundary", "property.orderIndependence",
  ])],
]);

// A series the vectors own, so they pin the lookup's semantics rather than f8n's tax data —
// which will grow, and would take the expected values with it. Deliberately written oldest
// first, since the type sorts its own entries and must not depend on authoring order.
const fixtureEntries: [LocalDate, string][] = [
  [LocalDate.parse("2010-01-01"), "A"],
  [LocalDate.parse("2011-01-04"), "B"],
];

const fixture = EffectiveDated.of<string>(fixtureEntries);

/** Probe dates spanning the series: before it, on each boundary, between, and after. */
const probes = [
  "2009-12-31", "2010-01-01", "2010-06-15", "2011-01-03", "2011-01-04", "2011-01-05", "9999-12-31",
];

// Properties: invariants that hold for every input, so the expected value is "true" and
// comes from the rule rather than from any implementation. That is what makes them the
// independent derivation path — a value captured from one language and blessed by another
// written against the capture would still satisfy the dataset, and would not satisfy these.

/** 17.5% and the proportion 0.175 are the same value; the pairs are derived by dividing by a hundred. */
function percentMatchesProportion(percent: string, proportion: string): string {
  const fromPercent = Percentage.fromPercent(percent);
  const fromProportion = Percentage.fromProportion(proportion);
  if (fromPercent.toString() !== fromProportion.toString()) {
    return `${percent}% is ${fromPercent.toString()}, but the proportion ${proportion} is ${fromProportion.toString()}`;
  }
  return "true";
}

/**
 * parse(canonical(x)) == x. The canonical form is unique per value, so a value that survives
 * a round trip has exactly one encoding — which is what makes wire equality mean value
 * equality.
 */
function percentageRoundTrip(percent: string): string {
  const canonical = Percentage.fromPercent(percent).toString();
  const reparsed = Percentage.parse(canonical).toString();
  if (reparsed !== canonical) {
    return `${percent}% canonicalises to ${canonical} but reparses to ${reparsed}`;
  }
  return "true";
}

function dateRoundTrip(text: string): string {
  const canonical = LocalDate.parse(text).toString();
  const reparsed = LocalDate.parse(canonical).toString();
  if (reparsed !== canonical) {
    return `${text} canonicalises to ${canonical} but reparses to ${reparsed}`;
  }
  return "true";
}

/**
 * A comparison is a total order: equal to itself, antisymmetric, and transitive. The dates
 * arrive in ascending order and every pair is checked in both directions.
 */
function orderIsTotal(ascending: string[]): string {
  const dates = ascending.map((text) => LocalDate.parse(text));
  for (let i = 0; i < dates.length; i++) {
    if (dates[i]!.compareTo(dates[i]!) !== 0) {
      return `${dates[i]!.toString()} does not compare equal to itself`;
    }
    for (let j = i + 1; j < dates.length; j++) {
      if (dates[i]!.compareTo(dates[j]!) >= 0) {
        return `${dates[i]!.toString()} does not sort before ${dates[j]!.toString()}`;
      }
      if (dates[j]!.compareTo(dates[i]!) <= 0) {
        return `${dates[j]!.toString()} does not sort after ${dates[i]!.toString()}`;
      }
    }
  }
  return "true";
}

/** Every entry is in effect on the day it takes effect. */
function asOfAtEachBoundary(): string {
  for (const [from, expected] of fixtureEntries) {
    const actual = fixture.asOf(from);
    if (actual === undefined) {
      return `no value at ${from.toString()}, which is an entry's own effective date`;
    }
    if (actual !== expected) {
      return `at ${from.toString()} the series gave "${actual}", not the entry's own "${expected}"`;
    }
  }
  return "true";
}

/**
 * The type sorts its own entries, so the order a data file happened to be written in cannot
 * change an answer. Asserted rather than assumed: nothing else tests it, and the failure it
 * prevents is a data file rearranged for readability changing a rate.
 */
function orderIndependence(): string {
  const other = EffectiveDated.of<string>([...fixtureEntries].reverse());
  for (const probe of probes) {
    const date = LocalDate.parse(probe);
    const first = fixture.asOf(date) ?? "(none)";
    const second = other.asOf(date) ?? "(none)";
    if (first !== second) {
      return `at ${probe} the two orderings gave "${first}" and "${second}"`;
    }
  }
  return "true";
}

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
    case "f8n.Percentage.property.percentMatchesProportion":
      return percentMatchesProportion(inputs[0], inputs[1]);
    case "f8n.Percentage.property.roundTrip":
      return percentageRoundTrip(inputs[0]);
    case "f8n.LocalDate.property.roundTrip":
      return dateRoundTrip(inputs[0]);
    case "f8n.LocalDate.property.orderIsTotal":
      return orderIsTotal(inputs);
    case "f8n.EffectiveDated.property.asOfAtEachBoundary":
      return asOfAtEachBoundary();
    case "f8n.EffectiveDated.property.orderIndependence":
      return orderIndependence();
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
