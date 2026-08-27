/**
 * How the units left over by an exact partition are handed out.
 *
 * Parameterised and with **no default**, unlike `RoundingMode`. A jurisdiction or a contract
 * can mandate an apportionment method, and a silently chosen one is non-compliance that
 * nothing later detects — so the caller must say which they mean.
 *
 * A closed set of built-ins, deliberately. A caller-supplied distributor would move the "both
 * languages identical" guarantee into the caller's own two callbacks, which f8n cannot test
 * and therefore should not promise. A genuinely new mandated method arrives here as a named
 * built-in with its own vectors.
 */
export type AllocationRule =
  | { readonly method: "LargestRemainder" }
  | { readonly method: "Sequential" }
  | { readonly method: "Designated"; readonly part: number };

export const AllocationRule = {
  /**
   * Hamilton: leftover units go to the largest fractional remainders, ties by ascending
   * index. The usual choice where no method is mandated.
   */
  largestRemainder: { method: "LargestRemainder" } as AllocationRule,

  /** The first parts absorb the leftover, one unit each. */
  sequential: { method: "Sequential" } as AllocationRule,

  /** One nominated part — a designated ledger line — absorbs all of the residual. */
  designated(part: number): AllocationRule {
    if (!Number.isInteger(part) || part < 0) {
      throw new Error("a part index is a non-negative integer");
    }
    return { method: "Designated", part };
  },
} as const;

/**
 * A conserving partition of an integer quantity — the second conformance surface, after
 * `divideWithRounding`.
 *
 * **Not rounding.** Rounding each part independently does not conserve: £100 split three ways
 * at 2 dp gives 33.33 three times, and a penny disappears. This takes the exact share of each
 * part, keeps the whole units, and then hands out the units left over — so the parts sum to
 * the whole exactly, by construction rather than by checking.
 *
 * **Sign is handled by working on the magnitude.** The design says the shares are "floored",
 * which is right for a non-negative total and wrong for a negative one: flooring -1.5 gives -2
 * where negating the floor of 1.5 gives -1. Since `allocate(-m) == -allocate(m)` is a stated
 * requirement, the sign is lifted out first and reapplied at the end, which makes the symmetry
 * true by construction instead of a property that has to hold.
 */
export function distribute(total: bigint, weights: readonly bigint[], rule: AllocationRule): bigint[] {
  if (weights.length === 0) {
    throw new Error("a partition needs at least one part");
  }

  let weightTotal = 0n;
  for (const weight of weights) {
    if (weight < 0n) {
      throw new Error("a weight is not negative");
    }
    weightTotal += weight;
  }
  if (weightTotal === 0n) {
    throw new Error("the weights are all zero, so there is nothing to be proportional to");
  }
  if (rule.method === "Designated" && rule.part >= weights.length) {
    throw new Error(`there is no part ${rule.part} in a partition of ${weights.length}`);
  }

  const negative = total < 0n;
  const magnitude = negative ? -total : total;

  // Exact share of part i is magnitude * wᵢ / weightTotal. Both operands are non-negative
  // here, so integer division is a floor and the remainder is the fractional part's numerator
  // — which is what the largest-remainder rule ranks on.
  const parts: bigint[] = [];
  const remainders: bigint[] = [];
  let allocated = 0n;
  for (const weight of weights) {
    const scaled = magnitude * weight;
    const share = scaled / weightTotal;
    parts.push(share);
    remainders.push(scaled - share * weightTotal);
    allocated += share;
  }

  const leftover = magnitude - allocated;
  for (const i of recipients(leftover, remainders, rule)) {
    parts[i] = parts[i]! + 1n;
  }

  return negative ? parts.map((part) => -part) : parts;
}

/** Which parts receive one of the leftover units, in the rule's order. */
function recipients(leftover: bigint, remainders: readonly bigint[], rule: AllocationRule): number[] {
  const count = Number(leftover);
  const chosen: number[] = [];

  if (rule.method === "Designated") {
    // One line absorbs all of it, so this repeats the same index.
    for (let n = 0; n < count; n++) {
      chosen.push(rule.part);
    }
    return chosen;
  }

  if (rule.method === "Sequential") {
    for (let i = 0; i < count; i++) {
      chosen.push(i);
    }
    return chosen;
  }

  // Hamilton. Ordering by remainder descending with ties by ascending index is what makes
  // this deterministic — and the tie-break is not a detail: an equal split gives every part
  // the same remainder, so ties are the normal case rather than the edge one, and without a
  // stated order the two languages could disagree on every three-way split of an odd amount.
  const order = remainders.map((_, i) => i);
  order.sort((left, right) => {
    const leftRemainder = remainders[left]!;
    const rightRemainder = remainders[right]!;
    if (leftRemainder !== rightRemainder) {
      return rightRemainder > leftRemainder ? 1 : -1;
    }
    return left - right;
  });
  for (let i = 0; i < count; i++) {
    chosen.push(order[i]!);
  }
  return chosen;
}
