# omut

The intent lives in `docs/spec-v0.md`; the comparison with neighbours in `docs/research/competitors.md`.

- Go, standard library only. Templates are plain markdown in `templates/`, embedded into the binary.
- The CLI never writes meaning into the meta: it lays down templates and checks shape. Keep it that way.
- Changing a rule in `internal/check` means updating spec §8 and, if agents need to know, `templates/root/omut/conventions.md`.
- `go vet ./... && go test ./...` before calling anything done.

<!-- omut:start -->
## Project meta (omut)
At the start of a session read `.omut/brief.md`, the `open` and `doubted` decisions in `.omut/decisions.md` and the unfinished threads in `.omut/threads/`.
At the end of a session add a line to the top of `.omut/log.md` (`- YYYY-MM-DD · agent/model · what was done → link`), update "Now" in `.omut/brief.md`, and record unfinished work as a thread in `.omut/threads/`.
Full rules: `../.omut/conventions.md`, when the project sits in a pack.
<!-- omut:end -->
