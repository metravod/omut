# Handoff comparison: first controlled pilot

Reported 2026-11-06. [Protocol and corrections](handoff-benchmark.md); [machine-readable results](handoff-benchmark-results.json).

This pilot compares agent-written omut files with an agent-written free-form handoff note. A fresh session with no saved handoff is a lower bound. It is an exploratory smoke test on one source tree, not a ranking of memory products.

Both the plain note and omut passed 3/3 balanced scenarios; no saved handoff passed 2/3. This pilot did not show an advantage for omut over a good note. The small sample does not establish equivalence or an efficiency gain.

All model sessions used Codex CLI 0.166.0 with `gpt-6.1-sol`, high reasoning, and a four-minute per-session limit. The balanced table has one source/receiver pair per scenario and condition. Source is pinned to `52c4601ea573c021545491b57155eaaa433d2fd7`.

## Balanced result

| Scenario | No handoff | Plain note | omut |
|---|---|---|---|
| orderly-crlf | fail | pass | pass |
| stale-thread | pass | pass | pass |
| irrelevant-history | pass | pass | pass |

A pass requires completed source and receiver sessions, allowed changes, the full Go gate, independently retained policy tests, and the CRLF mutation check where applicable. A green existing test suite alone is insufficient.

## Recorded cost of the balanced blocks

These totals include source review, verification and memory writing, plus continuation and final memory updates. Cached input is a subset of input; reasoning output is a subset of output. No currency conversion is made.

| Condition | Passes | Source seconds | Receiver seconds | Input tokens | Cached subset | Uncached input | Output tokens |
|---|---:|---:|---:|---:|---:|---:|---:|
| none | 2/3 | 201.7 | 180.3 | 1721717 | 1516288 | 205429 | 7712 |
| note | 3/3 | 274.2 | 256.9 | 2084814 | 1852544 | 232270 | 13525 |
| omut | 3/3 | 278.8 | 325.2 | 1845070 | 1641216 | 203854 | 18514 |

Every source phase starts with a fresh adoption/review. This does not measure amortized maintenance across many tasks. Original runs encountered nested temporary-fixture problems; the corrected robustness repeats overlapped on one host. These times and token counts do not establish a general efficiency advantage.

## Corrections and retained initial attempts

The initial repair scorer wrongly rejected changes to relevant regression tests. It was corrected uniformly without new model calls; the original scores were retained.

Nested test temporary directories then exposed the fixture to its parent project and left a CLAUDE.md symlink in two omut cells. The source-phase occurrence prevented a receiving session; the receiver-phase occurrence passed functional tests but failed the workspace-scope check. Both complete robustness blocks were repeated for all three conditions with dedicated external scratch directories. The original blocks remain below and in the JSON; they were not selectively retried by condition.

| Original scenario | Condition | Status | Corrected strict pass |
|---|---|---|---|
| stale-thread | none | scored | true |
| irrelevant-history | none | scored | true |
| irrelevant-history | omut | scored | false |
| stale-thread | note | scored | true |
| stale-thread | omut | invalid-source | false |
| irrelevant-history | note | scored | true |

The retained dataset contains 29 completed model sessions, including supplementary attempts. A separate initial network-routing setup failure reported no token usage and produced no receiver; it is not task evidence. Offline preflight and rescoring invoke no model.

## Interpretation and limits

Compare the plain note with omut first. Identical pass counts on this small corpus do not establish equivalence; a win against no saved handoff alone would not distinguish omut from a good note. Do not infer general quality gains, token savings, or cross-provider portability from this pilot.

The orderly no-handoff session correctly reported that it could not identify a pending task and left code unchanged. Its existing tests passed, but the missing CRLF regression coverage failed the mutation criterion. Memory supplied actionable task intent that the code did not select on its own.

This is one repository and three controlled scenarios; two scenarios share a repair. Each summary cell has one source/receiver pair. Source agents wrote their own records from the same supplied facts, but their review and test trajectories differed. Native memory and native startup hooks were disabled. No blinded semantic score was assigned, and instruction-based read isolation is weaker than a container.

The next useful study needs independent tasks from several repositories, repeated paired runs, source implementation and multiple sequential handoffs to measure maintenance cost over time. Keep the ordinary-note baseline. No product behavior or check rule in omut was changed by this experiment.

Raw events, prompts, invocations, final patches and memory snapshots are retained in this checkout's ignored local meta. Public JSON retains per-stage accounting, check outcomes and manifest/runner hashes without host paths.

## Product decision after the pilot

On 2026-11-06, the human clarified that the acceptance question was whether
omut introduces task-success regressions. None appeared in these three balanced
scenarios; this is sufficient for the current stage, without constituting a
statistical non-inferiority result. Further comparative benchmarking is stopped
(D20). The broader study described above remains methodological context, not
scheduled work.

The expected long-term value is disciplined maintenance, a recognizable format,
startup-context assembly and support for revisiting older recorded claims.
Those are the product direction, not benefits established by this pilot.
