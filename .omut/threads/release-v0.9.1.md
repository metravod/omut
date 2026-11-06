---
status: active
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.9.1
**Goal:** publish the Git-observation improvement to context as the patch release requested by the human.
**Where we got:** implementation committed as 28dc65a92b88bb8cc23e19920e1daf8833c594b7 and pushed. Main CI 42190542234 failed the merge-conflict fixture; its merge lacked an explicit Git identity and accepted any failure as a conflict. Reproduced locally with empty global/system config and user.useConfigOnly. Fix supplies the identity and requires MERGE_HEAD; reverify and push before tagging. Notes are prepared in local/; upgrading needs `omut zero` in the pack root.
**Verified:** the isolated fixture fails before the fix and passes after it; corrected local `go vet ./...` and `go test ./...` pass. Prior six-project smoke preserves indices and hook parity; main metadata 42190542128 and installer jobs pass. Corrected Linux CI pending.
**Not verified:** corrected Linux CI, tagged macOS gate, published v0.9.1 assets/cask and checksum-verified installation.
**Open:** none.
**Next step:** inspect the failed Linux test log, reproduce/fix and push; await all main gates before creating annotated v0.9.1; then verify publication and installation.
