/**
 * run-vector — executes a vector file and reports what this implementation produced.
 *
 * Deliberately thin: it makes no assertions and uses no test framework. Comparison belongs
 * to the driver, so the same driver can be pointed at somebody else's implementation and
 * audit it the same way. This one reads the cases and ignores any expected values in the
 * file, which is what keeps it from grading its own work.
 */

import { readFileSync } from "node:fs";
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
  groups: VectorGroup[];
}

interface CaseResult {
  id: string;
  out?: string;
  error?: string;
}

const knownOperations = new Set(["fromPercent", "fromProportion", "parse"]);

function execute(op: string, inputs: string[]): string {
  switch (op) {
    case "fromPercent":
      return Percentage.fromPercent(inputs[0]).toString();
    case "fromProportion":
      return Percentage.fromProportion(inputs[0]).toString();
    case "parse":
      return Percentage.parse(inputs[0]).toString();
    default:
      throw new Error(`unknown op ${op}`);
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

  const results: CaseResult[] = [];
  for (const group of document.groups) {
    for (const testCase of group.cases) {
      // An unknown op is a fault in the harness, not a failing case — reporting it as a
      // case error would let a reject case "pass" for entirely the wrong reason.
      if (!knownOperations.has(testCase.op)) {
        process.stderr.write(`run-vector: ${testCase.id}: unknown op "${testCase.op}"\n`);
        return 2;
      }
      try {
        results.push({ id: testCase.id, out: execute(testCase.op, testCase.in) });
      } catch (cause) {
        results.push({ id: testCase.id, error: cause instanceof Error ? cause.message : String(cause) });
      }
    }
  }

  process.stdout.write(JSON.stringify(results) + "\n");
  return 0;
}

process.exitCode = main();
