// Package launchctl wraps launchctl commands for managing launchd agents.
package launchctl

import (
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
)

// Client executes launchctl commands in the gui/<uid> domain.
type Client struct {
	// run is the underlying command executor (swappable in tests).
	run func(name string, args ...string) error
	// Domain is e.g. "gui/501". Set by New().
	Domain string
}

// New returns a Client configured for the current user's gui domain.
func New() (*Client, error) {
	uid, err := currentUID()
	if err != nil {
		return nil, err
	}
	return &Client{
		Domain: "gui/" + strconv.Itoa(uid),
		run:    execRun,
	}, nil
}

// Bootstrap loads the plist at plistPath into the domain.
func (c *Client) Bootstrap(plistPath string) error {
	if err := c.run("launchctl", "bootstrap", c.Domain, plistPath); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}
	return nil
}

// Bootout unloads the service identified by label from the domain.
func (c *Client) Bootout(label string) error {
	if err := c.run("launchctl", "bootout", c.Domain+"/"+label); err != nil {
		return fmt.Errorf("launchctl bootout: %w", err)
	}
	return nil
}

// Kickstart triggers an immediate run of the service (non-blocking).
// If force is true, any currently running instance is killed first (-k flag).
func (c *Client) Kickstart(label string, force bool) error {
	args := []string{"kickstart"}
	if force {
		args = append(args, "-k")
	}
	args = append(args, c.Domain+"/"+label)
	if err := c.run("launchctl", args...); err != nil {
		return fmt.Errorf("launchctl kickstart: %w", err)
	}
	return nil
}

func execRun(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, string(out))
	}
	return nil
}

func currentUID() (int, error) {
	u, err := user.Current()
	if err != nil {
		return 0, fmt.Errorf("failed to get current user: %w", err)
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, fmt.Errorf("failed to parse UID: %w", err)
	}
	return uid, nil
}
