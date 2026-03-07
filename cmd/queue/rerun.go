package queuecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/rerun"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var rerunFrom string

var rerunCmd = &cobra.Command{
	Use:   "rerun <id>",
	Short: "Rerun a queue item from a specific step",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		original, err := q.Load(id)
		if err != nil {
			return fmt.Errorf("load queue item: %w", err)
		}

		tk, err := task.Load(p.TaskFile(original.Task))
		if err != nil {
			return fmt.Errorf("load task: %w", err)
		}

		result, err := rerun.Run(q, original, tk, rerunFrom)
		if err != nil {
			return err
		}

		fmt.Printf("Rerun: %s from step %q (pid %d)\n", result.Item.ID, result.Item.RerunFromStep, result.PID)
		return nil
	},
}

func init() {
	rerunCmd.Flags().StringVar(&rerunFrom, "from", "", "step name to rerun from (defaults to paused/failed step)")
	Cmd.AddCommand(rerunCmd)
}
