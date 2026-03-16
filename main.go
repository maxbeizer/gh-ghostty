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
	"strconv"
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
		Short: "Manage Ghostty terminal configuration",
		Long:  "Manage Ghostty themes and configuration directly from the gh CLI.",
	}

	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(setCmd())
	rootCmd.AddCommand(randomCmd())
	rootCmd.AddCommand(currentCmd())
	rootCmd.AddCommand(previewCmd())
	rootCmd.AddCommand(pickCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(fontSizeCmd())
	rootCmd.AddCommand(fontFamilyCmd())
	rootCmd.AddCommand(cursorStyleCmd())
	rootCmd.AddCommand(backgroundOpacityCmd())

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

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set Ghostty configuration options",
		Long:  "Read and write arbitrary Ghostty configuration options in ~/.config/ghostty/config.",
	}

	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	return cmd
}

func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			value, ok := getConfigFromLines(lines, args[0])
			if !ok {
				fmt.Fprintf(cmd.OutOrStdout(), "(%s not set)\n", args[0])
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			updated := setConfigInLines(lines, args[0], args[1])
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s to %s\n", args[0], args[1])
			return nil
		},
	}
}

func fontSizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "font-size [size]",
		Short: "Get or set the font size",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				value, ok := getConfigFromLines(lines, "font-size")
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "(font-size not set)")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), value)
				}
				return nil
			}
			size := args[0]
			if _, err := strconv.ParseFloat(size, 64); err != nil {
				return fmt.Errorf("invalid font size %q: must be a number", size)
			}
			updated := setConfigInLines(lines, "font-size", size)
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set font-size to %s\n", size)
			return nil
		},
	}
}

func fontFamilyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "font-family [name]",
		Short: "Get or set the font family",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				value, ok := getConfigFromLines(lines, "font-family")
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "(font-family not set)")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), value)
				}
				return nil
			}
			updated := setConfigInLines(lines, "font-family", args[0])
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set font-family to %s\n", args[0])
			return nil
		},
	}
}

func cursorStyleCmd() *cobra.Command {
	validStyles := []string{"block", "bar", "underline", "block_hollow"}

	return &cobra.Command{
		Use:   "cursor-style [style]",
		Short: "Get or set the cursor style",
		Long:  "Get or set the cursor style. Valid styles: block, bar, underline, block_hollow.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				value, ok := getConfigFromLines(lines, "cursor-style")
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "(cursor-style not set)")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), value)
				}
				return nil
			}
			style := args[0]
			valid := false
			for _, s := range validStyles {
				if s == style {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid cursor style %q: must be one of %s", style, strings.Join(validStyles, ", "))
			}
			updated := setConfigInLines(lines, "cursor-style", style)
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set cursor-style to %s\n", style)
			return nil
		},
	}
}

func backgroundOpacityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "background-opacity [value]",
		Short: "Get or set the background opacity",
		Long:  "Get or set the background opacity (0.0 to 1.0).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := readConfigLines()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				value, ok := getConfigFromLines(lines, "background-opacity")
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "(background-opacity not set)")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), value)
				}
				return nil
			}
			val, err := strconv.ParseFloat(args[0], 64)
			if err != nil || val < 0 || val > 1 {
				return fmt.Errorf("invalid background-opacity %q: must be a number between 0.0 and 1.0", args[0])
			}
			updated := setConfigInLines(lines, "background-opacity", args[0])
			if err := writeConfigLines(updated); err != nil {
				return err
			}
			reloadGhostty(cmd)
			fmt.Fprintf(cmd.OutOrStdout(), "Set background-opacity to %s\n", args[0])
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

func getConfigFromLines(lines []string, targetKey string) (string, bool) {
	for _, line := range lines {
		key, value, ok := parseConfigLine(line)
		if ok && key == targetKey {
			return value, true
		}
	}
	return "", false
}

func setConfigInLines(lines []string, targetKey, value string) []string {
	for i, line := range lines {
		key, _, ok := parseConfigLine(line)
		if ok && key == targetKey {
			lines[i] = fmt.Sprintf("%s = %s", targetKey, value)
			return lines
		}
	}
	return append(lines, fmt.Sprintf("%s = %s", targetKey, value))
}

func currentThemeFromLines(lines []string) (string, bool) {
	return getConfigFromLines(lines, "theme")
}

func setThemeInLines(lines []string, value string) []string {
	return setConfigInLines(lines, "theme", value)
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
