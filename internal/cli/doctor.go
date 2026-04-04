package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"luumen/internal/doctor"
	"luumen/internal/workspace"
)

type doctorRunner interface {
	Run(state workspace.Workspace) doctor.Report
}

type doctorCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	runner          doctorRunner
}

func defaultDoctorCommandDeps() doctorCommandDeps {
	return doctorCommandDeps{
		detectWorkspace: workspace.Detect,
		runner:          doctor.NewRunner(nil),
	}
}

func newDoctorCmd(deps doctorCommandDeps) *cobra.Command {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.runner == nil {
		deps.runner = doctor.NewRunner(nil)
	}

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run repository health checks",
		Long:  "Doctor validates tool availability, config health, and common setup issues in the current workspace.",
		Example: "luu doctor\n" +
			"luu doctor --quiet",
		Args: requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Running health checks...")
			state, err := deps.detectWorkspace("")
			if err != nil {
				return fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
			}

			report := deps.runner.Run(state)
			printDoctorReport(cmd.OutOrStdout(), report)
			if report.HasErrors() {
				return fmt.Errorf("doctor found %d error(s)", report.Errors)
			}
			return nil
		},
	}

	return cmd
}

func printDoctorReport(writer io.Writer, report doctor.Report) {
	fmt.Fprintln(writer)

	for _, result := range report.Results {
		styledSeverity := styleDoctorSeverity(writer, result.Severity)
		fmt.Fprintf(writer, "%s %s", styledSeverity, result.Message)
		if strings.TrimSpace(result.ID) != "" {
			fmt.Fprintf(writer, " %s", styleMuted(writer, "("+result.ID+")"))
		}
		fmt.Fprintln(writer)
		if strings.TrimSpace(result.Suggestion) != "" {
			fmt.Fprintf(writer, "%s %s\n", nextPrefix(writer), result.Suggestion)
		}
	}

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, styleMuted(writer, "summary"))
	fmt.Fprintf(writer, "  %s %d\n", styleSuccess(writer, "pass:"), report.Passes)
	fmt.Fprintf(writer, "  %s %d\n", styleWarning(writer, "warning:"), report.Warnings)
	fmt.Fprintf(writer, "  %s %d\n", styleError(writer, "error:"), report.Errors)
}

func styleDoctorSeverity(writer io.Writer, severity doctor.Severity) string {
	label := string(severity) + ":"
	switch severity {
	case doctor.SeverityPass:
		return styleSuccess(writer, label)
	case doctor.SeverityWarning:
		return styleWarning(writer, label)
	case doctor.SeverityError:
		return styleError(writer, label)
	default:
		return label
	}
}
