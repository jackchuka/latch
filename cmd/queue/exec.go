package queuecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/runner"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:    "exec <id>",
	Short:  "Execute an approved queue item",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
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
		if item.Status != queue.StatusRunning {
			return fmt.Errorf("item %s is not running (status: %s)", id, item.Status)
		}

		tk, err := task.Load(p.TaskFile(item.Task))
		if err != nil {
			return fmt.Errorf("load task: %w", err)
		}

		timeout := runner.ResolveTimeout(tk)
		result, runErr := pipeline.RunWithContext(tk, item.PausedAtStep, timeout, item.StepsCompleted)

		runner.ApplyResult(item, result, runErr)
		if err := q.Save(item); err != nil {
			return fmt.Errorf("update queue item: %w", err)
		}

		if runErr != nil {
			return fmt.Errorf("pipeline failed: %w", runErr)
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(execCmd)
}
