---
status: active
updated: 2026-11-11
by: zcode/glm-5.3
---
# Release v0.11.1
**Goal:** trim the session-start context by selection, not by modes (D26), at the human's request after they measured the dump: a parked plan was 26% of it.
**Where we got:** the startup hook now collapses a parked/dropped plan (optional frontmatter `status:`) to a pointer line, prints due claims as locations only (their lines are already in the dump), keeps Access and Runbook behind pointers, caps the local log at three entries, and carries a tighter intro. `context` keeps the full selection. Spec §5, help, README rules template updated with the plan status convention; D26 recorded. On omut itself the dump went 15 098 → 10 033 characters; the first dream (next step) compresses the brief further.
**Verified:** go vet ./... and go test ./... pass; new fixtures: parked plan collapses and an active one keeps its body, Access/Runbook pointers, claim locations with no repeated `how`, local log capped at three; TestClaimsAreCheckedByAgents updated to the location format. `check --fail-on warn` on this repository clean.
**Not verified:** publication, the release workflow, archives, cask, installer, released-binary smoke, the pack's zero refresh.
**Open:** none besides the pending checks.
**Next step:** tag v0.11.1, verify, brew upgrade, zero the pack, record; then run the first real dream on omut itself.
