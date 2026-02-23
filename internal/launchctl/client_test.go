package launchctl

import (
	"errors"
	"fmt"
	"testing"
)

// newTestClient creates a Client with a controllable runner (no real launchctl calls).
func newTestClient(domain string, runFn func(name string, args ...string) error) *Client {
	return &Client{Domain: domain, run: runFn}
}

func TestBootstrap_CommandArgs(t *testing.T) {
	var gotName string
	var gotArgs []string
	c := newTestClient("gui/501", func(name string, args ...string) error {
		gotName = name
		gotArgs = args
		return nil
	})

	if err := c.Bootstrap("/path/to/com.ldcron.abc12345.plist"); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if gotName != "launchctl" {
		t.Errorf("command: got %q, want launchctl", gotName)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "bootstrap" || gotArgs[1] != "gui/501" || gotArgs[2] != "/path/to/com.ldcron.abc12345.plist" {
		t.Errorf("args: got %v", gotArgs)
	}
}

func TestBootout_CommandArgs(t *testing.T) {
	var gotArgs []string
	c := newTestClient("gui/501", func(_ string, args ...string) error {
		gotArgs = args
		return nil
	})

	if err := c.Bootout("com.ldcron.abc12345"); err != nil {
		t.Fatalf("Bootout: %v", err)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "bootout" || gotArgs[1] != "gui/501/com.ldcron.abc12345" {
		t.Errorf("args: got %v", gotArgs)
	}
}

func TestKickstart_CommandArgs(t *testing.T) {
	var gotArgs []string
	c := newTestClient("gui/501", func(_ string, args ...string) error {
		gotArgs = args
		return nil
	})

	if err := c.Kickstart("com.ldcron.abc12345", false); err != nil {
		t.Fatalf("Kickstart: %v", err)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "kickstart" || gotArgs[1] != "gui/501/com.ldcron.abc12345" {
		t.Errorf("args (no force): got %v", gotArgs)
	}
}

func TestKickstart_ForceFlag(t *testing.T) {
	var gotArgs []string
	c := newTestClient("gui/501", func(_ string, args ...string) error {
		gotArgs = args
		return nil
	})

	if err := c.Kickstart("com.ldcron.abc12345", true); err != nil {
		t.Fatalf("Kickstart: %v", err)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "kickstart" || gotArgs[1] != "-k" || gotArgs[2] != "gui/501/com.ldcron.abc12345" {
		t.Errorf("args (force): got %v", gotArgs)
	}
}

func TestBootstrap_PropagatesError(t *testing.T) {
	want := errors.New("launchctl failed")
	c := newTestClient("gui/501", func(_ string, _ ...string) error { return want })
	err := c.Bootstrap("/path/to/plist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("error: got %v, want wrapping %v", err, want)
	}
}

func TestBootout_PropagatesError(t *testing.T) {
	want := fmt.Errorf("service not found")
	c := newTestClient("gui/501", func(_ string, _ ...string) error { return want })
	err := c.Bootout("com.ldcron.missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKickstart_PropagatesError(t *testing.T) {
	want := errors.New("kickstart failed")
	c := newTestClient("gui/501", func(_ string, _ ...string) error { return want })
	err := c.Kickstart("com.ldcron.abc12345", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("error: got %v, want wrapping %v", err, want)
	}
}
