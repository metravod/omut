---
status: active
updated: 2026-11-11
by: zcode/glm-5.3
---
# Release v0.11.0
**Goal:** publish memory dreaming at the human's request: `omut dream`, the volume-based `dreamed:` reminder, `hook start --preview`, stable-first output ordering.
**Where we got:** the other session's parked threads-introduction/memorilla work committed first (71105d0), then the feature (e7b9abe): internal/dream (counter, digest, contracts), cmd/omut/dream.go, the reminder and --preview in hook, stable-first reorder, shape validation of dreamed/dream, spec §5/§7/§8/§9, help.go, templates (README/LEAD/brief), docs/dream.md recipes, README section, D25. This repository's tool file refreshed by zero with the new binary; the pack root's LEAD.md waits for the brew upgrade per the runbook. Publication and released-binary checks pending.
**Verified:** go vet ./... and go test ./... pass; `check --fail-on warn` on this repository passes after the log ordering fix; `git diff --check` clean. Dream fixtures: counting per kind (log, lenient archive, thread, decision, graveyard), strictly-after dreamed:, threshold default/override/invalid, due boundary, report vs apply wording, JSON, no writes on --apply, scopes with `..`, incomplete on read/shape errors. Reminder fires at the threshold, extinguishes after dreamed:, frozen excluded. --preview with and without a payload; commit --preview refused. Live smoke: 81-record digest, reminder active on this repository, section order checked.
**Not verified:** publication, the release workflow and its macOS gate, archives/checksums, the Homebrew cask, the installer, released-binary smoke, a real dream run by an agent, the pack root's LEAD.md refresh.
**Open:** none besides the pending checks above.
**Next step:** tag v0.11.0, watch CI, verify assets/cask/installer, brew upgrade and zero the pack, record the verified release.
