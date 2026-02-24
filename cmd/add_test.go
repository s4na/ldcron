package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateInlineScript_EmptyScript(t *testing.T) {
	err := validateInlineScript("")
	if err == nil {
		t.Error("expected error for empty script, got nil")
	}
}

func TestValidateInlineScript_NullByte(t *testing.T) {
	// Null bytes are not allowed in XML 1.0 and must be rejected before
	// the script is stored in a plist.
	tests := []struct {
		name   string
		script string
	}{
		{"null_only", "\x00"},
		{"null_in_middle", "echo hello\x00world"},
		{"null_at_end", "echo hello\x00"},
		{"multiple_nulls", "\x00\x00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInlineScript(tc.script)
			if err == nil {
				t.Errorf("expected error for script with null byte, got nil")
			}
			if !strings.Contains(err.Error(), "null byte") {
				t.Errorf("error should mention 'null byte', got: %v", err)
			}
		})
	}
}

func TestValidateInlineScript_NullByteErrorPosition(t *testing.T) {
	// The error message must report the exact byte position of the null byte.
	tests := []struct {
		name    string
		script  string
		wantPos int
	}{
		{"null_at_zero", "\x00abc", 0},
		{"null_at_five", "hello\x00world", 5},
		{"null_at_ten", "0123456789\x00", 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInlineScript(tc.script)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			want := fmt.Sprintf("position %d", tc.wantPos)
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should contain %q", err.Error(), want)
			}
		})
	}
}

func TestValidateInlineScript_XMLForbiddenControlChars(t *testing.T) {
	// XML 1.0 forbids U+0001–U+0008, U+000B–U+000C, and U+000E–U+001F.
	// Go's xml encoder silently replaces these with U+FFFD, which would
	// corrupt the plist without any visible error. They must be rejected.
	tests := []struct {
		name string
		ch   rune
	}{
		// U+0001–U+0008
		{"U+0001_SOH", '\x01'},
		{"U+0002_STX", '\x02'},
		{"U+0003_ETX", '\x03'},
		{"U+0004_EOT", '\x04'},
		{"U+0005_ENQ", '\x05'},
		{"U+0006_ACK", '\x06'},
		{"U+0007_BEL", '\x07'},
		{"U+0008_BS", '\x08'},
		// U+000B–U+000C
		{"U+000B_VT", '\x0B'},
		{"U+000C_FF", '\x0C'},
		// U+000E–U+001F (sample)
		{"U+000E_SO", '\x0E'},
		{"U+000F_SI", '\x0F'},
		{"U+001B_ESC", '\x1B'},
		{"U+001F_US", '\x1F'},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			script := "echo start" + string(tc.ch) + "end"
			err := validateInlineScript(script)
			if err == nil {
				t.Errorf("expected error for U+%04X in script, got nil", tc.ch)
			}
			if err != nil && !strings.Contains(err.Error(), "control character") {
				t.Errorf("error should mention 'control character': %v", err)
			}
		})
	}
}

func TestValidateInlineScript_AllowedControlChars(t *testing.T) {
	// Tab (U+0009), newline (U+000A), and carriage return (U+000D) are
	// the only control characters permitted by XML 1.0 and must pass validation.
	tests := []struct {
		name   string
		script string
	}{
		{"tab", "echo\thello"},
		{"newline", "echo hello\necho world"},
		{"carriage_return", "echo hello\recho world"},
		{"crlf", "echo hello\r\necho world"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInlineScript(tc.script); err != nil {
				t.Errorf("unexpected error for %q: %v", tc.name, err)
			}
		})
	}
}

func TestValidateInlineScript_ValidScripts(t *testing.T) {
	// These scripts must pass validation without error.
	tests := []struct {
		name   string
		script string
	}{
		{"simple_command", "echo hello"},
		{"multiline", "echo hello\ndate\necho world"},
		{"multiline_blank_lines", "echo start\n\necho end"},
		{"xml_special_chars", `echo "a < b" && echo "c > d" & echo "e & f"`},
		{"tab_chars", "echo\thello\tworld"},
		{"unicode", "echo '日本語テスト'"},
		{"backslashes", `find /tmp -name "*.log" -exec rm {} \;`},
		{"single_quotes", `echo 'it'\''s fine'`},
		{"env_vars", "echo $HOME && echo ${PATH}"},
		{"heredoc_style", "cat << 'EOF'\nhello\nEOF"},
		{"carriage_return_lf", "echo hello\r\necho world"},
		{"only_spaces", "   "},
		{"only_tabs", "\t\t\t"},
		{"only_newlines", "\n\n\n"},
		{"leading_newline", "\necho hello"},
		{"trailing_newline", "echo hello\n"},
		{"xml_like_tags", "echo '<string>value</string>'"},
		{"cdata_sequence", "echo '<![CDATA[raw]]>'"},
		{"del_char", "echo\x7fhello"}, // DEL (0x7F) is allowed in XML 1.0
		{"high_unicode", "echo '🎉 done'"},
		{"set_e_script", "set -e\nset -u\nset -o pipefail"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInlineScript(tc.script); err != nil {
				t.Errorf("unexpected error for %q: %v", tc.name, err)
			}
		})
	}
}

func TestRunAdd_RelativePathWithMultipleArgs(t *testing.T) {
	// When the first argument is a relative path AND additional arguments are
	// present, runAdd must return an error with the tip about inline scripts.
	// This check happens before any filesystem or launchd interaction.
	tests := []struct {
		name     string
		wantPart string
		args     []string
	}{
		{
			name:     "relative_plus_one_arg",
			args:     []string{"0 * * * *", "myscript.sh", "arg1"},
			wantPart: "must be an absolute path",
		},
		{
			name:     "relative_plus_multiple_args",
			args:     []string{"0 * * * *", "script", "a", "b", "c"},
			wantPart: "tip:",
		},
		{
			name:     "dotslash_plus_arg",
			args:     []string{"0 * * * *", "./run.sh", "--flag"},
			wantPart: "must be an absolute path",
		},
		{
			name:     "dotdotslash_plus_arg",
			args:     []string{"0 * * * *", "../sibling.sh", "arg"},
			wantPart: "must be an absolute path",
		},
		{
			name:     "tilde_path_plus_arg",
			args:     []string{"0 * * * *", "~/bin/script.sh", "arg"},
			wantPart: "must be an absolute path",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runAdd(addCmd, tc.args)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantPart) {
				t.Errorf("error %q should contain %q", err.Error(), tc.wantPart)
			}
		})
	}
}

func TestRunAdd_RelativePathTipIncludesSchedule(t *testing.T) {
	// The tip in the error message must echo back the schedule so the user
	// can copy-paste it correctly.
	schedule := "*/10 9-17 * * 1-5"
	err := runAdd(addCmd, []string{schedule, "my_script.sh", "arg"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), schedule) {
		t.Errorf("error should echo the schedule %q back to the user; got: %v", schedule, err)
	}
}

func TestRunAdd_InlineScriptWithNullByte(t *testing.T) {
	// When the inline script contains a null byte, runAdd must return the
	// validation error before any filesystem or launchd interaction.
	tests := []struct {
		name   string
		script string
	}{
		{"null_only", "\x00"},
		{"null_in_middle", "echo hello\x00world"},
		{"null_at_start", "\x00start"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runAdd(addCmd, []string{"0 * * * *", tc.script})
			if err == nil {
				t.Fatal("expected error for inline script with null byte, got nil")
			}
			if !strings.Contains(err.Error(), "null byte") {
				t.Errorf("error should mention null byte: %v", err)
			}
		})
	}
}

func TestRunAdd_InlineScriptWithXMLForbiddenControlChar(t *testing.T) {
	// XML 1.0 illegal control chars in an inline script must be caught and
	// reported before any filesystem or launchd interaction.
	tests := []struct {
		name   string
		script string
	}{
		{"bel_char", "echo\x07bell"},
		{"esc_char", "\x1becho escaped"},
		{"form_feed", "clear\x0cecho"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runAdd(addCmd, []string{"0 * * * *", tc.script})
			if err == nil {
				t.Fatal("expected error for script with forbidden control char, got nil")
			}
			if !strings.Contains(err.Error(), "control character") {
				t.Errorf("error should mention 'control character': %v", err)
			}
		})
	}
}
