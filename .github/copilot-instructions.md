# Copilot Instructions for gh-ghostty

## What this is

A `gh` CLI extension (Go, single-binary) that manages [Ghostty](https://ghostty.org) terminal themes. It reads and writes `~/.config/ghostty/config` and signals Ghostty to live-reload.

## Architecture

- **Single-file CLI** — all logic lives in `main.go` using [cobra](https://github.com/spf13/cobra) for commands.
- **Config format** — simple `key = value` lines with `#` comments. Theme is set via `theme = <value>` where value can be a name or `dark:<name>,light:<name>`.
- **Theme discovery** — `ghostty +list-themes` with a hardcoded fallback list.
- **Live reload** — simulates `Cmd+Shift+,` via AppleScript to trigger Ghostty's config reload on macOS.
- **Testability** — `configPathFunc` is a package-level var so tests can redirect I/O to temp dirs.

## Commands

| Command | Description |
|---------|-------------|
| `list` | List available Ghostty themes |
| `set <theme>` | Set theme (supports `--dark`/`--light` flags) |
| `random` | Pick and apply a random theme |
| `current` | Show the current theme |
| `pick` | Interactive fuzzy-search theme picker |
| `preview <theme>` | Apply theme temporarily, prompt to keep or revert |

## Build & Test

```bash
make build        # Build to bin/gh-ghostty
make test         # Run unit tests
make test-race    # Tests with -race and coverage
make ci           # Build + vet + test-race
make lint         # golangci-lint (if installed)
make fmt          # gofmt
make tidy         # go mod tidy
```

## Conventions

- Go 1.22+, single module, no internal packages.
- Tests go in `main_test.go` using table-driven style.
- Use `withTempConfig(t, content)` helper in tests to mock the config file path.
- Commands write to `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` for testability.
- Keep the fallback theme list in `fallbackThemes` if `ghostty +list-themes` is unavailable.
- Releases via GoReleaser (`.goreleaser.yml`), installed as a `gh` extension.

## Style

- Minimal comments — only where logic is non-obvious.
- No external dependencies beyond cobra and [survey](https://github.com/AlecAivazis/survey) (same prompt library used by `gh` CLI).
- Functions that manipulate config lines (`parseConfigLine`, `setThemeInLines`, `currentThemeFromLines`) are pure and easy to test.
