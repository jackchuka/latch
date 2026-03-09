package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/jackchuka/latch/internal/detach"
	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		stop, _ := cmd.Flags().GetBool("stop")
		port, _ := cmd.Flags().GetInt("port")
		fg, _ := cmd.Flags().GetBool("foreground")

		p, err := paths.New()
		if err != nil {
			return err
		}

		pidFile := filepath.Join(p.DataDir(), fmt.Sprintf("serve-%d.pid", port))

		if stop {
			if cmd.Flags().Changed("port") {
				return stopOne(pidFile)
			}
			return stopAll(p.DataDir())
		}

		if !fg {
			return startBackground(pidFile, port)
		}

		return runForeground(p, pidFile, port)
	},
}

func startBackground(pidFile string, port int) error {
	if data, err := os.ReadFile(pidFile); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		if pid > 0 {
			if proc, err := os.FindProcess(pid); err == nil {
				if proc.Signal(syscall.Signal(0)) == nil {
					return fmt.Errorf("server already running (pid %d)", pid)
				}
			}
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	bg := exec.Command(exe, "serve", "--foreground", "-p", strconv.Itoa(port))
	bg.SysProcAttr = detach.DetachAttr()
	if err := bg.Start(); err != nil {
		return fmt.Errorf("start background server: %w", err)
	}

	fmt.Printf("Server started on http://localhost:%d (pid %d)\n", port, bg.Process.Pid)
	return nil
}

func runForeground(p *paths.Paths, pidFile string, port int) error {
	if err := os.MkdirAll(p.DataDir(), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	defer func() { _ = os.Remove(pidFile) }()

	q := queue.New(p.QueueDir())
	logger := log.New(os.Stderr, "latch-web: ", log.LstdFlags)
	srv := web.NewServer(q, p.TasksDir(), logger)

	addr := fmt.Sprintf(":%d", port)
	logger.Printf("listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, srv.Handler())
}

func stopAll(dataDir string) error {
	files, _ := filepath.Glob(filepath.Join(dataDir, "serve-*.pid"))
	if len(files) == 0 {
		return fmt.Errorf("no running servers found")
	}
	var errs []string
	for _, f := range files {
		if err := stopOne(f); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func stopOne(pidFile string) error {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("no running server found (missing pid file)")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid pid file: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = os.Remove(pidFile)
		return fmt.Errorf("failed to stop server (pid %d): %w", pid, err)
	}
	_ = os.Remove(pidFile)
	fmt.Printf("Stopped server (pid %d)\n", pid)
	return nil
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serveCmd.Flags().Bool("stop", false, "Stop the running server")
	serveCmd.Flags().Bool("foreground", false, "Run in the foreground")
	rootCmd.AddCommand(serveCmd)
}
