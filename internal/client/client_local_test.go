package client

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simonski/ticket/internal/config"
	"github.com/simonski/ticket/internal/store"
)

func TestLocalModeClientUsesSQLiteDirectly(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TICKET_MODE", "local")
	t.Setenv("TICKET_HOME", tempDir)

	dbPath := filepath.Join(tempDir, "ticket.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	api := New(config.Config{})
	projects, err := api.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].ID != 1 {
		t.Fatalf("ListProjects() = %#v", projects)
	}

	task, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "task",
		Title:     "Local task",
		Stage:     "develop",
		State:     "idle",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if strings.TrimSpace(task.Assignee) != "" || task.Status != "develop/idle" {
		t.Fatalf("CreateTask() = %#v", task)
	}

	requested, err := api.RequestTask(TaskRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("RequestTask() error = %v", err)
	}
	if requested.Status != "ASSIGNED" || requested.Task == nil {
		t.Fatalf("RequestTask() = %#v", requested)
	}

	updated, err := api.UpdateTask(task.ID, TaskUpdateRequest{
		Title:       task.Title,
		Description: task.Description,
		ParentID:    task.ParentID,
		Assignee:    requested.Task.Assignee,
		Stage:       "develop",
		State:       "active",
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Status != "develop/active" {
		t.Fatalf("UpdateTask().Status = %q, want develop/active", updated.Status)
	}

	parent, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "epic",
		Title:     "Parent epic",
	})
	if err != nil {
		t.Fatalf("CreateTask(parent) error = %v", err)
	}
	reparented, err := api.SetTaskParent(task.ID, parent.ID)
	if err != nil {
		t.Fatalf("SetTaskParent() error = %v", err)
	}
	if reparented.ParentID == nil || *reparented.ParentID != parent.ID {
		t.Fatalf("SetTaskParent() = %#v", reparented)
	}

	detached, err := api.UnsetTaskParent(task.ID)
	if err != nil {
		t.Fatalf("UnsetTaskParent() error = %v", err)
	}
	if detached.ParentID != nil {
		t.Fatalf("UnsetTaskParent() = %#v", detached)
	}

	comment, err := api.AddComment(task.ID, "hello")
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if comment.Text != "hello" || comment.Author == "" {
		t.Fatalf("AddComment() = %#v", comment)
	}
}

func TestLocalModeClientIgnoresOwnershipForStatusChanges(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TICKET_MODE", "local")
	t.Setenv("TICKET_HOME", tempDir)

	dbPath := filepath.Join(tempDir, "ticket.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	api := New(config.Config{})
	task, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "task",
		Title:     "Unassigned local task",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if strings.TrimSpace(task.Assignee) != "" {
		t.Fatalf("CreateTask().Assignee = %q, want unassigned", task.Assignee)
	}

	updated, err := api.UpdateTask(task.ID, TaskUpdateRequest{
		Title:       task.Title,
		Description: task.Description,
		ParentID:    task.ParentID,
		Assignee:    task.Assignee,
		Stage:       "done",
		State:       "complete",
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if updated.Status != "done/complete" {
		t.Fatalf("UpdateTask().Status = %q, want done/complete", updated.Status)
	}
}

func TestLocalModeClientDeleteTask(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TICKET_MODE", "local")
	t.Setenv("TICKET_HOME", tempDir)

	dbPath := filepath.Join(tempDir, "ticket.db")
	if err := store.Init(dbPath, "admin", "secret"); err != nil {
		t.Fatalf("store.Init() error = %v", err)
	}

	api := New(config.Config{})
	task, err := api.CreateTask(TaskCreateRequest{
		ProjectID: 1,
		Type:      "task",
		Title:     "Delete me",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := api.DeleteTask(task.ID); err != nil {
		t.Fatalf("DeleteTask() error = %v", err)
	}
	if _, err := api.GetTask(task.ID); !errors.Is(err, store.ErrTaskNotFound) {
		t.Fatalf("GetTask(deleted) error = %v, want ErrTaskNotFound", err)
	}
}
