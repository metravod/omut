# Measuring omut's contribution to task continuation

Status: exploratory protocol, frozen before the first model run on 2026-11-05.
Runner: [tools/handoffbench](../../tools/handoffbench). This is research tooling,
separate from the omut CLI. It does not add semantic memory writing to the CLI.

## Question and comparisons

Does agent-maintained omut memory improve successful continuation relative to
an agent-maintained free-form handoff note, after including the cost of writing
and maintaining that memory?

The primary comparison is **omut versus a plain note**. A fresh session with no
saved handoff is a secondary lower bound. A win against no saved handoff alone
would not establish a reason to use omut instead of a good note.

| Condition | Source session | Fresh receiving session |
|---|---|---|
| `none` | Reviews the source task; no persistent handoff configured | Gets the current tree and ordinary project rules |
| `note` | Same review; instructed to maintain `handoff.md`, choosing its own structure | Project rules point to the note and require keeping it current |
| `omut` | Same review; project pointer and unfilled templates from the pinned release | Project rules point to the agent-maintained omut files |

All conditions receive the same source facts and the same receiving request
within a scenario. The runner never writes the source facts into either memory
format. It only creates the omut templates and the pointers. The source agent
must inspect the code, run checks and decide what to retain. The note has no
artificial length cap. Its token and time costs remain part of the comparison.

The initial runner uses Codex file discovery through `AGENTS.md`, not native
startup hooks. Native memory is disabled in **every** condition. Each phase gets
a fresh Codex home and runs ephemerally, without personal configuration, plugins,
host skills, hooks or prior session history. Thus `none` is **not** a test of the
best native memory a coding agent can provide. That needs a separate experiment.

## Initial corpus: controlled smoke cases

Source is pinned to omut commit
`52c4601ea573c021545491b57155eaaa433d2fd7`. Only Go product source, embedded
templates, `go.mod` and the spec enter each fixture. The real project's meta,
worked handoff, research documents, runner, gold verifier and git history are
excluded. Each fixture gets a new local git repository.

These are controlled modifications of **one** real source tree, not three
independent real-world development tasks. Source sessions review and verify
existing work; they do not implement the original feature. The smoke validates
the experiment machinery before spending on a broader corpus.

| Scenario | Source experience | Receiving task | Executable criterion |
|---|---|---|---|
| `orderly-crlf` | Volume-budget feature exists; visible boundary tests exercise LF only. Next phase is explicitly left pending: CRLF coverage for both alphabets and both file budgets | Generic request to continue unfinished work; the request does not reveal the pending test | Full gate, independent policy tests, unchanged production code, and the agent's tests must catch a deliberate CRLF-counting bug |
| `stale-thread` | Source verifies the complete feature and can correctly record completion | After the source ends, the controller replaces the thread's volume check with a line-only check, leaving the records stale. Same generic continuation request | Full gate and independent policy tests; changes limited to the relevant implementation and memory |
| `irrelevant-history` | Source reviews completed ADR-status handling | Controller injects the same thread regression; receiving request explicitly describes the new volume bug | Same functional criterion; prior ADR experience is irrelevant to the requested fix |

The last two cases share a repair and must not be treated as independent
evidence about general task diversity. The stale case is an injected mismatch,
not a test of recovering from an actual process crash.

Before **any** model outcomes are inspected, preflight verifies that the baseline
passes, the injected regression fails an actual test assertion, and the retained
reference answer passes the full scorer. For CRLF coverage, preflight additionally
requires that the LF-only baseline survives the counting mutation and the
reference CRLF tests kill it. A compiler error or infrastructure failure does
not count as a killed mutation. Verification restores its source changes.

## Outcomes and accounting

The automated outcome is a **functional pass**: completed source and receiver
sessions, permitted file changes, `go vet ./...`, `go test ./... -count=1`, the
held-out policy tests, and, where applicable, the mutation check all pass.
Source product changes invalidate a cell because the continuation states would
otherwise differ between conditions. Incomplete sessions remain in the report;
they are never silently retried or dropped. Diagnose infrastructure failures
separately from task failures before making comparisons.

The held-out policy test is retained from the pinned release and is installed
only after the receiver exits. The receiver does not have the runner's gold
test file, although it can infer requirements from the public spec. Score
functionality, not textual agreement with an expected explanation.

Capture separately for source and receiver:

- Input tokens, cached input tokens (a **subset** of input), output tokens and
  reasoning output tokens (a subset of output). Do not add subset categories
  again when reporting totals.
- Wall time and completed command count. Total agent time includes the source's
  review, verification and memory writing, plus continuation and final updates.
- Source handoff bytes, exact prompts and invocation, complete JSONL events,
  stderr, final reports and every verifier log.

No human hints are supplied during a run. The initial runner does not assign a
semantic score to whether final records are accurate, whether work was repeated,
or whether rejected decisions were revived. Those require a separate blinded
review; a functional pass is not proof that memory was maintained correctly.

For that review, an evaluator should receive the initial and final code, source
facts, anonymized action transcript and final answer, without the condition's
label or memory files. Use a rubric fixed before reviewing outcomes: recovered
task, respected current decision, distinguished observed from recorded state,
unnecessary rework, and needed human corrections. Some condition clues may remain
in the transcript; report the limits of blinding. Review memory accuracy separately.

## Reproduce

Requires Go 1.24+, git, and a Codex CLI supporting the recorded flags. The model
defaults to the configured model used in the earlier worked handoff,
`gpt-6.1-sol`, with high reasoning. The runner records the installed CLI version,
resolved source commit, executable SHA-256, complete configuration, randomized
cell order and per-phase prompts/invocations. Pin the CLI binary too when
reproducing another machine's results. `plan.json` is saved before preflight and
model execution; `manifest.json` is updated as cells finish.

Offline fixture and verifier checks; the output directory must not exist:

```sh
go run ./tools/handoffbench -out /tmp/omut-benchmark-offline
```

One scenario, all three conditions, one source/receiver pair each (six model
sessions), with five-minute session limits:

```sh
go run ./tools/handoffbench \
  -out /tmp/omut-benchmark-smoke \
  -cases orderly-crlf -repeats 1 -execute
```

All three smoke scenarios (18 sessions):

```sh
go run ./tools/handoffbench \
  -out /tmp/omut-benchmark-three-cases \
  -repeats 1 -execute
```

`-execute` invokes the model and consumes the connected account's usage. The
default is offline. A timeout or configuration failure is recorded, not hidden
by a retry. Fix the setup, start a new output directory, and keep the failed
attempt. Read `report.md` for the summary and `manifest.json` for individual
checks. Keep raw local output out of git; transcripts include machine paths.

Auth is referenced through a temporary symlink to the existing Codex auth file,
never printed or copied into the fixture. The symlink is removed after each
phase. No personal Codex settings or persistent memory are changed.

### Isolation limits

`workspace-write` is not a container: it restricts writes, not every host read.
The no-parent/no-external-files input boundary is an explicit agent instruction.
Inspect transcripts for violations before interpreting results. Strong public
claims require container/VM isolation with only the allowed inputs mounted and
network access blocked at the tool boundary. Hidden verifier material and other
cells must then be unreachable, not merely excluded from the initial workspace.

The shared Go cache is warmed by preflight. Runtime latency, cache effects and
provider caching still affect wall time. Randomize cell order with a fixed seed;
interpret time as secondary to task success and token accounting. No monetary
price assumptions are embedded in this runner.

## Next experiment after smoke

Freeze 6–10 independent tasks from several repositories, with at least three
source/receiver repeats per condition. Include source implementation, changing
decisions, interrupted recording, useless history, multiple sequential handoffs
and held-out tasks not used to design the rules. A sequence matters: record
maintenance cost over several tasks, rather than repeatedly paying adoption
cost for one continuation.

Keep the primary contrast omut versus plain note. Report paired outcomes by task,
failures, source and receiver costs, and uncertainty clustered by independent
task. A small pilot establishes feasibility and suggests effect size; it does
not establish general superiority. Only add another model/provider after the
first comparison is informative. Test native memories and native hooks as
separate, explicitly configured conditions rather than changing this protocol
after seeing a result.

## Related benchmarks

- [DreamBench-SWE](https://arxiv.org/abs/2608.20664): multi-session software tasks
  needing earlier evidence, evaluated by executable hidden oracles; a close
  methodological reference for handoff testing, with synthetic scenarios.
- [MemDev-Bench](https://arxiv.org/abs/2610.27524): memory over earlier tasks from
  real repository histories, with control over historical relevance. Useful for
  checking both benefit and memory-induced regressions.
- [ContextWeave](https://arxiv.org/abs/2608.04830): longitudinal real workflow
  streams, downstream task quality and robustness to misleading recall.
- [VibeMemBench](https://arxiv.org/abs/2609.23570): executable repository tasks
  under matched memory conditions. It reports that eleven of twelve evaluated
  solver/system pairs did not exceed their memory-off baseline; this is a
  result of that setup, not a verdict about omut.
- [LongMemEval](https://github.com/xiaowu0162/LongMemEval) and
  [LoCoMo](https://github.com/snap-research/locomo): useful recall/reasoning
  evaluations over interaction history; they do not directly establish a
  benefit for finishing repository work.

The [existing worked handoff](../handoff.md) is feasibility evidence from one
source tree and controlled states. It has no ordinary-note control group and
must not be presented as a comparative benchmark.

## Scorer correction, 2026-11-05

During the first robustness run, the `irrelevant-history` receiver in the `none`
condition repaired the defect and expanded the relevant boundary tests. The
initial scorer rejected its `cmd/omut/size_test.go` changes because the repair
allowlist admitted only the implementation file. The receiving task did not
forbid regression-test changes; this was a scorer defect, not a memory failure.

Scorer 2 admits changes to the relevant boundary test and new CLI test files for
repair tasks as well as test-only tasks, while retaining protection of unrelated
existing tests. Full and held-out checks remain mandatory. The original scores
and all model outputs are preserved. Apply the correction uniformly to every
cell, without new model calls, and write separate results:

```sh
go run ./tools/handoffbench \
  -rescore /tmp/omut-benchmark-smoke \
  -out /tmp/omut-benchmark-smoke-rescored
```

The rescore records its original manifest's SHA-256 and the corrected runner's
hash. Report this as a disclosed correction after observing one rejected valid
patch; it is not a fully preregistered analysis. Do not erase or relabel the
initial automated failure as a product failure.

## Test-environment correction, 2026-11-05

The original stale-thread/omut source review completed, but its source scope
check found a new `CLAUDE.md -> AGENTS.md` link. The transcript shows that its
file edits changed only memory; its nested `TMPDIR` test run discovered the
workspace's parent project, failed several isolation assumptions, and left the
link as a test side effect. The runner correctly refused to continue from that
changed source tree. Treat this cell as a setup-contaminated, unscored handoff,
not evidence that omut's receiving agent failed.

The amended runner explicitly supplies a distinct external `TMPDIR` per phase
and tells every condition to use it as provided. Repeat the **whole stale-thread
block**, all three conditions, with this common correction; do not selectively
retry omut or discard the original block's two valid receiving results. Use the
replacement complete block for a balanced three-scenario exploratory summary,
and publish the incomplete initial block separately. This is a disclosed
environment correction after inspection, not a preregistered confirmatory test.
Token/time results from the original environment are especially noisy and must
not establish an efficiency claim.

During the full rescore, the same extra link was found after the original
irrelevant-history/omut receiver. Its functional tests passed, but its workspace
scope did not. Apply the same environment correction to the **entire
irrelevant-history block** as well, keeping its original results separately.
The balanced summary uses the original orderly block and the two complete
environment-corrected robustness blocks. The initial robustness block's
functional passes must not be mislabeled as fully protocol-compliant passes.
