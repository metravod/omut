---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.9.1
**Goal:** publish the Git-observation improvement to context as the patch release requested by the human.
**Where we got:** implementation 28dc65a plus portable merge-fixture fix b2e940f434740fdd03404f4c2e8839ea125bcc8d are pushed; annotated v0.9.1 points at b2e940f. Published release and notes: https://github.com/metravod/omut/releases/tag/v0.9.1. Upgrade guidance calls for `omut zero` in the pack root. The initial Linux failure was the fixture's implicit Git identity; explicit identity and a MERGE_HEAD precondition fixed it.
**Verified:** isolated fixture fails before the fix and passes after it; local `go vet ./...`, `go test ./...`, meta and diff checks pass. Corrected main CI 42191212688 and meta 42191212701 are green on the tagged commit. Release workflow 42194282165 passed macOS vet/tests and GoReleaser. Six archive hashes in checksums.txt match GitHub asset digests; Homebrew cask is 0.9.1 with four matching hashes. install.sh checksum-verified macOS arm64; the binary reports 0.9.1 and passes own meta check. Released project/six-project context and pack hook agree; all six indices retain bytes and mtime. Fresh Git fixtures verify actual private-worktree status with main-checkout meta and embedded lead guidance.
**Not verified:** running the other five released binaries locally, Homebrew upgrade on this machine, a fresh native lead following the guide, large-repository performance or model-token/task-quality gains. The system installation and existing hook configuration were not changed.
**Open:** none.
**Next step:** none.
