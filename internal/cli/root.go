package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var quiet bool
	verbose := true

	root := &cobra.Command{
		Use:           "luu",
		Short:         "Unified Roblox filesystem workflow CLI",
		Long:          "Luumen orchestrates existing Roblox tooling (Rokit, Wally, and Rojo) through a single CLI.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			ctx := withQuietMode(cmd.Context(), quiet)
			cmd.SetContext(withVerboseMode(ctx, verbose))
		},
	}

	root.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress status output")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", true, "show underlying tool output (use --verbose=false to hide)")

	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		writer := cmd.OutOrStdout()
		fmt.Fprintf(writer, "%s %s\n\n", styleAccent(writer, "luu"), styleMuted(writer, "unified Roblox workflow CLI"))
		defaultHelp(cmd, args)
	})

	root.AddCommand(newAddCmd(defaultAddCommandDeps()))
	root.AddCommand(newCreateCmd(defaultCreateCommandDeps()))
	root.AddCommand(newInitCmd(defaultInitCommandDeps()))
	root.AddCommand(newInstallCmd(defaultInstallCommandDeps()))
	root.AddCommand(newRunCmd(defaultRunCommandDeps()))
	workflowDeps := defaultWorkflowCommandDeps()
	root.AddCommand(newServeCmd(workflowDeps))
	root.AddCommand(newSourcemapCmd(workflowDeps))
	root.AddCommand(newBuildCmd(workflowDeps))
	root.AddCommand(newDevCmd(workflowDeps))
	root.AddCommand(newDoctorCmd(defaultDoctorCommandDeps()))
	return root
}
