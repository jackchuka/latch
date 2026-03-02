package detach

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jackchuka/latch/internal/queue"
)

// Approve transitions a pending queue item to running and spawns a background
// process to execute the remaining pipeline steps. Returns the child PID.
func Approve(q *queue.Queue, id string) (int, error) {
	item, err := q.Load(id)
	if err != nil {
		return 0, fmt.Errorf("load queue item: %w", err)
	}
	if item.Status != queue.StatusPending {
		return 0, fmt.Errorf("item %s is not pending (status: %s)", id, item.Status)
	}

	item.Status = queue.StatusRunning
	if err := q.Save(item); err != nil {
		return 0, fmt.Errorf("update queue item: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("resolve executable: %w", err)
	}

	bg := exec.Command(exe, "queue", "exec", id)
	bg.SysProcAttr = detachAttr()
	if err := bg.Start(); err != nil {
		item.Status = queue.StatusPending
		item.PID = 0
		_ = q.Save(item)
		return 0, fmt.Errorf("start background execution: %w", err)
	}

	item.PID = bg.Process.Pid
	if err := q.Save(item); err != nil {
		return 0, fmt.Errorf("save pid: %w", err)
	}

	return bg.Process.Pid, nil
}
