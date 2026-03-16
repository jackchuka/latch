package web

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
)

//go:embed templates/*.html templates/partials/*.html
var templateFS embed.FS

type Server struct {
	queue    *queue.Queue
	tasksDir string
	logger   *log.Logger
	index    *template.Template
	show     *template.Template
	tasks    *template.Template
	taskEdit *template.Template
	partial  *template.Template
}

func NewServer(q *queue.Queue, tasksDir string, logger *log.Logger) *Server {
	funcMap := template.FuncMap{
		"statusClass":  statusClass,
		"formatTime":   formatTime,
		"relativeTime": relativeTime,
		"hasStep": func(m map[string]pipeline.StepResult, name string) bool {
			_, ok := m[name]
			return ok
		},
		"joinArgs": func(args []string) string {
			return strings.Join(args, ", ")
		},
		"truncateID": func(id string) string {
			if len(id) > 15 {
				return id[:15]
			}
			return id
		},
		"effectiveOutput": func(sr pipeline.StepResult) string {
			return sr.EffectiveOutput()
		},
		"hasOverride": func(sr pipeline.StepResult) bool {
			return sr.OutputOverride != ""
		},
	}

	// Parse shared templates (layout + partials) as a base, then clone per page
	// so each page gets its own "content" definition.
	base := template.Must(
		template.New("").Funcs(funcMap).ParseFS(templateFS,
			"templates/layout.html",
			"templates/partials/queue_table.html",
		),
	)

	index := template.Must(template.Must(base.Clone()).ParseFS(templateFS, "templates/index.html"))
	show := template.Must(template.Must(base.Clone()).ParseFS(templateFS, "templates/show.html"))
	tasks := template.Must(template.Must(base.Clone()).ParseFS(templateFS, "templates/tasks.html"))
	taskEdit := template.Must(template.Must(base.Clone()).ParseFS(templateFS, "templates/task_edit.html"))

	return &Server{
		queue:    q,
		tasksDir: tasksDir,
		logger:   logger,
		index:    index,
		show:     show,
		tasks:    tasks,
		taskEdit: taskEdit,
		partial:  base,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /partials/queue", s.handleQueuePartial)
	mux.HandleFunc("GET /queue/{id}", s.handleShow)
	mux.HandleFunc("POST /queue/{id}/approve", s.handleApprove)
	mux.HandleFunc("POST /queue/{id}/reject", s.handleReject)
	mux.HandleFunc("POST /queue/{id}/rerun", s.handleRerun)
	mux.HandleFunc("POST /queue/{id}/steps/{step}/output", s.handleStepOutputUpdate)
	mux.HandleFunc("POST /queue/clear", s.handleClear)
	mux.HandleFunc("POST /queue/clear-all", s.handleClearAll)
	mux.HandleFunc("GET /tasks", s.handleTaskList)
	mux.HandleFunc("GET /tasks/{name}", s.handleTaskEdit)
	mux.HandleFunc("POST /tasks/{name}", s.handleTaskSave)
	mux.HandleFunc("POST /tasks/{name}/run", s.handleTaskRun)

	return mux
}

func statusClass(status string) string {
	switch status {
	case queue.StatusPending:
		return "pending"
	case queue.StatusRunning:
		return "running"
	case queue.StatusDone:
		return "done"
	case queue.StatusFailed:
		return "failed"
	case queue.StatusRejected:
		return "rejected"
	default:
		return "pending"
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
