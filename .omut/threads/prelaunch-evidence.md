---
status: done
updated: 2026-11-05
by: codex/GPT-6
---
# Evidence before broad public launch
**Goal:** verify task continuation, bounded startup memory and usable private setup before wider promotion.
**Where we got:** D19 adds 6000/8000 Unicode-character warnings alongside line limits; brief history moved to log-archive/brief-before-prelaunch-2026-11-05.md. README clarifies private Codex discovery, the badge and safe opt-out. Native Codex cases on real omut source are documented → docs/handoff.md; initial review → docs/research/hn-prelaunch-review.md.
**Verified:** size regression failed before the fix and passes at/above ASCII/Cyrillic and LF/CRLF boundaries; go vet ./... and go test ./... pass. Native Codex trust review and transcript evidence confirm private startup injection; a fresh session finished the CRLF regression. With worktree hook configuration corrected, a fresh session observed failing tests in a controlled unfinished patch, restored the accepted guard and updated main-checkout memory without a second .omut/. Manual opt-out preserves personal notes, tracked rules and unrelated excludes.
**Not verified:** comparative quality/speed/token savings, cross-provider handoff, actual process-crash recovery, Codex resume/compact/fork triggers and Cursor live. The interrupted case was an injected state, not a production crash. These claims are outside this bounded pass.
**Open:** none for this pass; future live-use checks remain in the brief.
**Next step:** none; gather outside attempts after publication.
