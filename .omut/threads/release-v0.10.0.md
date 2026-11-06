---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.10.0
**Goal:** publish optional work-plan support at the human's request, preserving the task package another agent is executing.
**Where we got:** implementation a35e46a plus metadata link fix ae8be8b23cc28ed57cd259482871e32238ca1c17 are pushed; annotated v0.10.0 points at ae8be8b. Published release and notes: https://github.com/metravod/omut/releases/tag/v0.10.0. A macOS case-alias privacy regression was reproduced and fixed before publication. Linux meta CI exposed a reference spelling that depended on ignored local evidence; the supported project-root local reference fixes clean checkouts. Existing active task packages were not migrated or rewritten.
**Verified:** final go vet ./... and go test ./... pass, including the case-alias regression (failed before fix); built and released CLI strict meta checks and diff check pass. Export without local evidence passes. Tagged commit main CI 42244035396 and meta 42244035527 are green. Release workflow 42244186249 passed macOS vet/tests and GoReleaser. Six archive asset digests match checksums.txt; Homebrew cask is 0.10.0 with four matching hashes. install.sh checksum-verified macOS arm64; downloaded binary reports 0.10.0 and passes own meta check. Isolated released-binary smoke verifies shared/local selection, pack-hook parity, project startup, no linked-card inclusion, embedded rules, and unchanged plan/card/owner bytes and mtimes after reads and zero. Local payload/notes/smoke evidence stays in local/.
**Not verified:** running the other five released binaries, a Homebrew upgrade on this machine, live task-continuation or token-quality gains. System omut remains 0.9.1; no existing hook configuration or other-session project files were changed.
**Open:** none.
**Next step:** none for the release; adopt the optional convention only with the active package's keeper or on future work.
