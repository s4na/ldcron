package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogSetupRotation(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	wantPattern := filepath.Join(home, "Library", "Logs", "ldcron", "*.log")

	var buf bytes.Buffer
	cmd := logSetupRotationCmd
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, wantPattern) {
		t.Errorf("output should contain log pattern %q, got:\n%s", wantPattern, output)
	}
	if !strings.Contains(output, "GNB") {
		t.Errorf("output should contain flags GNB, got:\n%s", output)
	}
	if !strings.Contains(output, "1024") {
		t.Errorf("output should contain size 1024, got:\n%s", output)
	}
}
