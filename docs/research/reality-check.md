# Checking the meta against reality (D11)

A research note as of 2026-11-03, for D11 and D15. Nothing here is decided. The section "Two layers" was added the same day, after the human pointed out that the variants mixed two different jobs; read it first. The question: how does omut check that "Where it lives" and "Now" are true, not only that they are well-formed and recently edited? This is what sets omut apart (spec §1), and today it is prose in `LEAD.md` ("Audit") and one `verified:` date per brief.

## What the claims actually are

The "Where it lives" sections of the six projects in `~/oss_projects`, sorted by what it takes to observe them:

| Class | Examples from the pack | Who can observe it | Changes |
|---|---|---|---|
| **Local footprint** | symlink `~/.claude/skills/adr` → the project; `~/.local/bin/zymi` is 0.9.3; MCP server `zymi` in `~/.claude.json`; `~/.zymi` exists; dependents in `~/Documents/pet_projects` | anything on this machine, offline | on moves, installs |
| **Public footprint** | repo public/private, last commit, tags; PyPI `zymi-core` 0.9.3; the Homebrew cask; a GHCR image | anyone with the network, no secrets | on releases |
| **Private remote footprint** | container `wimt-agent` on VPS workhorse, image `wimt_agent:62b9ca0`, backups in `~/wimt_backups/`, secrets in `.env` | whoever has ssh: the human, an agent with permission | on deploys, by itself (a crash) |
| **Relations** | stare-decisis is used by wimt_agent and zymi-core (`adr/`); five neighbours depend on zymi-core | the pack, by reading the other projects | rarely |
| **Absences** | "Больше нигде не работает", "Nothing runs anywhere" | only by looking everywhere | silently |

Hand-checked on 2026-11-03, one day after the audit: three skill symlinks, the zymi binary and PyPI versions, the MCP config, visibility of six repos, commit counts, six neighbour directories. All true. The meta is right on day one; the question is day ninety.

Two observations from this inventory:
- **Most claims are already half-structured.** `symlink X → here`, `PyPI: name version`, `GitHub: url (public)`. They are written as prose, but a parser or an agent can lift them without new syntax.
- **The riskiest claims are the least checkable.** A prod container on a VPS and "nothing else runs" matter most and need either credentials or an exhaustive search.

## Two layers (D15)

Accepted and implemented the same day: `.omut/local/` (`instance.md`, `log.md`), `.omut/.gitignore`, the `leak`, `local-not-ignored` and `local-in-git` rules; see spec §4–§5 and §8.

The variants below first mixed two jobs that have different readers, privacy and lifetimes:

| | **Project layer** (`.omut/` in git) | **Instance layer** (the memory of *my* copy) |
|---|---|---|
| Question | what this is, why it is built this way, what comes next | where and how it lives *here* |
| Reader | any agent or person who clones the repository | me and my agents on this machine |
| Lives | in git, often public | next to it, never in git, like `.env` |
| Holds | Why, What it is, decisions, threads, the work log, the public footprint (releases, registries) | symlinks, installs, venvs, servers, containers, where secrets are kept (not the values), backups, deploy and rollback, moves |
| Same for everyone | yes | no: every clone has its own |

**The test for a fact:** would a stranger who clones the repository need it, and is it true for them? "PyPI has zymi-core 0.9.3" passes; "deploy by hand, roll back with `docker tag`" is true only for the owner of that VPS.

The evidence that the split is real, from the first pack:
- **Pack D1 hid the meta** because of internal things (prod, strategy). Pack D3 opens it after a cleaning pass: that pass is the split done by hand, after the fact.
- **Finding 9 ("local-only meta")** was this need, misread as "keep the whole meta out of git", and dropped on 2026-11-03 for that reason.
- **The 2026-11-02 move** broke only instance facts: skill symlinks, venv paths, agents' memory kept by path. The project layer survived whole.
- **wimt_agent's "Where it lives"** is almost all instance layer: VPS, container, `.env` on the host, the bot. It must never reach a public repository.
- **The pack itself is instance layer:** `~/oss_projects` is one person's arrangement of projects on one machine.

Mapping the five classes of claims above:
- **Public footprint** → project layer. Checked from outside, the same for everyone: CI and the badge.
- **Local footprint, private remote footprint, absences** → instance layer. This is where nearly all of D11's checking is.
- **Relations** → the pack, which is instance layer too.

### What the split changes for D11

- **Most reality checking is about the instance.** My prod, my symlinks, "nothing else runs here" are knowledge about my copy, not about the project.
- **The security problem mostly dissolves.** Probes in my instance layer were written by me on my machine; running them is running my own script, not a repository's untrusted code. One guard is needed: a file that is tracked by git is not instance layer, whatever its path, or a repository could ship "local" probes. Run instance probes only from untracked files (or from outside the checkout).
- **The project layer keeps only safe checks:** shape, freshness, the public footprint (variants 3 and 8). `check` on a stranger's clone stays offline and runs nothing.
- **Variant 6 (a trust gate) is not needed** for the instance layer, and stays out for the project layer.
- **Variant 4 (discovery) is an instance tool by nature:** it scans *this* machine and diffs with *this* instance's claims.

### Where the instance layer could live

- **`.omut/local/`, inside the checkout, excluded from git** (`.git/info/exclude`, written by `start`). Next to the project, any agent finds it, works without a pack, follows the `.local` convention (`.env.local`, `CLAUDE.local.md`, `settings.local.json`). Dies with the checkout; `kill` already refuses to remove gitignored files that no build recreates.
- **Outside, `~/.omut/<project>/`.** Survives a fresh clone, but is keyed by path or name, and the 2026-11-02 move showed what breaks when memory is keyed by path.
- **In the pack root.** Natural for a pack, nothing for a single repository.

### Hard parts

- **Every fact needs a layer, and agents will misplace them.** A short test in the rules, and a `check` rule as a safety net: warn about instance-looking facts in the project layer (home paths, `/Users/…`, IPs, hosts from `~/.ssh/config`).
- **Borderline facts.** "Autodeploy from CI was rejected": a project decision or an instance one? An instance probably needs decisions too; no new format until a live example demands one.
- **Backup.** The instance layer exists only on this machine, like `.env`. Fine, or a recipe (the pack as a private repository of its own)?
- **Handoff of an instance.** Passing prod to a colleague means passing the instance layer privately; out of scope for now.

## What neighbours do

- **Terraform / OpenTofu drift detection.** `plan -refresh-only -detailed-exitcode` reads real resources and diffs them with state, on a schedule. It only sees what is in state: resources created outside are invisible. ([Kestra](https://kestra.io/resources/infrastructure/terraform-drift-detection), [Buddy](https://buddy.works/docs/terraform/drift-detection))
- **Argo CD.** Desired state (git) vs live state, `Synced`/`OutOfSync`, health reported separately from sync; optional self-heal reverts manual drift; `ignoreDifferences` for fields other controllers own. ([OneUptime](https://oneuptime.com/blog/post/2026-02-26-argocd-sync-status-explained/markdown), [Red Hat lab](https://redhat-scholars.github.io/summit-2023-gitops-lab-guide/summit-2023-gitops-workshop-guide/main/m2-argocd-introduction/self-healing.html))
- **goss / Serverspec / InSpec.** Server state as declarative YAML (ports, services, files, users); `goss autoadd` *generates* the tests from the current state, `goss validate` checks them, `goss serve` exposes them as a health endpoint. ([goss](https://github.com/goss-org/goss), [man page](https://manpages.ubuntu.com/manpages/resolute/en/man1/goss.1.html))
- **osquery.** The OS as SQL tables: `crontab`, `systemd_units`, `listening_ports`, `docker_containers`. An inventory to discover a footprint, not to assert one. ([Facebook](https://engineering.fb.com/production-engineering/introducing-osquery/))
- **healthchecks.io.** Inverted: the job pings a URL when it succeeds; silence past the grace time is the alarm (a dead man's switch). Reality reports itself; nobody needs credentials to the host. ([docs](https://healthchecks.io/docs))
- **Swimm.** Docs coupled to code elements; a CI check fails when coupled code changed significantly, small changes are auto-synced. ([Swimm](https://docs.swimm.io/new-to-swimm/benefits-of-code-coupled-documentation))
- **Doc Detective.** "Docs as tests": runs the procedures written in docs, fails CI when they break. ([CI guide](https://mintlify.com/doc-detective/doc-detective/guides/ci-cd))
- **LLM doc-drift tools (2026).** Sentinel, DriftGuard, a Codex CLI doc-rot workflow: deterministic checks for hard facts (deleted files, renamed functions, links), an LLM only for semantic drift; a full sweep costs hundreds of thousands of tokens. ([Sentinel](https://github.com/jcentner/sentinel/wiki/Detector:-Semantic-Drift), [Codex doc-rot](https://codex.danielvaughan.com/2026/04/26/codex-cli-doc-rot-detection-automated-documentation-repair/))
- **Service catalogs** (Backstage, OpsLevel, Cortex, Port). Scorecards over catalog entities, fed by integrations; Backstage flags orphaned and stale entities. Heavy, team-scale, a server. ([Backstage](https://backstage.io/docs/features/software-catalog/configuration), [Roadie](https://roadie.io/changelog/warnings-on-orphaned-catalog-entities))
- **direnv / mise trust.** A repo's file may execute code only after the user approves its exact content on this machine; mise had trust-bypass CVEs in 2026, which shows how hard the gate is to get right. ([mise trust](https://docs.rs/crate/mise/2026.8.15/source/src/cli/trust.rs), [advisories](https://advisories.gitlab.com/pkg/cargo/mise/))
- **in-toto / SLSA Verification Summary.** A signed statement "verifier X checked artifact Y against policy Z: PASSED", with the evidence it used. Trust in the observation, not only in the result. ([SLSA VSA](https://slsa.dev/spec/v0.2/verification_summary), [Sigstore](https://docs.sigstore.dev/cosign/verifying/attestation/))

What carries over: **declared vs observed as two separate things** (Terraform, Argo); **generate the declaration from observation** (goss autoadd); **observe what is outside the declaration** (Terraform's blind spot is exactly our "absences"); **let reality report itself** (healthchecks); **deterministic first, LLM for meaning** (doc-drift tools); **never run a repo's commands without the user's approval** (direnv).

## The axes

Every variant below is a choice on three axes:
1. **Who observes:** the CLI, CI, an agent with the human's permissions, the deployed thing itself, other people.
2. **Where the checkable form lives:** prose only, typed lines in the brief, a separate file, nowhere (inferred from prose).
3. **What `check` verifies:** reality itself, or the *evidence* that someone observed reality, and how old it is.

The third axis matters most. `check` today is offline, deterministic and safe to run on any clone, in CI and in hooks. Running probes inside it breaks all three. Checking evidence keeps them.

## Variants, from obvious to unrealistic

### 1. Per-claim stamps (obvious)
Each line of "Where it lives" may end with when and how it was last observed: `· seen 2026-11-02 (readlink)`. `check` warns on stamps older than N days, by line, instead of one `verified:` for the whole brief. Nothing runs. It is the Wikipedia citation applied to infrastructure.
- For: no execution, no network, works for every class including ssh-only and absences; tells a reader which claims are fresh. Fits "the CLI checks shape, agents write meaning".
- Against: it checks that someone *says* they looked. An agent can stamp without looking. Noise on every line.

### 2. Typed local probes (obvious)
Optional typed lines the CLI understands, offline and without a shell: `path`, `symlink → target`, `file contains`, `git: tag/commit/remote`. Go standard library only.
- For: catches moves and broken installs (finding 12) mechanically; safe on any clone (it only reads files); covers the whole "local footprint" class in this pack.
- Against: the most important claims (prod, absences) are out of reach; a second syntax in the brief.

### 3. Public probes, opt-in (obvious)
`check --online`: read-only requests to public endpoints for typed claims: GitHub visibility and latest release, PyPI/npm/crates version, the Homebrew cask, a URL answers, a DNS name resolves. No secrets, no shell.
- For: the claims a stranger cares about; can run in CI weekly, so the badge means "the meta matches what the world sees", not only "fresh".
- Against: network in a tool that promised none (spec §11), rate limits, flaky CI; a narrow class.

### 4. Discovery: `omut observe` (medium, strong)
The reverse direction, after goss autoadd and Terraform's blind spot. Scan this machine for anything that points into a project: symlinks from `~/.claude/skills`, `~/.codex`, `~/.local/bin`; MCP configs of Claude Code, Codex, Cursor; crontab, launchd/systemd units; docker containers and compose files; virtualenvs. Diff with "Where it lives" both ways: declared but not found, found but not declared.
- For: the only variant that checks **absences** ("nothing else runs here") and finds the undeclared footprint; read-only, offline, no repo commands; turns "adopting" and "moving" from a hunt into a diff; for a pack it is one scan for all projects.
- Against: per-OS code (macOS launchd vs systemd), only this machine (a VPS is out of reach unless omut runs there too); matching prose claims to findings needs either typed lines or an agent.

### 5. Agent-run probes with receipts (medium)
Probe lines may say anything, ssh included (`ssh workhorse docker inspect wimt-agent`), but the CLI never runs them. `hook start` or `check` lists stale ones; the agent runs them under the human's permissions, as it already does in the audit ritual, and writes the observation back as a stamp (variant 1). `check` verifies the receipts' age.
- For: covers the private remote class; trust stays with the agent's permission system, where it already is; no credentials in omut.
- Against: it is variant 1 with better instructions; the truth still rests on the agent.

### 6. A trust gate, like direnv (medium, risky)
`check --probe` runs shell probes, but only after `omut trust` approved this exact probe set on this machine (hash kept outside the repo).
- For: fully automatic checks of private infrastructure from the human's own machine.
- Against: omut becomes a code-execution tool; mise's 2026 CVEs show the gate is easy to get wrong; a big step away from "a small CLI".

### 7. Reality reports itself (medium, inverted)
After healthchecks.io: deployed things push evidence instead of omut pulling it. A deploy step, a cron job or a CI release writes a small observation (`.omut/observed/<claim>`: what, when, by whom) or pings; `check` reads the age and warns when it goes silent.
- For: no credentials anywhere; absences of activity become visible (a dead cron); fits CI-heavy projects.
- Against: something has to write into the repo or a store, so it is integration work per project; machine-written files next to the meta blur "agents write meaning".

### 8. CI as the independent observer (medium)
The weekly `omut.yml` already exists. Give it the public probes (3) and the git facts (2), and the badge becomes "matches reality as the public sees it". Optionally a `workflow_dispatch` run with secrets for private probes in the project's own CI.
- For: visible from outside, independent of the agent that wrote the claim, zero setup beyond what the badge already needs.
- Against: only the public slice without secrets; a red badge from a flaky registry.

### 9. Audit lottery (bold, cheap)
At session start, `hook start` picks one or two claims with the oldest evidence and asks the agent to check them before it starts ("While you are here: verify that the zymi MCP server still points at ~/.local/bin/zymi"). Verification is amortised over normal work: every claim gets touched eventually, no session pays much.
- For: cheap; uses agents that are already in the project; works for every class the agent can reach; makes the audit continuous instead of a ritual nobody runs.
- Against: relies on the agent complying; an extra paragraph in every session's context.

### 10. Semantic audit by an LLM (bold)
Hybrid after the 2026 doc-drift tools: deterministic checks for facts, an agent for "Now" and "What it is" against the code, the README and recent commits. Run on demand (`omut audit` prints a prompt with the evidence) or in a scheduled agent.
- For: covers the part no probe can: "Now says it works", "What it is" after a refactor.
- Against: cost (hundreds of thousands of tokens per sweep), false positives, model-dependent; the CLI would only prepare context, an agent does the judging.

### 11. Export, don't run (bold, lazy in a good way)
Typed claims compile to the formats existing tools run: a `goss.yaml` per host, healthchecks.io checks, a Terraform data source, an osquery pack. omut declares; goss, InSpec or osquery observe; their results come back as stamps.
- For: no runtime of our own; mature checkers; users who have them already gain from it.
- Against: a dependency per class; few solo developers run goss.

### 12. Confidence decay (bold)
Each claim class has a half-life: a commit hash goes stale in days, a symlink in months, "Why" never. `check` and `hook start` rank claims by how likely they are to be wrong now, not by one 30-day threshold.
- For: honest about what the meta knows; focuses audits (feeds 9).
- Against: invented numbers; another concept to explain.

### 13. Signed observations (unrealistic for now)
After in-toto / SLSA VSA: an observation is signed by its observer: CI via OIDC and Sigstore, a human's key, an agent's key. A reader of a public `.omut/` can tell what was machine-verified, by whom, and what is just written.
- For: trust from outside, the property a public standard needs once strangers rely on other people's meta.
- Against: keys, infrastructure, ceremony; nobody needs it at the scale of one person.

### 14. Reconciliation (unrealistic, Argo/Nix)
"Where it lives" is desired state, and omut makes reality match: recreate the skill symlink after a move, re-register the MCP server.
- For: moves become free.
- Against: a tool that changes the user's machine from a markdown file; far outside "lays down templates and checks shape".

### 15. Reality writes the meta (unrealistic, inverted)
"Where it lives" is not written at all: it is generated from discovery (4) and receipts (7); agents write only "Why", "Now" and decisions.
- For: it cannot drift.
- Against: generated text is noise without meaning ("why does this cron exist?"); the human-facing value of the section is the explanation.

### 16. Witnesses across packs (unrealistic, network)
Other people's meta mentions your project ("depends on zymi-core 0.9.3 from PyPI"); each such mention is an independent observation. A public index reads `.omut/` of public repositories, runs the public probes on them and shows which meta matches the world.
- For: verification by many independent observers; makes truthful meta visible.
- Against: a service to build and run, privacy questions, gaming; only makes sense with many public `.omut/` directories.

### 17. The log as a regression suite (unrealistic)
Log lines already say "verified: how". Make the "how" replayable: CI re-runs past verifications at the commits they were made, and today, to show which old truths no longer hold.
- For: history becomes evidence.
- Against: most "how"s are not commands, and replaying is variant 6's security problem across all of history.

## How the variants fit together

They are not alternatives so much as layers:
- **Evidence format:** 1, with 12 as a refinement. Without it, nothing else has anywhere to report to.
- **Observers:** 2 and 4 (this machine, offline), 3 and 8 (the public, CI), 5 and 9 (agents, with the human's permissions), 7 (reality itself), 10 (meaning).
- **Trust:** none needed for 1–4, 7–9; the agent's own permission system for 5; a gate for 6; signatures for 13.

A first cut that keeps `check` offline, deterministic and safe on any clone could be: stamps (1) as the shared format; typed local probes (2) and discovery (4) as offline observers; public probes (3) only in CI (8); private infrastructure through agents (5, 9). Everything that executes commands from the repository (6, 17) or changes the machine (14) stays out. This note does not decide that; it is a starting point for D11.

With the two layers (D15) the first cut sorts itself:
- **Project layer:** shape and freshness as today, plus the public footprint checked in CI (3, 8). Runs nothing on a clone.
- **Instance layer:** stamps (1), typed local probes (2), discovery (4), and probes of private infrastructure the user wrote themselves, run on their machine (5, and 6 without a gate, from untracked files only). The audit lottery (9) draws from here.
- **Both:** `hook start` gives an agent the two layers; a `leak` rule keeps instance facts out of the project layer.

## Open questions

- D15: where the instance layer lives, whether it needs its own decisions, how it is backed up.
- Typed lines or inferred from prose? The pack's prose is already half-structured; typed lines are a second syntax in a file meant for people.
- Does discovery (4) belong in `check`, or in a separate `omut observe` that only prints?
- Is "the badge means matches reality" worth network in CI, given flaky registries?
- Where do observations live: on the line, in the brief's frontmatter, in a machine-written file next to the meta?
- What is the unit of verification for non-code projects (an article, a server, research)?
