package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackchuka/latch/internal/paths"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")

		p, err := paths.New()
		if err != nil {
			return err
		}

		q := queue.New(p.QueueDir())
		logger := log.New(os.Stderr, "latch-web: ", log.LstdFlags)
		srv := web.NewServer(q, p.TasksDir(), logger)

		addr := fmt.Sprintf(":%d", port)
		logger.Printf("listening on http://localhost%s", addr)
		return http.ListenAndServe(addr, srv.Handler())
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}
