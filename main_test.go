package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseConfigLine
// ---------------------------------------------------------------------------

func TestParseConfigLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{"simple key=value", "theme = Dracula", "theme", "Dracula", true},
		{"no spaces around eq", "theme=Dracula", "theme", "Dracula", true},
		{"extra whitespace", "  theme  =  Dracula  ", "theme", "Dracula", true},
		{"comment line", "# theme = Dracula", "", "", false},
		{"blank line", "", "", "", false},
		{"whitespace only", "   ", "", "", false},
		{"inline comment", "theme = Dracula # nice", "theme", "Dracula", true},
		{"no equals", "just-a-word", "", "", false},
		{"empty value", "theme = ", "", "", false},
		{"empty key", " = Dracula", "", "", false},
		{"dark/light value", "theme = dark:Dracula,light:Solarized Light", "theme", "dark:Dracula,light:Solarized Light", true},
		{"other config key", "font-size = 14", "font-size", "14", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val, ok := parseConfigLine(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if val != tt.wantVal {
				t.Errorf("val = %q, want %q", val, tt.wantVal)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// setThemeInLines
// ---------------------------------------------------------------------------

func TestSetThemeInLines(t *testing.T) {
	t.Run("replaces existing theme line", func(t *testing.T) {
		lines := []string{"font-size = 14", "theme = Nord", "cursor-style = block"}
		got := setThemeInLines(lines, "Dracula")
		if got[1] != "theme = Dracula" {
			t.Errorf("got %q, want %q", got[1], "theme = Dracula")
		}
		if len(got) != 3 {
			t.Errorf("expected 3 lines, got %d", len(got))
		}
	})

	t.Run("appends when no theme exists", func(t *testing.T) {
		lines := []string{"font-size = 14"}
		got := setThemeInLines(lines, "Nord")
		if len(got) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(got))
		}
		if got[1] != "theme = Nord" {
			t.Errorf("got %q, want %q", got[1], "theme = Nord")
		}
	})

	t.Run("handles dark/light combo", func(t *testing.T) {
		lines := []string{"theme = Nord"}
		got := setThemeInLines(lines, "dark:Dracula,light:Solarized Light")
		if got[0] != "theme = dark:Dracula,light:Solarized Light" {
			t.Errorf("got %q", got[0])
		}
	})

	t.Run("empty lines slice", func(t *testing.T) {
		got := setThemeInLines([]string{}, "Nord")
		if len(got) != 1 || got[0] != "theme = Nord" {
			t.Errorf("unexpected result: %v", got)
		}
	})
}

// ---------------------------------------------------------------------------
// currentThemeFromLines
// ---------------------------------------------------------------------------

func TestCurrentThemeFromLines(t *testing.T) {
	t.Run("finds theme", func(t *testing.T) {
		lines := []string{"font-size = 14", "theme = Dracula", "cursor-style = block"}
		val, ok := currentThemeFromLines(lines)
		if !ok || val != "Dracula" {
			t.Errorf("got (%q, %v), want (Dracula, true)", val, ok)
		}
	})

	t.Run("no theme set", func(t *testing.T) {
		lines := []string{"font-size = 14", "cursor-style = block"}
		_, ok := currentThemeFromLines(lines)
		if ok {
			t.Error("expected ok=false when no theme is set")
		}
	})

	t.Run("comment theme not counted", func(t *testing.T) {
		lines := []string{"# theme = Dracula", "font-size = 14"}
		_, ok := currentThemeFromLines(lines)
		if ok {
			t.Error("expected ok=false when theme is commented out")
		}
	})

	t.Run("empty lines", func(t *testing.T) {
		_, ok := currentThemeFromLines([]string{})
		if ok {
			t.Error("expected ok=false for empty lines")
		}
	})
}

// ---------------------------------------------------------------------------
// readConfigLines / writeConfigLines (using temp dir)
// ---------------------------------------------------------------------------

func withTempConfig(t *testing.T, content string) func() {
	t.Helper()
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, configFileName)

	if content != "" {
		if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	original := configPathFunc
	configPathFunc = func() (string, error) {
		return configFile, nil
	}

	return func() {
		configPathFunc = original
	}
}

func TestReadConfigLines(t *testing.T) {
	t.Run("reads existing file", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-size = 14\ntheme = Dracula\n")
		defer cleanup()

		lines, err := readConfigLines()
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got %d", len(lines))
		}
		if lines[1] != "theme = Dracula" {
			t.Errorf("got %q", lines[1])
		}
	})

	t.Run("returns empty for missing file", func(t *testing.T) {
		cleanup := withTempConfig(t, "")
		defer cleanup()
		// Remove the file so it doesn't exist
		p, _ := configPathFunc()
		os.Remove(p)

		lines, err := readConfigLines()
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 0 {
			t.Errorf("expected empty lines, got %d", len(lines))
		}
	})

	t.Run("handles CRLF", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-size = 14\r\ntheme = Dracula\r\n")
		defer cleanup()

		lines, err := readConfigLines()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lines {
			if strings.Contains(l, "\r") {
				t.Error("CRLF not normalized")
			}
		}
	})
}

func TestWriteConfigLines(t *testing.T) {
	t.Run("creates dirs and writes", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "sub", "dir", configFileName)
		original := configPathFunc
		configPathFunc = func() (string, error) {
			return configFile, nil
		}
		defer func() { configPathFunc = original }()

		err := writeConfigLines([]string{"theme = Nord", "font-size = 14"})
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(configFile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(string(data), "\n") {
			t.Error("file should end with newline")
		}
		if !strings.Contains(string(data), "theme = Nord") {
			t.Error("missing theme line")
		}
	})
}

// ---------------------------------------------------------------------------
// stripThemeSuffix
// ---------------------------------------------------------------------------

func TestStripThemeSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Dracula (resources)", "Dracula"},
		{"Spacegray Eighties Dull (resources)", "Spacegray Eighties Dull"},
		{"0x96f (resources)", "0x96f"},
		{"Dracula", "Dracula"},
		{"Nord (custom)", "Nord"},
		{"", ""},
		{"(resources)", "(resources)"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripThemeSuffix(tt.input)
			if got != tt.want {
				t.Errorf("stripThemeSuffix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Roundtrip: read → set → write → read
// ---------------------------------------------------------------------------

func TestRoundtrip(t *testing.T) {
	cleanup := withTempConfig(t, "font-size = 14\ntheme = Nord\ncursor-style = block\n")
	defer cleanup()

	lines, err := readConfigLines()
	if err != nil {
		t.Fatal(err)
	}

	updated := setThemeInLines(lines, "Dracula")
	if err := writeConfigLines(updated); err != nil {
		t.Fatal(err)
	}

	lines2, err := readConfigLines()
	if err != nil {
		t.Fatal(err)
	}

	val, ok := currentThemeFromLines(lines2)
	if !ok || val != "Dracula" {
		t.Errorf("roundtrip failed: got (%q, %v)", val, ok)
	}
}

// ---------------------------------------------------------------------------
// Command-level tests
// ---------------------------------------------------------------------------

func TestCurrentCommand(t *testing.T) {
	cleanup := withTempConfig(t, "theme = Tokyo Night\n")
	defer cleanup()

	cmd := currentCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "Tokyo Night" {
		t.Errorf("got %q, want %q", out, "Tokyo Night")
	}
}

func TestCurrentCommand_NoTheme(t *testing.T) {
	cleanup := withTempConfig(t, "font-size = 14\n")
	defer cleanup()

	cmd := currentCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "(theme not set)" {
		t.Errorf("got %q, want %q", out, "(theme not set)")
	}
}

func TestSetCommand(t *testing.T) {
	cleanup := withTempConfig(t, "font-size = 14\n")
	defer cleanup()

	cmd := setCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"Dracula"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	// Verify file was updated
	lines, err := readConfigLines()
	if err != nil {
		t.Fatal(err)
	}
	val, ok := currentThemeFromLines(lines)
	if !ok || val != "Dracula" {
		t.Errorf("theme not set: got (%q, %v)", val, ok)
	}
}

func TestSetCommand_DarkLight(t *testing.T) {
	cleanup := withTempConfig(t, "font-size = 14\n")
	defer cleanup()

	cmd := setCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--dark", "Dracula", "--light", "Solarized Light"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	lines, err := readConfigLines()
	if err != nil {
		t.Fatal(err)
	}
	val, ok := currentThemeFromLines(lines)
	if !ok || val != "dark:Dracula,light:Solarized Light" {
		t.Errorf("got (%q, %v)", val, ok)
	}
}

func TestSetCommand_Validation(t *testing.T) {
	t.Run("requires theme arg", func(t *testing.T) {
		cmd := setCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{})
		if err := cmd.Execute(); err == nil {
			t.Error("expected error with no args")
		}
	})

	t.Run("dark without light fails", func(t *testing.T) {
		cmd := setCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"--dark", "Dracula"})
		if err := cmd.Execute(); err == nil {
			t.Error("expected error with --dark but no --light")
		}
	})

	t.Run("arg with dark/light fails", func(t *testing.T) {
		cmd := setCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"--dark", "Dracula", "--light", "Nord", "Extra"})
		if err := cmd.Execute(); err == nil {
			t.Error("expected error with positional arg and --dark/--light")
		}
	})
}

// ---------------------------------------------------------------------------
// Generic config helpers
// ---------------------------------------------------------------------------

func TestGetConfigFromLines(t *testing.T) {
	lines := []string{"font-size = 14", "theme = Dracula", "cursor-style = block"}

	tests := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{"font-size", "14", true},
		{"theme", "Dracula", true},
		{"cursor-style", "block", true},
		{"missing-key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, ok := getConfigFromLines(lines, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if val != tt.want {
				t.Errorf("val = %q, want %q", val, tt.want)
			}
		})
	}
}

func TestSetConfigInLines(t *testing.T) {
	t.Run("replaces existing key", func(t *testing.T) {
		lines := []string{"font-size = 14", "cursor-style = block"}
		got := setConfigInLines(lines, "font-size", "16")
		if got[0] != "font-size = 16" {
			t.Errorf("got %q, want %q", got[0], "font-size = 16")
		}
		if len(got) != 2 {
			t.Errorf("expected 2 lines, got %d", len(got))
		}
	})

	t.Run("appends when key not present", func(t *testing.T) {
		lines := []string{"theme = Nord"}
		got := setConfigInLines(lines, "font-size", "14")
		if len(got) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(got))
		}
		if got[1] != "font-size = 14" {
			t.Errorf("got %q, want %q", got[1], "font-size = 14")
		}
	})

	t.Run("empty lines slice", func(t *testing.T) {
		got := setConfigInLines([]string{}, "background-opacity", "0.8")
		if len(got) != 1 || got[0] != "background-opacity = 0.8" {
			t.Errorf("unexpected result: %v", got)
		}
	})
}

// ---------------------------------------------------------------------------
// config get / config set commands
// ---------------------------------------------------------------------------

func TestConfigGetCommand(t *testing.T) {
	t.Run("gets existing value", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-size = 14\ncursor-style = block\n")
		defer cleanup()

		cmd := configCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"get", "font-size"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "14" {
			t.Errorf("got %q, want %q", strings.TrimSpace(buf.String()), "14")
		}
	})

	t.Run("shows not set for missing key", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-size = 14\n")
		defer cleanup()

		cmd := configCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"get", "cursor-style"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "not set") {
			t.Errorf("expected 'not set' message, got %q", buf.String())
		}
	})
}

func TestConfigSetCommand(t *testing.T) {
	cleanup := withTempConfig(t, "font-size = 14\n")
	defer cleanup()

	cmd := configCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"set", "cursor-style", "bar"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	lines, err := readConfigLines()
	if err != nil {
		t.Fatal(err)
	}
	val, ok := getConfigFromLines(lines, "cursor-style")
	if !ok || val != "bar" {
		t.Errorf("config not set: got (%q, %v)", val, ok)
	}
}

// ---------------------------------------------------------------------------
// Convenience commands
// ---------------------------------------------------------------------------

func TestFontSizeCommand(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-size = 14\n")
		defer cleanup()

		cmd := fontSizeCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "14" {
			t.Errorf("got %q, want %q", strings.TrimSpace(buf.String()), "14")
		}
	})

	t.Run("get not set", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := fontSizeCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "not set") {
			t.Errorf("expected 'not set', got %q", buf.String())
		}
	})

	t.Run("set valid", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := fontSizeCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"16"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		lines, _ := readConfigLines()
		val, ok := getConfigFromLines(lines, "font-size")
		if !ok || val != "16" {
			t.Errorf("got (%q, %v)", val, ok)
		}
	})

	t.Run("set float", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := fontSizeCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"13.5"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		lines, _ := readConfigLines()
		val, ok := getConfigFromLines(lines, "font-size")
		if !ok || val != "13.5" {
			t.Errorf("got (%q, %v)", val, ok)
		}
	})

	t.Run("set invalid", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := fontSizeCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"abc"})

		if err := cmd.Execute(); err == nil {
			t.Error("expected error for non-numeric font size")
		}
	})
}

func TestFontFamilyCommand(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		cleanup := withTempConfig(t, "font-family = JetBrains Mono\n")
		defer cleanup()

		cmd := fontFamilyCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "JetBrains Mono" {
			t.Errorf("got %q, want %q", strings.TrimSpace(buf.String()), "JetBrains Mono")
		}
	})

	t.Run("set", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := fontFamilyCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"Fira Code"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		lines, _ := readConfigLines()
		val, ok := getConfigFromLines(lines, "font-family")
		if !ok || val != "Fira Code" {
			t.Errorf("got (%q, %v)", val, ok)
		}
	})
}

func TestCursorStyleCommand(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		cleanup := withTempConfig(t, "cursor-style = block\n")
		defer cleanup()

		cmd := cursorStyleCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "block" {
			t.Errorf("got %q, want %q", strings.TrimSpace(buf.String()), "block")
		}
	})

	t.Run("set valid styles", func(t *testing.T) {
		for _, style := range []string{"block", "bar", "underline", "block_hollow"} {
			t.Run(style, func(t *testing.T) {
				cleanup := withTempConfig(t, "theme = Nord\n")
				defer cleanup()

				cmd := cursorStyleCmd()
				buf := new(bytes.Buffer)
				cmd.SetOut(buf)
				cmd.SetErr(new(bytes.Buffer))
				cmd.SetArgs([]string{style})

				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				lines, _ := readConfigLines()
				val, ok := getConfigFromLines(lines, "cursor-style")
				if !ok || val != style {
					t.Errorf("got (%q, %v)", val, ok)
				}
			})
		}
	})

	t.Run("set invalid style", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := cursorStyleCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"invalid"})

		if err := cmd.Execute(); err == nil {
			t.Error("expected error for invalid cursor style")
		}
	})
}

func TestBackgroundOpacityCommand(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		cleanup := withTempConfig(t, "background-opacity = 0.8\n")
		defer cleanup()

		cmd := backgroundOpacityCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "0.8" {
			t.Errorf("got %q, want %q", strings.TrimSpace(buf.String()), "0.8")
		}
	})

	t.Run("set valid", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := backgroundOpacityCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"0.5"})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		lines, _ := readConfigLines()
		val, ok := getConfigFromLines(lines, "background-opacity")
		if !ok || val != "0.5" {
			t.Errorf("got (%q, %v)", val, ok)
		}
	})

	t.Run("set out of range", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := backgroundOpacityCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"1.5"})

		if err := cmd.Execute(); err == nil {
			t.Error("expected error for opacity > 1.0")
		}
	})

	t.Run("set negative", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := backgroundOpacityCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"-0.1"})

		if err := cmd.Execute(); err == nil {
			t.Error("expected error for negative opacity")
		}
	})

	t.Run("set non-numeric", func(t *testing.T) {
		cleanup := withTempConfig(t, "theme = Nord\n")
		defer cleanup()

		cmd := backgroundOpacityCmd()
		cmd.SetOut(new(bytes.Buffer))
		cmd.SetErr(new(bytes.Buffer))
		cmd.SetArgs([]string{"abc"})

		if err := cmd.Execute(); err == nil {
			t.Error("expected error for non-numeric opacity")
		}
	})
}
