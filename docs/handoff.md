# A worked handoff on omut itself

Checked on 2026-11-05 with Codex CLI 0.166.0, `gpt-6.1-sol`, high reasoning
(the configured default). These are controlled cases on an isolated snapshot
of the real omut source, with actual edits and Go checks. They are not a
cross-provider comparison, a performance benchmark, or a production crash test.

## The unfinished task

The source session had added 6000/8000 Unicode-character budgets to the brief
and thread checks, keeping the 60/80-line budgets. ASCII and Cyrillic tests at
and above each limit passed. The spec, help and template had been updated.
The remaining action was a CRLF boundary regression in
`cmd/omut/size_test.go`, followed by the full verification gate.

The private memory recorded:

- Current state and the exact last passing command in `brief.md`.
- A superseded line-only decision and its replacement in `decisions.md`.
- The reason for code points rather than UTF-8 bytes: language fairness. The
  rejected tokenizer would have added a dependency to a standard-library CLI.
- The next file, test and unverified case in `threads/volume-budget.md`.

The private pointer did not change the tracked `AGENTS.md`. The receiving
sessions were fresh, without the source conversation. The initial receiving
prompt asked to resume the unfinished work, distinguish recorded from observed
state, and verify the result; it did not name the pending CRLF case or the
location of the memory. Expected facts were recorded before inspecting the
first receiving session's output.

## Recorded stop, then a fresh session

The project-local `SessionStart` hook was inspected and trusted through the
native `/hooks` UI. A new session then ran with no hook-trust bypass flag.
Its transcript contains a separate developer message with the full private
brief and omut header (2070 UTF-8 bytes).

Before editing, the receiving session correctly identified:

| Question | Observed answer/action |
|---|---|
| What already works? | Character and line budgets, ASCII/Cyrillic boundary checks, and matching spec/help/template |
| What is merely planned? | CRLF at the same character boundaries and the full suite after adding it |
| Which decision applies? | Unicode code points supplement line limits; the line-only decision is superseded |
| Why? | Long paragraphs escape line limits; byte counts penalize non-ASCII text; the CLI stays dependency-free |
| Where to continue? | `TestSizeBudgetsBoundLongParagraphs` in `cmd/omut/size_test.go` |

It also said that the original rationale for the superseded decision was not
recorded, rather than inventing one. It reran the recorded focused checks,
added LF/CRLF cases for ASCII and Cyrillic at both budgets and one character
over, and passed `go vet ./... && go test ./...`. It updated the brief, thread
and logs before its final answer. No human correction to the task facts was
needed. The CRLF regression was reviewed and brought back into omut's real test
suite; the CLI implementation did not need changing for CRLF.

## A controlled interrupted patch in a linked worktree

A linked worktree had no `.omut/` of its own. Its private memory remained in
the main checkout. The prior memory said the volume-budget task was complete.
To emulate a stop between a code edit and the memory ritual, the thread's
character-volume call was replaced with its former line-only check, without
updating the memory. The brief check and regression tests remained in place.
This is an injected unfinished state, not an observed process crash.

A receiving session had to notice that recorded completion did not match the
working tree, run the focused regression, restore the accepted policy and
record the newly observed result. The worktree's sandbox was explicitly given
write access to the main checkout for the shared private memory.

The first attempt exposed an onboarding problem in the experiment: after the
per-worktree hook file was removed, the session received no omut hook message.
It still found the main checkout's memory through file inspection, observed
the two failing thread cases, repaired the guard and passed vet/tests. That
shows a successful fallback; it does **not** verify hook delivery.

The hook file was restored in the worktree and its trust reviewed. A separate
read-only probe was instructed to use no tools or file reads. It named the
omut header, the main checkout's memory path and a verified fact from the brief.
The transcript confirms a separate 4454-byte developer hook message, and the
probe used no tools. The corrected setup was then used for another receiving
session on the same injected unfinished state.

With the corrected setup, the next session received a native developer hook
message containing both the worktree/main-checkout paths and `Unrecorded work`
(4860 UTF-8 bytes). Before changing code it distinguished the stale completion
record from the present line-only thread guard and the accepted Unicode policy.
It asked who made the unrecorded change rather than inventing its intent, then
confirmed the failure: an 8001-character thread passed with both LF and CRLF.

It reopened the thread, recorded the mismatch, restored the shared size check,
and passed the focused regression, fresh full tests, vet and `git diff --check`.
It updated the main checkout's private memory and verified it with
`check --fail-on warn` before its final answer. The author/intent of the injected
change remained explicitly unknown to the receiving session. No `.omut/` was
created in the worktree. No human correction or test-answer injection was sent
to that receiving session; the setup correction happened before it started.


## Trying a handoff on your own project

1. Use a small real task with a constraint or decision that code alone does
   not explain. Record what was checked and its result, plus the exact next
   action. Distinguish work done from work merely proposed.
2. Start a fresh session through your normal agent setup. For private Codex
   memory, follow the [hook setup and trust steps](hooks.md#codex). Ask it to
   resume; do not supply the answers you are trying to recover.
3. Check what it actually does. A plausible summary is insufficient: does it
   run the right check, avoid the rejected alternative, finish the next action
   and record the result?
4. Try an interruption separately. Compare the dirty tree with the old memory
   rather than assuming an old completion record still holds.

A user-level Codex hook covers worktrees without copying private project hook
configuration. If using `.codex/hooks.json`, make that configuration available
in the worktree too and check `/hooks` there. A sandboxed agent needs permission
to write the main checkout's private memory. Successful file discovery is not
evidence that a startup hook ran.

## Scope of the evidence

One model, one source tree and controlled states show that these paths can
work. The prompts explicitly requested verification, and the interrupted
session could inspect prior local run artifacts. There was no ordinary-docs
control group and no claim about saved tokens, time, or general reliability.
Only startup was exercised live here; resume, compaction and fork triggers and
other agents remain separate checks. Raw local session records and expected
facts are kept in the checkout's ignored local meta; the public record above
omits machine-specific paths and session transcripts.

A clean `omut check` is still a shape/freshness result. In a separate negative
control, a well-formed brief with an unsupported implementation claim passed.
The source and the agent's observed checks are the evidence of what works.
