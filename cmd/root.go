package cmd

import (
	"os"

	queuecmd "github.com/jackchuka/latch/cmd/queue"
	schedulecmd "github.com/jackchuka/latch/cmd/schedule"
	taskcmd "github.com/jackchuka/latch/cmd/task"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "latch",
	Short: "Task runner with approval gates",
}

func init() {
	rootCmd.AddCommand(taskcmd.Cmd)
	rootCmd.AddCommand(queuecmd.Cmd)
	rootCmd.AddCommand(schedulecmd.Cmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
