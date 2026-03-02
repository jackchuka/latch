package taskcmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/scheduler"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a task and its data",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p, err := paths.New()
		if err != nil {
			return err
		}

		taskFile := p.TaskFile(name)
		tk, err := task.Load(taskFile)
		if err != nil {
			return fmt.Errorf("load task: %w", err)
		}

		if tk.Schedule != "" {
			sched, err := scheduler.New()
			if err != nil {
				return fmt.Errorf("init scheduler: %w", err)
			}
			if err := sched.Uninstall(name); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to uninstall schedule: %v\n", err)
			}
		}

		if err := os.Remove(taskFile); err != nil {
			return fmt.Errorf("remove task file: %w", err)
		}

		q := queue.New(p.QueueDir())
		n, err := q.DeleteByTask(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to clean queue: %v\n", err)
		}
		if n > 0 {
			fmt.Printf("Deleted %d queue item(s).\n", n)
		}

		fmt.Printf("Removed task: %s\n", name)
		return nil
	},
}

func init() {
	Cmd.AddCommand(removeCmd)
}
