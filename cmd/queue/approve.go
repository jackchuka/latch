package queuecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/detach"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a queued item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		pid, err := detach.Approve(q, id)
		if err != nil {
			return err
		}

		fmt.Printf("Approved: %s (running in background, pid %d)\n", id, pid)
		return nil
	},
}

func init() {
	Cmd.AddCommand(approveCmd)
}
