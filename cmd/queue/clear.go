package queuecmd

import (
	"fmt"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear finished queue items",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := paths.New()
		if err != nil {
			return err
		}
		q := queue.New(p.QueueDir())

		all, _ := cmd.Flags().GetBool("all")

		if all {
			items, err := q.ListAll()
			if err != nil {
				return fmt.Errorf("list queue: %w", err)
			}
			for _, item := range items {
				if err := q.Delete(item.ID); err != nil {
					return fmt.Errorf("delete %s: %w", item.ID, err)
				}
			}
			fmt.Printf("Cleared %d item(s).\n", len(items))
			return nil
		}

		n, err := q.DeleteByStatus(queue.StatusDone)
		if err != nil {
			return fmt.Errorf("clear queue: %w", err)
		}
		fmt.Printf("Cleared %d item(s).\n", n)
		return nil
	},
}

func init() {
	clearCmd.Flags().Bool("all", false, "Clear all items including pending")
	Cmd.AddCommand(clearCmd)
}
