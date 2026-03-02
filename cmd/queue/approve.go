package queuecmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/spf13/cobra"
)

var approveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a queued item",
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
		if item.Status != queue.StatusPending {
			return fmt.Errorf("item %s is not pending (status: %s)", id, item.Status)
		}

		item.Status = queue.StatusRunning
		if err := q.Save(item); err != nil {
			return fmt.Errorf("update queue item: %w", err)
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}

		bg := exec.Command(exe, "queue", "exec", id)
		bg.SysProcAttr = detachAttr()
		if err := bg.Start(); err != nil {
			// Roll back status so the item can be retried.
			item.Status = queue.StatusPending
			item.PID = 0
			_ = q.Save(item)
			return fmt.Errorf("start background execution: %w", err)
		}

		item.PID = bg.Process.Pid
		if err := q.Save(item); err != nil {
			return fmt.Errorf("save pid: %w", err)
		}

		fmt.Printf("Approved: %s (running in background, pid %d)\n", id, bg.Process.Pid)
		return nil
	},
}

func init() {
	Cmd.AddCommand(approveCmd)
}
