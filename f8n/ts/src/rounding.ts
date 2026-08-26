/**
 * How a precision-losing operation resolves a value that falls between two results.
 *
 * Caller-supplied at every precision-losing operation rather than defaulted per call site: a
 * jurisdiction can mandate a method, and a silently chosen one is the kind of non-compliance
 * nothing later detects. `allocate` takes no mode at all — it conserves via an
 * `AllocationRule` instead. See f8n/DESIGN.md.
 */
export const RoundingMode = {
  /** Banker's rounding: an exact half goes to the even result. The default elsewhere in f8n. */
  HalfEven: "HalfEven",
  /** An exact half goes away from zero — so -2.5 rounds to -3, not -2. */
  HalfUp: "HalfUp",
} as const;

export type RoundingMode = (typeof RoundingMode)[keyof typeof RoundingMode];

/**
 * The one function the cross-language conformance claim rests on.
 *
 * Everything else in f8n's arithmetic is exact integer work on big integers, which cannot
 * diverge between languages. Division is the single place a result must be *chosen*, so it is
 * the surface the golden vectors target — and it is written once, here, rather than inlined
 * at each call site.
 *
 * `Money × Percentage` reaches it with the rational's denominator in place of a power of ten,
 * which is why making `Percentage` exact added no new conformance surface.
 */
export function divideWithRounding(
  numerator: bigint,
  denominator: bigint,
  mode: RoundingMode,
): bigint {
  if (denominator === 0n) {
    throw new Error("a denominator of zero has no quotient");
  }
  // Normalise the sign onto the numerator, so the true quotient's sign is the numerator's
  // and the comparisons below need only one case.
  if (denominator < 0n) {
    numerator = -numerator;
    denominator = -denominator;
  }

  // Both languages truncate toward zero on integer division, so the truncated quotient and
  // the remainder's sign are the same in each without any adjustment.
  const truncated = numerator / denominator;
  const remainder = numerator - truncated * denominator;
  if (remainder === 0n) {
    return truncated;
  }

  // Compare twice the remainder against the divisor rather than halving anything: halving
  // would be the one place a division could reintroduce the problem.
  const twiceRemainder = (remainder < 0n ? -remainder : remainder) * 2n;
  const awayFromZero = numerator < 0n ? truncated - 1n : truncated + 1n;

  if (twiceRemainder > denominator) {
    return awayFromZero;
  }
  if (twiceRemainder < denominator) {
    return truncated;
  }

  // Exactly half.
  switch (mode) {
    case RoundingMode.HalfUp:
      return awayFromZero;
    // To even: keep the truncated quotient when it is already even, else step away from
    // zero. Sign-symmetric, like HalfUp — one answer for negatives library-wide.
    case RoundingMode.HalfEven:
      return truncated % 2n === 0n ? truncated : awayFromZero;
    default:
      throw new Error(`unknown rounding mode ${String(mode)}`);
  }
}
