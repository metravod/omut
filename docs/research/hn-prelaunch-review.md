# Lessons from related Show HN launches

2026-11-05. This is an exploratory review of selected discussions and omut's
current behavior, not a representative user survey. The proposed follow-up is
small: improve the context budget, show a real continuation, and verify the
private onboarding path. No new memory architecture is implied.

Follow-up: the human approved implementation on the same date. Unicode budgets and compactness are implemented; the live setup and continuation results are in [the worked handoff](../handoff.md). The measurements and unchecked items below describe the initial review.

## Reader concerns and their relevance

| Concern observed in a discussion | What it suggests for omut | Existing answer / remaining evidence |
|---|---|---|
| Readers compare tools with Markdown plus grep, and report that agent/harness changes affect whether memory is used ([MCP Memory](https://news.ycombinator.com/item?id=49286073)) | Treat discovery and repeatable use as an integration concern, not a solved consequence of choosing a file format | Shared AGENTS.md pointers exist; private Codex discovery still needs its documented hook or global instruction |
| Short fresh sessions and focused files can already work well; maintained summaries can retain abandoned approaches ([Recall](https://news.ycombinator.com/item?id=48622590)) | Demonstrate useful facts that code cannot reconstruct, and avoid injecting release history into every task | Decisions and threads have separate homes; this checkout's brief still contains a long release chronology |
| Readers ask whether memory improves outcomes, rather than merely searching quickly; they ask about forgetting and contradictions ([Mnemo](https://news.ycombinator.com/item?id=48389586)) | Measure a resumed task's correctness and distinguish current state from historical decisions | `superseded` decisions and completed threads are represented; no model-quality comparison was run here |
| Readers ask when memories are written, whether records are trustworthy, and how a crash differs from a completed session ([Engrim](https://news.ycombinator.com/item?id=49594008)) | Verify both orderly handoff and interruption; describe what the CLI can detect | Start reminders and pre-commit checks exist; the manual interruption signal works, but does not prove the next agent reconstructs the work correctly |
| Readers ask for concrete records and before/after workflows, lifecycle and supersession rules rather than implementation details ([ContextVault](https://news.ycombinator.com/item?id=48900288)) | Show one small project moving from decision through partial work to continuation | The repository's `.omut/` is inspectable, but a short worked handoff would be easier to evaluate |
| Readers prefer a local file bridge and question third-party context storage ([Segue](https://news.ycombinator.com/item?id=49082779)) | Make the local, inspectable workflow easy to try and back out of | omut's CLI is local and `--private` preserves tracked files; reader familiarity with an external service is unnecessary |
| Readers ask about source provenance and precise pointers for restarting work ([Lossless-memory](https://news.ycombinator.com/item?id=49786419)) | A thread should identify the unfinished action, its file/check, and the last observed result | Thread labels allow these facts; their presence alone does not establish their quality |

These comments identify questions worth testing. They do not establish that
any particular memory technique helps or harms model performance.

## Concrete finding: the line budget does not bound context volume

Measured at the start of this review on the current working copy:

| File/output | Lines | Bytes | Whitespace-delimited words |
|---|---:|---:|---:|
| `.omut/brief.md` | 27 | 8,004 | 1,240 |
| `.omut/README.md` | 104 | 13,342 | 2,179 |
| `.omut/decisions.md` | 20 | 8,893 | 1,446 |
| `omut hook start` | 62 | 11,395 | 1,756 |

The hook includes the entire brief and the filled instance layer. It emits only
waiting decisions and summaries of unfinished threads; it does **not** inject
all accepted decisions or the full rules file. The rules can be read separately
by an agent. Word counts here are not token counts.

`internal/check/check.go` checks the brief's physical line count (60); it has no
independent character/word/byte limit. A disposable private fixture with a
57,446-byte, 17-line brief produced `[]` from `omut check --json`. Its long
paragraph consisted of repeated historical detail. This demonstrates an
unbounded size measure, not a measured decrease in task quality.

Suggested first action: make the project's own brief describe current behavior
and evidence, linking release history and detailed verification to their
existing records. Then choose a generous volume budget using real briefs,
including non-English ones. A Unicode character or byte warning needs only the
Go standard library; byte counts weight languages differently. A tokenizer is
not required. The threshold and policy remain undecided.

Any new check policy must update spec §8, the help table, the template rules and
appropriate behavior checks, per AGENTS.md. Avoid silently truncating meta or
having the CLI rewrite its meaning. Keep paths to omitted detail if hook output
is ever summarized; this would itself be a command behavior change.

The [official OpenAI hook documentation](https://learn.chatgpt.com/docs/hooks)
was checked on 2026-11-05. It supports the recipe's `cwd` input and plain stdout
as developer context, and says oversized hook messages are spilled to a file
with a preview at an approximate 2,500-token default. This increases the value
of concise output; no tokenizer or live Codex session was used here to establish
whether this particular output crosses that threshold. The current documented
start sources also include `fork`, absent from omut's example matcher; consider
whether fork support belongs in the recipe when testing it live.

## Manual checks performed

Using a freshly built binary and a disposable git repository:

1. Existing tracked `AGENTS.md`, `CLAUDE.md` and code were unchanged by
   `start --private`; `git status --porcelain --untracked-files=all` stayed empty.
   Repeating the command succeeded. The command explained Codex's extra setup.
2. A freshly dated, structurally valid brief intentionally claimed a CSV export
   that the fixture's Go file did not implement. `check --json` returned `[]`.
   This is the intended shape/freshness boundary, not a request for semantic
   validation in the CLI. A green check is not evidence that the prose is true.
3. The long-paragraph fixture described above had no `size` warning.
4. After moving the fixture log's mtime back and changing an uncommitted file,
   a simulated Codex `SessionStart` payload produced an `Unrecorded work`
   reminder naming the changed file and asking the agent to inspect `git diff`.
   This checks the CLI adapter and signal, not a model following the advice.

The raw measurements, fixture script, fixture location and startup output are
kept in ignored `.omut/local/hn-launch-analysis/`. User hook settings and source
repositories were not modified. No new model session was launched for this
review. `go vet ./... && go test ./...` is the project verification gate.

## A bounded continuation experiment

Before broad promotion, run one small, real continuation with a fresh agent
session; if practical, compare a second copy with the project's normal existing
docs. Keep the task, code state, model and tools equivalent. This is a worked
case, not a statistically conclusive benchmark.

Use a real project with an accepted constraint that is not obvious from code,
a superseded decision, and a partially completed task. Preserve the original
session as evidence outside the receiving agent's accessible context. Prepare
facts/questions before seeing the receiving session's answers:

1. What is actually implemented and what was merely planned?
2. Which decision currently applies, and which alternative was rejected? Why?
3. What was the last check, its observed result, and what remains unverified?
4. What exact file/function/check is the next action?
5. Does the agent continue without repeating an already rejected approach?

Run an orderly stop first: record the state, then have the receiving session
resume a concrete task. Separately interrupt work before the write ritual and
observe whether the receiving session reconstructs from the dirty tree instead
of treating an old brief as current. Score concrete facts and actions, count
human corrections, and keep actual failures. Do not substitute the speed of
`omut check`, plausible-sounding recall, or an author's impression for this
outcome. Do not claim token/time savings without measuring comparable sessions.

For the private path, the receiving session must start through the actual
Codex or other-agent setup the README asks users to perform, including trust
review. Passing a JSON payload directly to the CLI is not enough. Test from a
worktree as well if the launch highlights that capability. Existing unit tests
and the earlier live project-identification session are useful but cover less
than task resumption after interruption.

## What should precede the public launch

Recommended order, not accepted product decisions:

1. Compact the dogfood brief and address/decide the volume-budget gap.
2. Run and publish one worked continuation with exact partial-work pointers,
   including an honest interruption result. This can double as the demo.
3. Verify private onboarding in a real receiving session; make its Codex setup
   visible where `--private` is introduced. Add a short manual opt-out guide
   that preserves pre-existing agent files and notes. Never present `kill` as
   uninstall: it removes a project's whole directory.

A retrieval engine, automatic memory decay, a new database, transcript ingestion
or a semantic judge do not follow from these comments. The core shape-only
contract remains appropriate. Demonstrating that contract and its practical
benefit is the useful prelaunch work.

## Completed prelaunch pass

The human authorized these follow-ups on 2026-11-05. Implemented D19: 6000/8000
Unicode code-point warnings in addition to line limits, with spec/help/template
updates. The regression failed on the old implementation and passes with ASCII,
Cyrillic and LF/CRLF boundaries; the CLI still only reports and never rewrites.
Five of six real dogfood briefs measured 1751–3410 characters; only omut's
history-heavy brief exceeded 6000. Its current state is now compact, with the
previous full brief retained in the meta archive.

The [worked handoff](../handoff.md) records native Codex trust review, an orderly
continuation, an initially missing worktree hook, a tool-free delivery probe,
and a corrected continuation of the controlled unfinished patch. It separates
model behavior from hook delivery and documents the setup correction. There is
no ordinary-docs control or cross-provider/quality/speed conclusion.

README now explains private discovery before a new session, the badge's evidence
boundary, and safe opt-out. A manual private opt-out preserved existing personal
notes, tracked agent rules, unrelated excludes and a clean git status. Native
trust state and raw runs are local-only; no global hook file was installed.

Final dogfood measurements: brief 2,805 bytes (was 8,004); startup output 6,082 bytes (was 11,395), including the filled local layer. This is a byte-volume comparison, not token or task-quality evidence.
