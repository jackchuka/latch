package web

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackchuka/latch/internal/pipeline"
	"github.com/jackchuka/latch/internal/queue"
)

func newTestServer(t *testing.T) (*Server, *queue.Queue) {
	t.Helper()
	queueDir := t.TempDir()
	tasksDir := t.TempDir()
	q := queue.New(queueDir)
	logger := log.New(os.Stderr, "test: ", 0)
	srv := NewServer(q, tasksDir, logger)
	return srv, q
}

func seedItem(t *testing.T, q *queue.Queue, id, task, status string) {
	t.Helper()
	item := &queue.Item{
		ID:             id,
		Task:           task,
		Created:        time.Now(),
		Status:         status,
		StepsCompleted: map[string]pipeline.StepResult{},
	}
	if err := q.Save(item); err != nil {
		t.Fatalf("seed %s: %v", id, err)
	}
}

func TestHandleIndexEmpty(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !containsString(body, "No items in the queue") {
		t.Error("expected empty queue message")
	}
}

func TestHandleIndexWithItems(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "item-1", "deploy", queue.StatusPending)
	seedItem(t, q, "item-2", "backup", queue.StatusDone)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !containsString(body, "item-1") || !containsString(body, "item-2") {
		t.Error("expected both items in response")
	}
}

func TestHandleQueuePartial(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "item-1", "deploy", queue.StatusPending)

	req := httptest.NewRequest("GET", "/partials/queue", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !containsString(body, "item-1") {
		t.Error("expected item in partial response")
	}
	// Partial should not contain full page layout
	if containsString(body, "<!DOCTYPE html>") {
		t.Error("partial should not contain full HTML document")
	}
}

func TestHandleShow(t *testing.T) {
	srv, q := newTestServer(t)
	item := &queue.Item{
		ID:      "show-1",
		Task:    "deploy",
		Created: time.Now(),
		Status:  queue.StatusPending,
		StepsCompleted: map[string]pipeline.StepResult{
			"build": {Output: "compiled ok", Duration: "1.2s"},
		},
	}
	if err := q.Save(item); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/queue/show-1", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !containsString(body, "show-1") {
		t.Error("expected item ID in response")
	}
	if !containsString(body, "compiled ok") {
		t.Error("expected step output in response")
	}
}

func TestHandleShowNotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/queue/nonexistent", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleReject(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "rej-1", "deploy", queue.StatusPending)

	req := httptest.NewRequest("POST", "/queue/rej-1/reject", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusSeeOther)
	}

	// Item should be deleted
	_, err := q.Load("rej-1")
	if err == nil {
		t.Fatal("expected item to be deleted after reject")
	}
}

func TestHandleRejectNonPending(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "rej-2", "deploy", queue.StatusRunning)

	req := httptest.NewRequest("POST", "/queue/rej-2/reject", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusSeeOther)
	}

	// Item should still exist
	item, err := q.Load("rej-2")
	if err != nil {
		t.Fatal("item should still exist after reject of non-pending")
	}
	if item.Status != queue.StatusRunning {
		t.Errorf("status should still be running, got %s", item.Status)
	}
}

func TestHandleRejectHtmx(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "rej-3", "deploy", queue.StatusPending)

	req := httptest.NewRequest("POST", "/queue/rej-3/reject", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if loc := rec.Header().Get("HX-Redirect"); loc == "" {
		t.Error("expected HX-Redirect header for htmx request")
	}
}

func TestHandleClear(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "done-1", "deploy", queue.StatusDone)
	seedItem(t, q, "done-2", "backup", queue.StatusDone)
	seedItem(t, q, "pend-1", "test", queue.StatusPending)

	req := httptest.NewRequest("POST", "/queue/clear", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusSeeOther)
	}

	items, err := q.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "pend-1" {
		t.Errorf("expected only pending item to remain, got %d items", len(items))
	}
}

func TestHandleClearAll(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "a", "deploy", queue.StatusDone)
	seedItem(t, q, "b", "backup", queue.StatusPending)
	seedItem(t, q, "c", "test", queue.StatusFailed)

	req := httptest.NewRequest("POST", "/queue/clear-all", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusSeeOther)
	}

	items, err := q.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty queue after clear-all, got %d items", len(items))
	}
}

func TestHandleApproveNonPending(t *testing.T) {
	srv, q := newTestServer(t)
	seedItem(t, q, "app-1", "deploy", queue.StatusDone)

	req := httptest.NewRequest("POST", "/queue/app-1/approve", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// Should redirect with error
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status: got %d, want %d", rec.Code, http.StatusSeeOther)
	}
	loc := rec.Header().Get("Location")
	if !containsString(loc, "error") {
		t.Errorf("expected error in redirect location, got %q", loc)
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
