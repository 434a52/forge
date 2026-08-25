# c5n

The codegen engine (**Go**). Reads a typed schema + data (YAML) and emits typed, cross-language-conformant code (C#, TS; Swift/… pluggable). Generates the **data + a thin typed boundary**; consumers hand-write the behaviour.

Design: `./DESIGN.md`. Plan: `./PLAN.md`.

**Status: early.** `c5n build` and `c5n check` work for keyed tables (`table<T>`), emitting
C# and TypeScript; the first consumer's data is generated from source, committed, and
guarded against drift. Enums, nested construction and temporal series are not built yet —
the plan has the order.
