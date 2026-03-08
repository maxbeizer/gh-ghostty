package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

const (
	configDirName  = "ghostty"
	configFileName = "config"
)

var fallbackThemes = []string{
	"Dracula",
	"Solarized Dark",
	"Solarized Light",
	"Gruvbox Dark",
	"Gruvbox Light",
	"Nord",
	"One Dark",
	"One Light",
	"Catppuccin Mocha",
	"Catppuccin Latte",
	"Tokyo Night",
	"Monokai",
	"GitHub Dark",
	"GitHub Light",
	"Night Owl",
	"Pencil",
	"PaperColor Dark",
	"PaperColor Light",
	"Material",
	"Sakura",
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	rootCmd := &cobra.Command{
		Use:   "gh-ghostty",
		Short: "Ghostty theme switcher",
		Long:  "Manage Ghostty themes directly from the gh CLI.",
	}

	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(setCmd())
	rootCmd.AddCommand(randomCmd())
	rootCmd.AddCommand(currentCmd())
	rootCmd.AddCommand(previewCmd())
	rootCmd.AddCommand(pickCmd())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available Ghostty themes",
		RunE: func(cmd *cobra.Command, args []string) error {
			themes, err := listThemes()
			if err != nil {
				return err
			}
			for _, theme := range themes {
				fmt.Fprintln(cmd.OutOrStdout(), theme)
			}
			return nil
		},
	}
}

func setCmd() *cobra.Command {
	var dark string
	var light string

	cmd := &cobra.Command{
		Use:   "set <theme>",
		Short: "Set the Ghostty theme",
		Args: func(cmd *cobra.Command, args []string) error {
			if dark != "" || light != "" {
				if dark == "" || light == "" {
					return errors.New("both --dark and --light must be provided together")
				}
				if len(args) > 0 {
					return errors.New("theme argument not allowed when using --dark/--light")
				}
				return nil
			}
			if len(args) != 1 {
				return errors.New("provide a theme name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var value string
			if dark != "" && light != "" {
				value = fmt.Sprintf("dark:%s,light:%s", dark, light)
			} else {
				value = args[0]
			}

			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			updated := setThemeInLines(lines, value)
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set theme to %s\n", value)
			return nil
		},
	}

	cmd.Flags().StringVar(&dark, "dark", "", "Set dark theme")
	cmd.Flags().StringVar(&light, "light", "", "Set light theme")
	return cmd
}

func randomCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "random",
		Short: "Pick a random Ghostty theme",
		RunE: func(cmd *cobra.Command, args []string) error {
			themes, err := listThemes()
			if err != nil {
				return err
			}
			if len(themes) == 0 {
				return errors.New("no themes available")
			}
			choice := themes[rand.Intn(len(themes))]

			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			updated := setThemeInLines(lines, choice)
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set theme to %s\n", choice)
			return nil
		},
	}
}

func currentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the current Ghostty theme",
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			value, ok := currentThemeFromLines(lines)
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "(theme not set)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}

func previewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "preview <theme>",
		Short: "Preview a theme and keep or revert",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			original := append([]string(nil), lines...)

			updated := setThemeInLines(lines, args[0])
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)

			fmt.Fprintf(cmd.OutOrStdout(), "Previewing theme: %s\n", args[0])
			fmt.Fprint(cmd.OutOrStdout(), "Keep this theme? [y/N]: ")
			reader := bufio.NewReader(cmd.InOrStdin())
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response == "y" || response == "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Theme kept.")
				return nil
			}

			if err := writeConfigLines(original); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintln(cmd.OutOrStdout(), "Reverted to previous theme.")
			return nil
		},
	}
}

func pickCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pick",
		Short: "Interactively search and select a theme",
		RunE: func(cmd *cobra.Command, args []string) error {
			themes, err := listThemes()
			if err != nil {
				return err
			}
			if len(themes) == 0 {
				return errors.New("no themes available")
			}

			var choice string
			prompt := &survey.Select{
				Message: "Pick a theme:",
				Options: themes,
			}
			if err := survey.AskOne(prompt, &choice); err != nil {
				return err
			}

			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			updated := setThemeInLines(lines, choice)
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set theme to %s\n", choice)
			return nil
		},
	}
}

func listThemes() ([]string, error) {
	cmd := exec.Command("ghostty", "+list-themes")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		var themes []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				themes = append(themes, stripThemeSuffix(line))
			}
		}
		if len(themes) > 0 {
			return themes, nil
		}
	}
	return fallbackThemes, nil
}

// stripThemeSuffix removes the " (resources)" or similar parenthetical suffix
// that ghostty +list-themes appends to theme names.
func stripThemeSuffix(name string) string {
	if idx := strings.LastIndex(name, " ("); idx >= 0 && strings.HasSuffix(name, ")") {
		return name[:idx]
	}
	return name
}

func reloadGhostty(cmd *cobra.Command) {
	reload := exec.Command("osascript", "-e",
		`tell application "System Events" to keystroke "," using {command down, shift down}`)
	if err := reload.Run(); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Could not trigger Ghostty config reload.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Press Cmd+Shift+, in Ghostty to reload manually.")
	}
}

// configPathFunc is the function used to resolve the Ghostty config path.
// Override in tests to use a temp directory.
var configPathFunc = defaultConfigPath

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDirName, configFileName), nil
}

func configPath() (string, error) {
	return configPathFunc()
}

func readConfigLines() ([]string, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	return lines, nil
}

func writeConfigLines(lines []string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func currentThemeFromLines(lines []string) (string, bool) {
	for _, line := range lines {
		key, value, ok := parseConfigLine(line)
		if ok && key == "theme" {
			return value, true
		}
	}
	return "", false
}

func setThemeInLines(lines []string, value string) []string {
	for i, line := range lines {
		key, _, ok := parseConfigLine(line)
		if ok && key == "theme" {
			lines[i] = fmt.Sprintf("theme = %s", value)
			return lines
		}
	}
	return append(lines, fmt.Sprintf("theme = %s", value))
}

func parseConfigLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	if strings.Contains(trimmed, "#") {
		trimmed = strings.TrimSpace(strings.SplitN(trimmed, "#", 2)[0])
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return "", "", false
	}
	return key, value, true
}
