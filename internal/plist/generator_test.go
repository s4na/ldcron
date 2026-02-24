package plist_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s4na/ldcron/internal/job"
	"github.com/s4na/ldcron/internal/plist"
)

func TestGenerate_ContainsRequiredKeys(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/local/bin/myscript.sh"})
	data, err := plist.Generate(j.Label, j.Schedule, j.Args, "/Users/test/Library/Logs/ldcron")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	out := string(data)

	checks := []string{
		"<?xml version",
		"<!DOCTYPE plist",
		`<plist version="1.0">`,
		j.Label,
		"/usr/local/bin/myscript.sh",
		"StartCalendarInterval",
		"StandardOutPath",
		"StandardErrorPath",
		"X-Ldcron-Schedule",
		"0 12 * * *",
		j.ID + ".log",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("plist missing %q\nFull output:\n%s", want, out)
		}
	}
}

func TestGenerate_StepScheduleMultipleEntries(t *testing.T) {
	j := job.NewJob("*/15 * * * *", []string{"/usr/bin/true"})
	data, err := plist.Generate(j.Label, j.Schedule, j.Args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	out := string(data)

	// */15 should produce 4 Minute entries: 0, 15, 30, 45
	for _, want := range []string{
		"<integer>0</integer>",
		"<integer>15</integer>",
		"<integer>30</integer>",
		"<integer>45</integer>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing minute entry %q in plist", want)
		}
	}
}

func TestGenerate_MultipleArgs(t *testing.T) {
	j := job.NewJob("0 0 * * *", []string{"/usr/bin/ruby", "/path/to/script.rb", "--verbose"})
	data, err := plist.Generate(j.Label, j.Schedule, j.Args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	out := string(data)
	for _, want := range []string{"/usr/bin/ruby", "/path/to/script.rb", "--verbose"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in ProgramArguments", want)
		}
	}
}

func TestGenerate_InvalidScheduleError(t *testing.T) {
	j := job.NewJob("99 25 * * *", []string{"/usr/bin/true"})
	_, err := plist.Generate(j.Label, j.Schedule, j.Args, "/tmp/logs")
	if err == nil {
		t.Error("expected error for invalid schedule, got nil")
	}
}

func TestReadPlistInfo_LdcronManagedPlist(t *testing.T) {
	j := job.NewJob("0 9 * * 1-5", []string{"/usr/bin/ruby", "/path/to/script.rb"})
	data, err := plist.Generate(j.Label, j.Schedule, j.Args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if err = os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	label, schedule, args, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if label != j.Label {
		t.Errorf("label: got %q, want %q", label, j.Label)
	}
	if schedule != "0 9 * * 1-5" {
		t.Errorf("schedule: got %q, want %q", schedule, "0 9 * * 1-5")
	}
	if len(args) != 2 {
		t.Errorf("args: got %v", args)
	}
}

func TestReadPlistInfo_ExternalPlistNoSchedule(t *testing.T) {
	externalPlist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>Label</key><string>com.apple.example</string>
	<key>ProgramArguments</key><array><string>/usr/bin/example</string><string>--flag</string></array>
</dict></plist>`

	tmp := t.TempDir()
	path := filepath.Join(tmp, "com.apple.example.plist")
	if err := os.WriteFile(path, []byte(externalPlist), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	label, schedule, args, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if label != "com.apple.example" {
		t.Errorf("label: got %q, want %q", label, "com.apple.example")
	}
	if schedule != "" {
		t.Errorf("schedule: got %q, want empty", schedule)
	}
	if len(args) != 2 || args[0] != "/usr/bin/example" || args[1] != "--flag" {
		t.Errorf("args: got %v", args)
	}
}

func TestReadPlistInfo_FallsBackToFilename(t *testing.T) {
	// Plist without Label key.
	noLabelPlist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>ProgramArguments</key><array><string>/usr/bin/foo</string></array>
</dict></plist>`

	tmp := t.TempDir()
	path := filepath.Join(tmp, "com.example.nolabel.plist")
	if err := os.WriteFile(path, []byte(noLabelPlist), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	label, _, _, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if label != "com.example.nolabel" {
		t.Errorf("label: got %q, want %q", label, "com.example.nolabel")
	}
}

func TestReadPlistInfo_ProgramKeyFallback(t *testing.T) {
	// Plist using Program key instead of ProgramArguments.
	programPlist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>Label</key><string>com.example.daemon</string>
	<key>Program</key><string>/usr/sbin/daemon</string>
</dict></plist>`

	tmp := t.TempDir()
	path := filepath.Join(tmp, "com.example.daemon.plist")
	if err := os.WriteFile(path, []byte(programPlist), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	label, _, args, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if label != "com.example.daemon" {
		t.Errorf("label: got %q, want %q", label, "com.example.daemon")
	}
	if len(args) != 1 || args[0] != "/usr/sbin/daemon" {
		t.Errorf("args: got %v, want [/usr/sbin/daemon]", args)
	}
}

// --- Multi-line command tests ---

// writeAndReadArgs is a helper that generates a plist with the given args,
// writes it to a temp file, reads it back, and returns the recovered args.
func writeAndReadArgs(t *testing.T, args []string) []string {
	t.Helper()
	j := job.NewJob("0 * * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}
	_, _, gotArgs, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	return gotArgs
}

func TestGenerate_MultilineArgRoundTrip(t *testing.T) {
	// The primary use case: /bin/sh -c with a multi-line script.
	script := "cd /tmp\necho hello\ndate >> log.txt\necho world"
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if len(gotArgs) != 3 {
		t.Fatalf("args length: got %d, want 3; args=%q", len(gotArgs), gotArgs)
	}
	if gotArgs[2] != script {
		t.Errorf("script arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestGenerate_MultilineArgWithBlankLines(t *testing.T) {
	// Blank lines within the script must also survive the round-trip.
	script := "echo start\n\necho middle\n\necho end"
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("script with blank lines mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestGenerate_XMLSpecialCharsInArgs(t *testing.T) {
	// Args containing XML special characters must be escaped and restored
	// correctly.
	tests := []struct {
		name string
		arg  string
	}{
		{"ampersand", "echo a & b"},
		{"less_than", "if [ $x < 10 ]; then echo ok; fi"},
		{"greater_than", "echo result > /tmp/out.txt"},
		{"double_quote", `echo "hello world"`},
		{"combined", `echo "a < b" && echo "c > d" & echo "e & f"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"/bin/sh", "-c", tc.arg}
			gotArgs := writeAndReadArgs(t, args)
			if len(gotArgs) != 3 || gotArgs[2] != tc.arg {
				t.Errorf("arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], tc.arg)
			}
		})
	}
}

func TestGenerate_TabInArgs(t *testing.T) {
	// Literal tab characters in args should survive the round-trip.
	script := "echo\thello\tworld"
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("tab arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestGenerate_UnicodeInArgs(t *testing.T) {
	// Unicode characters (including multi-byte sequences) must be preserved.
	script := "echo '日本語テスト' && echo '🎉'"
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("unicode arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestGenerate_BackslashInArgs(t *testing.T) {
	// Backslashes in shell scripts (escape sequences, paths) must be preserved.
	script := `echo "line1\nline2" && ls /usr/local/bin`
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("backslash arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestReadPlistInfo_CRLFPreserved(t *testing.T) {
	// Although the XML 1.0 specification requires parsers to normalize \r\n
	// to \n, Go's encoding/xml decoder does NOT perform this normalization.
	// As a result, \r\n inside a <string> element is round-tripped unchanged.
	// This test documents the actual behaviour so callers are not surprised.
	scriptCRLF := "echo hello\r\ndate"

	args := []string{"/bin/sh", "-c", scriptCRLF}
	j := job.NewJob("0 * * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}

	_, _, gotArgs, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}

	// Go's xml decoder preserves \r\n as-is (no normalization).
	if len(gotArgs) != 3 || gotArgs[2] != scriptCRLF {
		t.Errorf("CRLF round-trip: got %q, want %q", gotArgs[2], scriptCRLF)
	}
}

func TestGenerate_SingleArgScript(t *testing.T) {
	// After inline-script wrapping, ProgramArguments is [/bin/sh, -c, script].
	// Verify a realistic wrapped multi-line script survives the round-trip.
	script := "set -e\ncd /var/log\nfind . -name '*.log' -mtime +30 -delete\necho cleaned"
	args := []string{"/bin/sh", "-c", script}

	gotArgs := writeAndReadArgs(t, args)

	if gotArgs[0] != "/bin/sh" {
		t.Errorf("args[0]: got %q, want /bin/sh", gotArgs[0])
	}
	if gotArgs[1] != "-c" {
		t.Errorf("args[1]: got %q, want -c", gotArgs[1])
	}
	if gotArgs[2] != script {
		t.Errorf("script round-trip mismatch:\ngot:  %q\nwant: %q", gotArgs[2], script)
	}
}

func TestReadPlistInfo_CROnlyPreserved(t *testing.T) {
	// \r (CR only, without LF) also survives the round-trip unchanged in
	// Go's encoding/xml decoder. This test documents that behaviour.
	scriptCR := "echo hello\rdate"

	args := []string{"/bin/sh", "-c", scriptCR}
	j := job.NewJob("0 * * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}

	_, _, gotArgs, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}

	if len(gotArgs) != 3 || gotArgs[2] != scriptCR {
		t.Errorf("CR-only round-trip: got %q, want %q", gotArgs[2], scriptCR)
	}
}

// --- Additional edge case tests ---

func TestGenerate_ScriptWithLeadingNewline(t *testing.T) {
	// A script starting with \n must not be trimmed during the plist round-trip.
	script := "\necho hello"
	args := []string{"/bin/sh", "-c", script}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("leading newline trimmed: got %q, want %q", gotArgs[2], script)
	}
}

func TestGenerate_ScriptWithTrailingNewline(t *testing.T) {
	// A script ending with \n must not be trimmed during the plist round-trip.
	script := "echo hello\n"
	args := []string{"/bin/sh", "-c", script}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("trailing newline trimmed: got %q, want %q", gotArgs[2], script)
	}
}

func TestGenerate_ScriptWithOnlyNewlines(t *testing.T) {
	// A script consisting only of newlines should survive unchanged.
	script := "\n\n\n"
	args := []string{"/bin/sh", "-c", script}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("newline-only script trimmed: got %q, want %q", gotArgs[2], script)
	}
}

func TestGenerate_ScriptWithLeadingTrailingSpaces(t *testing.T) {
	// Spaces at the boundaries of a script must not be trimmed by the XML layer.
	script := "   echo hello   "
	args := []string{"/bin/sh", "-c", script}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("spaces trimmed: got %q, want %q", gotArgs[2], script)
	}
}

func TestGenerate_VeryLongScript(t *testing.T) {
	// A large script (many lines, 5000+ bytes) must survive the round-trip.
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("echo 'step %04d: processing item %04d && cleanup > /tmp/log.txt'\n", i, i*100))
	}
	script := sb.String()
	args := []string{"/bin/sh", "-c", script}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != script {
		t.Errorf("very long script round-trip failed; want len=%d, got len=%d",
			len(script), len(gotArgs[2]))
	}
}

func TestGenerate_XMLLikeContentInScript(t *testing.T) {
	// Scripts that contain XML-like syntax must be properly escaped so they
	// cannot be misinterpreted as XML markup on the way in or out.
	tests := []struct {
		name string
		arg  string
	}{
		{"xml_tags", "echo '<string>hello</string>'"},
		{"xml_declaration", `echo '<?xml version="1.0"?>'`},
		{"xml_comment", "echo '<!-- this is a comment -->'"},
		{"plist_structure", "<key>Label</key><string>com.example</string>"},
		{"cdata_end_marker", "echo '<![CDATA[raw]]>'"},
		{"nested_angle_brackets", "if [ $x << 10 ]; then echo '<ok>'; fi"},
		{"double_ampersand", "cmd1 && cmd2 && cmd3"},
		{"or_chain", "cmd1 || cmd2 || echo 'fallback'"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"/bin/sh", "-c", tc.arg}
			gotArgs := writeAndReadArgs(t, args)
			if len(gotArgs) != 3 || gotArgs[2] != tc.arg {
				t.Errorf("arg mismatch:\ngot:  %q\nwant: %q", gotArgs[2], tc.arg)
			}
		})
	}
}

func TestGenerate_ManyProgramArguments(t *testing.T) {
	// ProgramArguments with many separate args must all survive the round-trip
	// in the correct order.
	args := []string{
		"/usr/bin/python3",
		"/path/to/script.py",
		"--input", "/tmp/data.csv",
		"--output", "/tmp/result.json",
		"--verbose",
		"--log-level", "debug",
		"--workers", "4",
	}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != len(args) {
		t.Fatalf("arg count mismatch: got %d, want %d; args=%q", len(gotArgs), len(args), gotArgs)
	}
	for i := range args {
		if gotArgs[i] != args[i] {
			t.Errorf("args[%d]: got %q, want %q", i, gotArgs[i], args[i])
		}
	}
}

func TestGenerate_EmptyStringArg(t *testing.T) {
	// An empty string as an explicit ProgramArguments element (e.g. a
	// placeholder) must survive the round-trip as an empty string, not be
	// dropped entirely.
	args := []string{"/bin/sh", "-c", ""}
	j := job.NewJob("0 * * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}
	_, _, gotArgs, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if len(gotArgs) != 3 {
		t.Fatalf("arg count: got %d, want 3; args=%q", len(gotArgs), gotArgs)
	}
	if gotArgs[2] != "" {
		t.Errorf("empty arg: got %q, want \"\"", gotArgs[2])
	}
}

func TestGenerate_ArgWithOnlySpaces(t *testing.T) {
	// An argument consisting of only spaces must not be trimmed.
	spaces := "   "
	args := []string{"/bin/sh", "-c", spaces}
	gotArgs := writeAndReadArgs(t, args)
	if len(gotArgs) != 3 || gotArgs[2] != spaces {
		t.Errorf("space-only arg: got %q, want %q", gotArgs[2], spaces)
	}
}

func TestGenerate_SchedulePreservedInRoundTrip(t *testing.T) {
	// The original cron expression stored in X-Ldcron-Schedule must be read
	// back byte-for-byte, including step expressions and ranges.
	schedules := []string{
		"0 12 * * *",
		"*/5 * * * *",
		"0 9-17 * * 1-5",
		"30 8 1,15 * *",
		"0 0 * * 0",
	}
	for _, schedule := range schedules {
		t.Run(schedule, func(t *testing.T) {
			args := []string{"/bin/sh", "-c", "echo ok"}
			j := job.NewJob(schedule, args)
			data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			tmp := t.TempDir()
			path := filepath.Join(tmp, j.Label+".plist")
			if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
				t.Fatalf("WriteFile: %v", writeErr)
			}
			_, gotSchedule, _, err := plist.ReadPlistInfo(path)
			if err != nil {
				t.Fatalf("ReadPlistInfo: %v", err)
			}
			if gotSchedule != schedule {
				t.Errorf("schedule: got %q, want %q", gotSchedule, schedule)
			}
		})
	}
}

func TestGenerate_LabelRoundTrip(t *testing.T) {
	// The Label stored in the plist must be read back unchanged.
	args := []string{"/bin/sh", "-c", "echo ok"}
	j := job.NewJob("0 * * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
		t.Fatalf("WriteFile: %v", writeErr)
	}
	gotLabel, _, _, err := plist.ReadPlistInfo(path)
	if err != nil {
		t.Fatalf("ReadPlistInfo: %v", err)
	}
	if gotLabel != j.Label {
		t.Errorf("label: got %q, want %q", gotLabel, j.Label)
	}
}

func TestGenerate_LogPathContainsJobID(t *testing.T) {
	// The StandardOutPath and StandardErrorPath must include the job ID so
	// logs are written to the correct file.
	args := []string{"/usr/bin/true"}
	j := job.NewJob("0 12 * * *", args)
	data, err := plist.Generate(j.Label, j.Schedule, args, "/tmp/ldcron-logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	out := string(data)
	wantPath := "/tmp/ldcron-logs/" + j.ID + ".log"
	if !strings.Contains(out, wantPath) {
		t.Errorf("plist should contain log path %q\nFull output:\n%s", wantPath, out)
	}
	// Should appear twice: once for stdout, once for stderr.
	if strings.Count(out, wantPath) < 2 {
		t.Errorf("log path %q should appear twice (stdout+stderr), got %d occurrences",
			wantPath, strings.Count(out, wantPath))
	}
}

func TestGenerate_SingleLineScript(t *testing.T) {
	// A simple one-liner shell script (the most common case) must round-trip.
	scripts := []string{
		"echo hello",
		"/usr/bin/true",
		"date >> /tmp/log.txt",
		"curl -s https://example.com > /dev/null",
		`find /var/log -name "*.log" -mtime +7 -delete`,
	}
	for _, script := range scripts {
		t.Run(script, func(t *testing.T) {
			args := []string{"/bin/sh", "-c", script}
			gotArgs := writeAndReadArgs(t, args)
			if len(gotArgs) != 3 || gotArgs[2] != script {
				t.Errorf("script mismatch: got %q, want %q", gotArgs[2], script)
			}
		})
	}
}

