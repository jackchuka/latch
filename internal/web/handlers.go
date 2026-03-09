package web

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"

	"github.com/jackchuka/latch/internal/detach"
	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
	"github.com/jackchuka/latch/internal/rerun"
	"github.com/jackchuka/latch/internal/task"
)

type indexData struct {
	Items []*queue.Item
	Flash string
	Error string
	Nav   string
}

type showData struct {
	Item  *queue.Item
	Task  *task.Task
	Flash string
	Error string
	Nav   string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	items, err := s.queue.ListAll()
	if err != nil {
		s.renderError(w, r, "Failed to list queue: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sortItems(items)

	data := indexData{
		Items: items,
		Flash: r.URL.Query().Get("flash"),
		Error: r.URL.Query().Get("error"),
		Nav:   "queue",
	}
	if err := s.index.ExecuteTemplate(w, "index.html", data); err != nil {
		s.logger.Printf("template error: %v", err)
	}
}

func (s *Server) handleQueuePartial(w http.ResponseWriter, r *http.Request) {
	items, err := s.queue.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sortItems(items)

	if err := s.partial.ExecuteTemplate(w, "queue_rows", items); err != nil {
		s.logger.Printf("template error: %v", err)
	}
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := s.queue.Load(id)
	if err != nil {
		s.renderError(w, r, "Item not found: "+id, http.StatusNotFound)
		return
	}

	tk, _ := task.Load(filepath.Join(s.tasksDir, item.Task+".yaml"))

	data := showData{
		Item:  item,
		Task:  tk,
		Flash: r.URL.Query().Get("flash"),
		Error: r.URL.Query().Get("error"),
		Nav:   "queue",
	}
	if err := s.show.ExecuteTemplate(w, "show.html", data); err != nil {
		s.logger.Printf("template error: %v", err)
	}
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pid, err := detach.Approve(s.queue, id)
	if err != nil {
		s.redirect(w, r, fmt.Sprintf("/queue/%s?error=%s", id, url.QueryEscape(err.Error())))
		return
	}
	s.redirect(w, r, fmt.Sprintf("/queue/%s?flash=Approved+%%28pid+%d%%29", id, pid))
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	item, err := s.queue.Load(id)
	if err != nil {
		s.redirect(w, r, "/?error=Item+not+found")
		return
	}
	if item.Status != queue.StatusPending {
		s.redirect(w, r, fmt.Sprintf("/queue/%s?error=Item+is+not+pending", id))
		return
	}

	if err := s.queue.Delete(id); err != nil {
		s.redirect(w, r, fmt.Sprintf("/queue/%s?error=%s", id, url.QueryEscape(err.Error())))
		return
	}
	s.redirect(w, r, "/?flash=Rejected+"+id)
}

func (s *Server) handleRerun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	original, err := s.queue.Load(id)
	if err != nil {
		s.redirect(w, r, "/?error=Item+not+found")
		return
	}

	tk, err := task.Load(filepath.Join(s.tasksDir, original.Task+".yaml"))
	if err != nil {
		s.redirect(w, r, fmt.Sprintf("/queue/%s?error=Task+not+found", id))
		return
	}

	fromStep := r.FormValue("from")
	result, err := rerun.Run(s.queue, original, tk, fromStep)
	if err != nil {
		s.redirect(w, r, fmt.Sprintf("/queue/%s?error=%s", id, url.QueryEscape(err.Error())))
		return
	}

	s.redirect(w, r, fmt.Sprintf("/queue/%s?flash=Rerun+started+%%28pid+%d%%29", result.Item.ID, result.PID))
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	n, err := s.queue.DeleteByStatus(queue.StatusDone)
	if err != nil {
		s.redirect(w, r, "/?error="+err.Error())
		return
	}
	s.redirect(w, r, fmt.Sprintf("/?flash=Cleared+%d+done+items", n))
}

func (s *Server) handleClearAll(w http.ResponseWriter, r *http.Request) {
	n, err := s.queue.DeleteByStatus(queue.StatusPending, queue.StatusRunning, queue.StatusDone, queue.StatusFailed)
	if err != nil {
		s.redirect(w, r, "/?error="+err.Error())
		return
	}
	s.redirect(w, r, fmt.Sprintf("/?flash=Cleared+%d+items", n))
}

// redirect sends a POST-redirect-GET response, or sets HX-Redirect for htmx requests.
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, target string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) renderError(w http.ResponseWriter, _ *http.Request, msg string, code int) {
	w.WriteHeader(code)
	data := indexData{Error: msg, Nav: "queue"}
	if err := s.index.ExecuteTemplate(w, "index.html", data); err != nil {
		s.logger.Printf("template error: %v", err)
		http.Error(w, msg, code)
	}
}

type tasksData struct {
	Tasks []*task.Task
	Flash string
	Error string
	Nav   string
}

func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) {
	tasks, err := task.LoadAll(s.tasksDir)
	if err != nil {
		s.renderError(w, r, "Failed to load tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	data := tasksData{
		Tasks: tasks,
		Flash: r.URL.Query().Get("flash"),
		Error: r.URL.Query().Get("error"),
		Nav:   "tasks",
	}
	if err := s.tasks.ExecuteTemplate(w, "tasks.html", data); err != nil {
		s.logger.Printf("template error: %v", err)
	}
}

func (s *Server) handleTaskRun(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	tk, err := task.Load(filepath.Join(s.tasksDir, name+".yaml"))
	if err != nil {
		s.redirect(w, r, "/tasks?error="+url.QueryEscape("Task not found: "+name))
		return
	}

	item := queue.NewItem(tk.Name, pipeline.StatusPaused, nil, 0)
	item.Status = queue.StatusRunning
	item.StepsCompleted = make(map[string]pipeline.StepResult)

	pid, err := detach.Run(s.queue, item)
	if err != nil {
		s.redirect(w, r, "/tasks?error="+url.QueryEscape(err.Error()))
		return
	}

	s.redirect(w, r, fmt.Sprintf("/?flash=Running+%s+%%28pid+%d%%29", tk.Name, pid))
}

// sortItems orders items: pending first, then running, then by created descending.
func sortItems(items []*queue.Item) {
	statusOrder := map[string]int{
		queue.StatusPending: 0,
		queue.StatusRunning: 1,
		queue.StatusFailed:  2,
		queue.StatusDone:    3,
	}
	sort.Slice(items, func(i, j int) bool {
		oi, oj := statusOrder[items[i].Status], statusOrder[items[j].Status]
		if oi != oj {
			return oi < oj
		}
		return items[i].Created.After(items[j].Created)
	})
}
