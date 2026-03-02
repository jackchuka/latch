package schedulecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/scheduler"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Register scheduled tasks with the system scheduler",
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
			if err := sched.Install(tk.Name, tk.Schedule); err != nil {
				return fmt.Errorf("install %s: %w", tk.Name, err)
			}
			count++
		}

		fmt.Printf("Registered %d task(s) with scheduler.\n", count)
		return nil
	},
}

func init() {
	Cmd.AddCommand(installCmd)
}
