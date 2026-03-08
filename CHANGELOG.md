# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
