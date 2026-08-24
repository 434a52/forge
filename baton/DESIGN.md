# baton — design

A **shared document store for humans and agents editing the same corpus**. Documents live on a server; readers never block; writers hold a lease. Named for the mechanism — you hold the baton or you don't, passing it is the whole protocol, and dropping it is a lease expiring.

**Status: designed, not started. Queued behind `ampersand`** — captured here so the reasoning survives, not because it's in the build.

## The problem

Two people, each with their own coding agent, working on one corpus. Four writers, two of them autonomous, on separate machines.

- **Git gives distribution without coordination.** Three-way text merge is built for code that gets reviewed before it lands; applied to prose being edited concurrently by a person and an agent it produces plausible-looking wrong answers rather than conflicts anyone notices.
- **Agents don't back off politely.** Two can be mid-edit on the same file with no awareness of each other.
- **Whole-file writers defeat CRDTs.** Conflict-free replicated types need *operations*; an editor or an agent writing a file emits a before-and-after, so a sync layer has to infer operations by diffing — at which point the CRDT has bought nothing.
- **File-sync services are last-write-wins** under the covers, which is the failure this exists to prevent.

**Contention is bimodal, and that decides the design.** Working separately, collisions are rare — different areas, different times. Working *together* — on a call, discussing the same thing — collisions are the norm, because the same discussion means the same documents. A design tuned for "rare contention" fails precisely when the shared store is earning its keep.

## Architecture

**Server-authoritative.** Documents are rows; markdown lives in a column. Git-as-transport makes every read a fetch and every write a push-and-hope, and the lock has to live with the text it protects.

One service holds the store, the MCP server, the review surface and the comment worker, over one database.

### Concurrency — pessimistic, deliberately

The usual anti-locking orthodoxy comes from high-contention, high-throughput database work. **This is two people editing prose**: contention is rare-to-bursty and each edit is long-lived, which is the regime locking suits.

It also fits agents better than optimistic concurrency does. *"You can't have it, wait"* is trivial for an agent to handle; *"you lost, re-read and re-apply"* means re-reasoning about content that changed underneath it — expensive, and the point at which an agent starts inventing a merge.

**Two mechanisms for two operations:**

- **Reads never block, and return a version.** Non-negotiable — an agent has to read widely to answer anything. The version is how a writer learns that something it reasoned over has since moved.
- **Writes take a lease**, acquired on *intent to edit*, before composing rather than at commit. Composing first and locking at commit converges on optimistic-with-a-mutex and throws away expensive agent work when it loses.

**Rules that decide whether locking survives contact:**

| | |
|---|---|
| **Leases, not locks** | Auto-expire after N minutes. One crashed process must not block a document forever — that's the failure everyone remembers. |
| **Visibility** | Who holds what, and since when. A human blocked by an invisible lock gets angry; one who can see *"held by another session, 3 minutes ago"* just waits. |
| **Human force-break** | Sometimes necessary. Without it people edit around the system and the invariant is lost anyway. |
| **Specified agent behaviour on refusal** | Report and pick up something else, or stop. Left unstated an agent retries in a tight loop. |
| **Escalation** | Need a document that wasn't in the original set: try to acquire, **short timeout, then release everything and retry or report**. |
| **Canonical ordering** | Acquire in a fixed order (alphabetical path) to make deadlock impossible rather than merely time it out. |

**Identity is a first-class concern.** Locks belong to a **person**, not to a service — otherwise everyone sees *"held by the doc API"* and two agents share one lock identity. Auth propagates from the human, through their agent, to the store.

### MCP — one server, several clients

The store exposes an **MCP server**. Both people's coding agents and the comment worker point at the same endpoint and therefore see the same locks, versions and documents.

**Multi-agent coordination then falls out rather than being designed.** Two agents sharing a locked, versioned store *are* coordinating: each takes leases, each sees what the other holds, and a shared plan is just another document they both read.

### Review surface

Rendered markdown, served live from the store.

**The edit affordance *is* the lock acquisition.** Clicking "edit" switches the rendered view to a markdown editor and takes the lease; leaving releases it. The UI action and the concurrency primitive are the same event, so there is no separate concept to explain.

This is **an application, not a static site generator** — live documents with leases cannot be a static build. Different lifecycle, different hosting.

### Comments — how agents are addressed

Rather than a chat surface: **threads anchored to documents, with `@claude` in a comment.**

- **Anchored** — *"this section"* is unambiguous, so no context has to be pasted or described.
- **Shared by construction** — both people see the question and the answer, in place, permanently. A chat is inherently one person's.
- **Asynchronous** — post, a worker picks it up, a reply appears. No streaming, no live session, no conversation state machine. A queue and one API call.
- **Works from a phone** — posting a comment is a text box, which recovers question-asking and capture without a desktop session.

**Two rules:**

**Anchor to headings in v1.** Range anchoring against an edited document is the genuinely hard problem (fuzzy re-matching, as Google Docs and Hypothesis do). Markdown gives natural section structure; a renamed heading orphans its thread, and that is an acceptable v1 cost.

**Claude proposes; a human commits.** The worker replies in the thread with suggested text — it never takes a lease and edits. The agent stays out of the write path entirely, and a bad suggestion costs nothing.

## Explicitly out of scope

Written down because the scope inflated from a lock server to a platform inside one evening, and will again.

- **A chat client.** Both users already have a better one. The value is the shared *store*, not a shared chat.
- **CRDTs and real-time collaborative cursors.** Whole-file writers make the first pointless and the second is a hard UX problem solving a case that a shared call already handles.
- **Continuous bidirectional file sync.** The hard part of a Dropbox is that it is automatic and continuous; explicit fetch-and-commit under a lease is a hundred lines.
- **Range-anchored comments.** Later room.
- **Tags, hierarchies, task trees, project management, instruction cascades.** None of it. This is a document store with leases and comments.

## Open questions

- **Human editing surface** — browser editor only, or a thin CLI (`lock` / `commit`) so existing local editors keep working? The CLI is smaller and avoids editor lock-in; the browser is better for casual review and for phones.
- **Auth model** — how identity propagates from person to agent to store, and how the MCP endpoint is authenticated given it fronts an entire corpus and is publicly reachable.
- **Comment worker context** — document plus thread is the obvious minimum; whether it should retrieve related documents is a retrieval question, not a v1 one.
- **Where conversation history lives** if it ever exists — another table beside documents is the natural answer.

## Change log

- 2026-08-24: created. A shared document store for humans and agents on one corpus, named for the mechanism (hold it, pass it, drop it = lease expiry). Core decisions: **server-authoritative** (git gives distribution without coordination; whole-file writers defeat CRDTs); **pessimistic leases** chosen deliberately for the regime — two people, prose, long-lived edits, and **bimodally** distributed contention (rare apart, constant when working together, which is exactly when the store matters); **reads never block and return a version, writes take a lease on intent-to-edit**; leases + visibility + human force-break + specified refusal behaviour + escalation with canonical ordering; **identity belongs to the person, not the service**. **One MCP server, several clients** — multi-agent coordination is emergent, not designed. Review surface where **the edit button is the lock acquisition**; an application, not a static generator. **Comments with `@claude` instead of a chat surface** — anchored, shared, async, phone-friendly; heading-anchored in v1; **agent proposes, human commits**. Scope boundary written down explicitly.
