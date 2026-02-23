package job_test

import (
	"testing"

	"github.com/s4na/ldcron/internal/job"
)

func TestNewJob_IDAndLabel(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/local/bin/myscript.sh"})
	if len(j.ID) != 16 {
		t.Errorf("ID length: got %d, want 16", len(j.ID))
	}
	want := "com.ldcron." + j.ID
	if j.Label != want {
		t.Errorf("Label: got %q, want %q", j.Label, want)
	}
}

func TestNewJob_Deterministic(t *testing.T) {
	schedule := "0 12 * * *"
	args := []string{"/usr/local/bin/myscript.sh"}
	j1 := job.NewJob(schedule, args)
	j2 := job.NewJob(schedule, args)
	if j1.ID != j2.ID {
		t.Errorf("IDs differ: %q vs %q", j1.ID, j2.ID)
	}
}

func TestNewJob_DifferentSchedulesDifferentIDs(t *testing.T) {
	j1 := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	j2 := job.NewJob("0 13 * * *", []string{"/usr/bin/foo"})
	if j1.ID == j2.ID {
		t.Errorf("different schedules should produce different IDs, got %q", j1.ID)
	}
}

func TestNewJob_DifferentArgsDifferentIDs(t *testing.T) {
	j1 := job.NewJob("0 12 * * *", []string{"/usr/bin/foo"})
	j2 := job.NewJob("0 12 * * *", []string{"/usr/bin/bar"})
	if j1.ID == j2.ID {
		t.Errorf("different args should produce different IDs, got %q", j1.ID)
	}
}

func TestNewJob_MultipleArgs(t *testing.T) {
	j := job.NewJob("0 12 * * *", []string{"/usr/bin/ruby", "/path/to/script.rb", "--verbose"})
	if j.Args[0] != "/usr/bin/ruby" {
		t.Errorf("Args[0]: got %q, want /usr/bin/ruby", j.Args[0])
	}
	if len(j.Args) != 3 {
		t.Errorf("len(Args): got %d, want 3", len(j.Args))
	}
}
