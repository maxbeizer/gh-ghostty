# gh-ghostty

A `gh` CLI extension to manage Ghostty terminal themes and configuration.

## Install

```bash
gh extension install maxbeizer/gh-ghostty
```

## Usage

### Theme management

```bash
gh ghostty list                              # List available themes
gh ghostty set <theme>                       # Set theme in config
gh ghostty set --dark <theme> --light <theme> # Separate light/dark
gh ghostty random                            # Random theme
gh ghostty current                           # Show current theme
gh ghostty preview <theme>                   # Set temporarily, prompt keep/revert
gh ghostty pick                              # Interactive fuzzy search and select
```

### Configuration

```bash
gh ghostty config get <key>                  # Get any config value
gh ghostty config set <key> <value>          # Set any config value
```

### Convenience commands

```bash
gh ghostty font-size [size]                  # Get or set font size
gh ghostty font-family [name]               # Get or set font family
gh ghostty cursor-style [style]             # Get or set cursor style (block, bar, underline, block_hollow)
gh ghostty background-opacity [value]       # Get or set background opacity (0.0–1.0)
```

Convenience commands with no argument print the current value. With an argument, they set the value and reload Ghostty.

## How it works

- Reads/writes Ghostty config at `~/.config/ghostty/config`
- Updates `key = value` lines (e.g. `theme = ...`, `font-size = ...`)
- Triggers Ghostty config reload via AppleScript on macOS after changes
- `list` pulls from `ghostty +list-themes` and falls back to a bundled list

## Notes

- Config reload uses AppleScript on macOS to simulate `Cmd+Shift+,`. If Ghostty isn't running or focused, you may need to reload manually.
- The config format is simple `key = value` lines with `#` comments.
- For a full list of Ghostty config options, see the [Ghostty config reference](https://ghostty.org/docs/config/reference).

## Development

```bash
make build        # Build to bin/gh-ghostty
make test         # Run unit tests
make ci           # Build + vet + test with race detector
```

## License

[MIT](LICENSE)
