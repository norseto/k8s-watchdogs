package version

import (
	"fmt"

	watchdogs "github.com/norseto/k8s-watchdogs"
	"github.com/spf13/cobra"
)

// NewCommand returns a command that prints application version information.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "version: %s\nGitVersion: %s\n", watchdogs.RELEASE_VERSION, watchdogs.GitVersion)
		},
	}
}
