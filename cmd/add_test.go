package cmd

import (
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInlineScript(tc.script); err != nil {
				t.Errorf("unexpected error for %q: %v", tc.name, err)
			}
		})
	}
}
