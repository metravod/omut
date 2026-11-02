# omut

The intent lives in `docs/spec-v0.md`; the comparison with neighbours in `docs/research/competitors.md`.

- Go, standard library only. Templates are plain markdown in `templates/`, embedded into the binary.
- The CLI never writes meaning into the meta: it lays down templates and checks shape. Keep it that way.
- Changing a rule in `internal/check` means updating spec §8 and, if agents need to know, `templates/project/omut/README.md`.
- `go vet ./... && go test ./...` before calling anything done.

<!-- omut:start -->
## Project meta (omut)
`.omut/` holds this project's meta: why it exists, what state it is in, what comes next. To understand the project, read `.omut/brief.md` first, then the decisions in `.omut/decisions.md`.
If you work on the project, keep the meta:
- at the start of a session, read `.omut/brief.md`, the `open` and `doubted` decisions in `.omut/decisions.md` and the unfinished threads in `.omut/threads/`;
- at the end, add a line to the top of `.omut/log.md` (`- YYYY-MM-DD · agent/model · what was done → link`), update "Now" in `.omut/brief.md`, and record unfinished work as a thread in `.omut/threads/`.
Full rules: `.omut/README.md`.
<!-- omut:end -->
