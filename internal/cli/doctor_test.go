package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"luumen/internal/doctor"
	"luumen/internal/workspace"
)

type fakeDoctorRunner struct {
	report doctor.Report
}

func (f *fakeDoctorRunner) Run(_ workspace.Workspace) doctor.Report {
	return f.report
}

func TestDoctorHelpFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"doctor", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected doctor help success, got: %v", err)
	}
	if !strings.Contains(output.String(), "Doctor validates") {
		t.Fatalf("expected doctor help output, got: %q", output.String())
	}
}

func TestDoctorReturnsErrorWhenReportHasErrors(t *testing.T) {
	t.Parallel()

	runner := &fakeDoctorRunner{report: doctor.Report{
		Results: []doctor.CheckResult{{ID: "rojo-binary", Severity: doctor.SeverityError, Message: "missing"}},
		Errors:  1,
	}}

	cmd := newDoctorCmd(doctorCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo"}, nil
		},
		runner: runner,
	})
	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected doctor command error")
	}
	if !strings.Contains(output.String(), "error:") || !strings.Contains(output.String(), "rojo-binary") {
		t.Fatalf("expected readable error output, got: %q", output.String())
	}
	if !strings.Contains(output.String(), "summary") || !strings.Contains(output.String(), "error: 1") {
		t.Fatalf("expected summary output, got: %q", output.String())
	}
}

func TestDoctorDetectFailure(t *testing.T) {
	t.Parallel()

	detectErr := errors.New("detect failed")
	cmd := newDoctorCmd(doctorCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{}, detectErr
		},
		runner: &fakeDoctorRunner{},
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected detect error")
	}
	if !errors.Is(err, detectErr) {
		t.Fatalf("expected wrapped detect error, got: %v", err)
	}
}
