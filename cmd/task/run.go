package taskcmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/runner"
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

		timeout := runner.ResolveTimeout(tk)
		result, runErr := pipeline.Run(tk, 0, timeout)

		item := runner.SaveResult(q, tk.Name, result, runErr)
		if saveErr := q.Save(item); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save queue item: %v\n", saveErr)
		}

		if runErr != nil {
			return fmt.Errorf("pipeline failed: %w", runErr)
		}

		switch result.Status {
		case pipeline.StatusPaused:
			var stepName string
			if result.PausedAtStep >= 0 && result.PausedAtStep < len(tk.Steps) {
				stepName = tk.Steps[result.PausedAtStep].Name
			}

			fmt.Printf("Pipeline paused at step '%s'. Run 'latch queue list' to review.\n", stepName)

		case pipeline.StatusCompleted:
			fmt.Printf("Pipeline completed: %s\n", tk.Name)
		}

		return nil
	},
}

func init() {
	Cmd.AddCommand(runCmd)
}
