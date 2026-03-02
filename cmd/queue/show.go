package queuecmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
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
		fmt.Printf("Paused At Step: %d\n", item.PausedAtStep)

		names := make([]string, 0, len(item.StepsCompleted))
		for name := range item.StepsCompleted {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
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
