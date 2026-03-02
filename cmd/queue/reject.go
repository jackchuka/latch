package queuecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/cobra"
)

var rejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a queued item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		item, err := q.Load(id)
		if err != nil {
			return fmt.Errorf("load queue item: %w", err)
		}
		if item.Status != queue.StatusPending {
			return fmt.Errorf("item %s is not pending (status: %s)", id, item.Status)
		}

		if err := q.Delete(id); err != nil {
			return fmt.Errorf("delete queue item: %w", err)
		}

		fmt.Printf("Rejected: %s\n", id)
		return nil
	},
}

func init() {
	Cmd.AddCommand(rejectCmd)
}
