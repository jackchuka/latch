package queuecmd

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage the approval queue",
}
