package queuecmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending approvals",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		pending, err := q.ListPending()
		if err != nil {
			return fmt.Errorf("list pending: %w", err)
		}

		if len(pending) == 0 {
			fmt.Println("No pending approvals.")
			return nil
		}

		if output.Format(cmd) == output.FormatJSON {
			return output.JSON(os.Stdout, pending)
		}

		rows := make([][]string, len(pending))
		for i, item := range pending {
			rows[i] = []string{
				item.ID,
				item.Task,
				item.Created.Format("2006-01-02 15:04"),
				strconv.Itoa(item.PausedAtStep),
			}
		}
		return output.Table(os.Stdout, []string{"ID", "Task", "Created", "Paused At"}, rows)
	},
}

func init() {
	output.AddFlag(listCmd)
	Cmd.AddCommand(listCmd)
}
