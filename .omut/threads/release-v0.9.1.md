---
status: active
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.9.1
**Goal:** publish the Git-observation improvement to context as the patch release requested by the human.
**Where we got:** Git observations, private-worktree scope, read-only behavior, help/spec and lead guidance are implemented. Prepare notes, commit/push, wait for main gates, then tag v0.9.1. Refreshing the pack root with `omut zero` activates the updated lead instructions.
**Verified:** the implementation turn passed local vet/tests, metadata check, real Git fixtures and six-project pack smoke, including unchanged Git indices and hook/context parity. Release workflow builds six archives after macOS vet/tests and updates the Homebrew cask.
**Not verified:** new main CI, tagged macOS gate, published v0.9.1 assets/cask and checksum-verified installation.
**Open:** none.
**Next step:** complete local release gates, commit and push; await main CI/meta before pushing annotated v0.9.1; verify release, notes, hashes and installer; record the final result.
