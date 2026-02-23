package plist_test

import (
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

func TestReadSchedule_RoundTrip(t *testing.T) {
	j := job.NewJob("30 9 * * 1-5", []string{"/usr/bin/ruby", "/path/to/script.rb"})
	data, err := plist.Generate(j.Label, j.Schedule, j.Args, "/tmp/logs")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, j.Label+".plist")
	if err = os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	schedule, args, err := plist.ReadSchedule(path)
	if err != nil {
		t.Fatalf("ReadSchedule: %v", err)
	}
	if schedule != "30 9 * * 1-5" {
		t.Errorf("schedule: got %q, want %q", schedule, "30 9 * * 1-5")
	}
	if len(args) != 2 || args[0] != "/usr/bin/ruby" || args[1] != "/path/to/script.rb" {
		t.Errorf("args: got %v", args)
	}
}
