package taskcmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/detach"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a task now",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p, err := paths.New()
		if err != nil {
			return err
		}

		tk, err := task.Load(p.TaskFile(name))
		if err != nil {
			return fmt.Errorf("load task: %w", err)
		}

		q := queue.New(p.QueueDir())

		// NewItem with StatusPaused gives us a valid item; we override to running
		// since this is a direct run, not a pipeline pause.
		item := queue.NewItem(tk.Name, pipeline.StatusPaused, nil, 0)
		item.Status = queue.StatusRunning
		item.StepsCompleted = make(map[string]pipeline.StepResult)

		pid, err := detach.Run(q, item)
		if err != nil {
			return err
		}

		fmt.Printf("Running: %s (pid %d)\n", item.ID, pid)
		return nil
	},
}

func init() {
	Cmd.AddCommand(runCmd)
}
