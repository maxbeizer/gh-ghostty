# gh-ghostty

A `gh` CLI extension to switch Ghostty terminal themes quickly.

## Install

```bash
gh extension install maxbeizer/gh-ghostty
```

## Usage

```bash
gh ghostty list                              # List available themes
gh ghostty set <theme>                       # Set theme in config
gh ghostty set --dark <theme> --light <theme> # Separate light/dark
gh ghostty random                            # Random theme
gh ghostty current                           # Show current theme
gh ghostty preview <theme>                   # Set temporarily, prompt keep/revert
gh ghostty pick                              # Interactive fuzzy search and select
```

## How it works

- Reads/writes Ghostty config at `~/.config/ghostty/config`
- Updates `theme = ...` or `theme = dark:Name,light:Name`
- Triggers Ghostty config reload via AppleScript on macOS after changes
- `list` pulls from `ghostty +list-themes` and falls back to a bundled list

## Notes

- Config reload uses AppleScript on macOS to simulate `Cmd+Shift+,`. If Ghostty isn't running or focused, you may need to reload manually.
- The config format is simple `key = value` lines with `#` comments.

## Development

```bash
make build        # Build to bin/gh-ghostty
make test         # Run unit tests
make ci           # Build + vet + test with race detector
```

## License

[MIT](LICENSE)
