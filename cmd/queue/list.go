package queuecmd

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/jackchuka/latch/internal/output"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/task"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List queue items (pending approvals by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		statusFlag, _ := cmd.Flags().GetString("status")
		statuses, err := parseStatuses(statusFlag)
		if err != nil {
			return err
		}

		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		items, err := q.List(statuses...)
		if err != nil {
			return fmt.Errorf("list queue: %w", err)
		}

		if output.Format(cmd) == output.FormatJSON {
			if items == nil {
				items = []*queue.Item{}
			}
			return output.JSON(os.Stdout, items)
		}

		if len(items) == 0 {
			if statusFlag == queue.StatusPending {
				fmt.Println("No pending approvals.")
			} else {
				fmt.Println("No matching queue items.")
			}
			return nil
		}

		tasks := make(map[string]*task.Task)
		rows := make([][]string, len(items))
		for i, item := range items {
			tk, ok := tasks[item.Task]
			if !ok {
				tk, _ = task.Load(p.TaskFile(item.Task))
				tasks[item.Task] = tk
			}
			rows[i] = []string{
				item.ID,
				item.Task,
				item.Status,
				item.Created.Format("2006-01-02 15:04"),
				item.StepLabel(tk),
			}
		}
		return output.Table(os.Stdout, []string{"ID", "Task", "Status", "Created", "Step"}, rows)
	},
}

// parseStatuses turns a comma-separated --status value into statuses.
// "all" returns nil, which matches every item.
func parseStatuses(value string) ([]string, error) {
	if value == "all" {
		return nil, nil
	}
	var statuses []string
	for _, s := range strings.Split(value, ",") {
		s = strings.TrimSpace(s)
		if !slices.Contains(queue.Statuses, s) {
			return nil, fmt.Errorf("invalid status %q: use %s, or all", s, strings.Join(queue.Statuses, ", "))
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}

func init() {
	output.AddFlag(listCmd)
	listCmd.Flags().StringP("status", "s", queue.StatusPending,
		"Statuses to list, comma-separated: "+strings.Join(queue.Statuses, ", ")+", or all")
	Cmd.AddCommand(listCmd)
}
