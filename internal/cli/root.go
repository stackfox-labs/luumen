package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var quiet bool
	verbose := true
	var yes bool
	var noPrompt bool
	var installMissing bool

	root := &cobra.Command{
		Use:           "luu",
		Short:         "Unified Luau workflow CLI",
		Long:          "Luumen orchestrates existing Luau tooling (Rokit, Wally, Rojo, and Lute) through a single CLI.",
		Version:       currentVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			ctx := withQuietMode(cmd.Context(), quiet)
			ctx = withVerboseMode(ctx, verbose)
			ctx = withYesMode(ctx, yes)
			ctx = withNoPromptMode(ctx, noPrompt)
			ctx = withInstallMissingMode(ctx, installMissing)
			cmd.SetContext(ctx)
		},
	}

	root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress status output")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", true, "show underlying tool output (use --verbose=false to hide)")
	root.PersistentFlags().BoolVar(&yes, "yes", false, "auto-accept confirmation prompts")
	root.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable interactive prompts and fail when input is required")
	root.PersistentFlags().BoolVar(&installMissing, "install-missing", false, "install known missing tools automatically without prompting")

	root.AddGroup(
		&cobra.Group{ID: "workflow", Title: "Workflow Commands"},
		&cobra.Group{ID: "project", Title: "Project Commands"},
		&cobra.Group{ID: "deps", Title: "Dependency Commands"},
		&cobra.Group{ID: "health", Title: "Health Commands"},
		&cobra.Group{ID: "other", Title: "Other Commands"},
	)
	root.SetHelpCommandGroupID("other")
	root.SetCompletionCommandGroupID("other")

	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		writer := cmd.OutOrStdout()
		fmt.Fprintf(writer, "%s %s\n", styleAccent(writer, cmd.Root().Use), styleMuted(writer, "unified Luau workflow CLI"))
		if cmd == root {
			fmt.Fprintln(writer)
		}
		defaultHelp(cmd, args)
	})

	addCmd := newAddCmd(defaultAddCommandDeps())
	addCmd.GroupID = "deps"
	createCmd := newCreateCmd(defaultCreateCommandDeps())
	createCmd.GroupID = "project"
	initCmd := newInitCmd(defaultInitCommandDeps())
	initCmd.GroupID = "project"
	installCmd := newInstallCmd(defaultInstallCommandDeps())
	installCmd.GroupID = "deps"
	runCmd := newRunCmd(defaultRunCommandDeps())
	runCmd.GroupID = "workflow"
	doctorCmd := newDoctorCmd(defaultDoctorCommandDeps())
	doctorCmd.GroupID = "health"

	root.AddCommand(addCmd)
	root.AddCommand(createCmd)
	root.AddCommand(initCmd)
	root.AddCommand(installCmd)
	root.AddCommand(runCmd)
	workflowDeps := defaultWorkflowCommandDeps()
	buildCmd := newBuildCmd(workflowDeps)
	buildCmd.GroupID = "workflow"
	devCmd := newDevCmd(workflowDeps)
	devCmd.GroupID = "workflow"
	lintCmd := newLintCmd(workflowDeps)
	lintCmd.GroupID = "workflow"
	formatCmd := newFormatCmd(workflowDeps)
	formatCmd.GroupID = "workflow"
	testCmd := newTestCmd(workflowDeps)
	testCmd.GroupID = "workflow"

	root.AddCommand(buildCmd)
	root.AddCommand(devCmd)
	root.AddCommand(lintCmd)
	root.AddCommand(formatCmd)
	root.AddCommand(testCmd)
	root.AddCommand(doctorCmd)
	return root
}
