package queuecmd

import (
	"fmt"
	"os"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a queued item",
	Args:  cobra.ExactArgs(1),
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

		if output.Format(cmd) == output.FormatJSON {
			return output.JSON(os.Stdout, item)
		}

		fmt.Printf("Task:           %s\n", item.Task)
		fmt.Printf("Created:        %s\n", item.Created.Format("2006-01-02 15:04"))
		fmt.Printf("Status:         %s\n", item.Status)

		tk, _ := task.Load(p.TaskFile(item.Task))
		switch item.Status {
		case queue.StatusDone:
		case queue.StatusFailed:
			fmt.Printf("Failed At Step: %s\n", item.StepLabel(tk))
		default:
			fmt.Printf("Paused At Step: %s\n", item.StepLabel(tk))
		}

		for _, name := range item.StepNames(tk) {
			fmt.Printf("\n--- step: %s ---\n", name)
			fmt.Println(item.StepsCompleted[name].Output)
		}

		return nil
	},
}

func init() {
	output.AddFlag(showCmd)
	Cmd.AddCommand(showCmd)
}
