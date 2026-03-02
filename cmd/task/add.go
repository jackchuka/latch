package taskcmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/scheduler"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a task",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := paths.New()
		if err != nil {
			return err
		}
		if err := p.EnsureDirs(); err != nil {
			return fmt.Errorf("ensure dirs: %w", err)
		}

		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("please provide a task file with -f")
		}

		tk, err := task.Load(filePath)
		if err != nil {
			return fmt.Errorf("load task file: %w", err)
		}

		src, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read task file: %w", err)
		}

		dest := p.TaskFile(tk.Name)
		if err := os.WriteFile(dest, src, 0o644); err != nil {
			return fmt.Errorf("write task file: %w", err)
		}

		if tk.Schedule != "" {
			sched, err := scheduler.New()
			if err != nil {
				return fmt.Errorf("init scheduler: %w", err)
			}
			if err := sched.Install(tk.Name, tk.Schedule); err != nil {
				return fmt.Errorf("install scheduled job: %w", err)
			}
			fmt.Printf("Added task: %s (schedule: %s)\n", tk.Name, tk.Schedule)
		} else {
			fmt.Printf("Added task: %s\n", tk.Name)
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringP("file", "f", "", "Path to task definition file")
	Cmd.AddCommand(addCmd)
}
