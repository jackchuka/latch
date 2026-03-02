package schedulecmd

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage the system scheduler",
}
