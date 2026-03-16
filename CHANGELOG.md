# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v0.2.0] - 2026-03-16

### Added
- `config get <key>` — Read any Ghostty config value
- `config set <key> <value>` — Write any Ghostty config value and reload
- `font-size [size]` — Get/set font size (validates numeric input)
- `font-family [name]` — Get/set font family
- `cursor-style [style]` — Get/set cursor style (validates: block, bar, underline, block_hollow)
- `background-opacity [value]` — Get/set background opacity (validates 0.0–1.0)
- Generic `setConfigInLines`/`getConfigFromLines` helpers; theme functions now delegate to them

## [v0.1.0] - 2026-03-08

### Added
- `list` — List available Ghostty themes
- `set` — Set theme with optional `--dark`/`--light` flags
- `pick` — Interactive fuzzy-search theme picker
- `random` — Pick and apply a random theme
- `current` — Show the current theme
- `preview` — Preview a theme temporarily, prompt to keep or revert
- Config reload via AppleScript on macOS after theme changes
- Fallback theme list when `ghostty +list-themes` is unavailable
