package taskcmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := paths.New()
		if err != nil {
			return err
		}

		tasks, err := task.LoadAll(p.TasksDir())
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Println("No tasks configured.")
				return nil
			}
			return fmt.Errorf("load tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks configured.")
			return nil
		}

		if output.Format(cmd) == output.FormatJSON {
			return output.JSON(os.Stdout, tasks)
		}

		rows := make([][]string, len(tasks))
		for i, tk := range tasks {
			stepNames := ""
			for j, s := range tk.Steps {
				if j > 0 {
					stepNames += ", "
				}
				stepNames += s.Name
			}
			schedule := tk.Schedule
			if schedule == "" {
				schedule = "-"
			}
			rows[i] = []string{tk.Name, schedule, stepNames}
		}
		return output.Table(os.Stdout, []string{"Name", "Schedule", "Steps"}, rows)
	},
}

func init() {
	output.AddFlag(listCmd)
	Cmd.AddCommand(listCmd)
}
