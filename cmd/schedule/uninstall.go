package schedulecmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/scheduler"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Unregister all tasks from the system scheduler",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := paths.New()
		if err != nil {
			return err
		}

		tasks, err := task.LoadAll(p.TasksDir())
		if err != nil {
			return fmt.Errorf("load tasks: %w", err)
		}

		sched, err := scheduler.New()
		if err != nil {
			return fmt.Errorf("init scheduler: %w", err)
		}

		count := 0
		for _, tk := range tasks {
			if tk.Schedule == "" {
				continue
			}
			if err := sched.Uninstall(tk.Name); err != nil {
				fmt.Printf("Warning: failed to uninstall %s: %v\n", tk.Name, err)
				continue
			}
			count++
		}

		fmt.Printf("Unregistered %d task(s) from scheduler.\n", count)

		purge, _ := cmd.Flags().GetBool("purge")
		if purge {
			if err := os.RemoveAll(p.QueueDir()); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to purge queue: %v\n", err)
			} else {
				fmt.Println("Purged queue.")
			}
		}

		return nil
	},
}

func init() {
	uninstallCmd.Flags().Bool("purge", false, "Also delete all queue items")
	Cmd.AddCommand(uninstallCmd)
}
